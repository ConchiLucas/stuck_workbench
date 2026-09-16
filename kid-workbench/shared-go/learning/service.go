package learning

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
)

type Service struct {
	cfg mastery.Config
}

func NewService(cfg mastery.Config) *Service {
	return &Service{cfg: cfg}
}

type AttemptInput struct {
	// Assisted preserves the observed result without treating it as independent mastery evidence.
	Assisted   bool
	ClientID   string
	KpID       int64
	QuestionID *int64
	PlanItemID *int64
	Selected   string
	SkillCode  string
	IsCorrect  bool
	CostMs     int
	Source     string
	At         time.Time
}

type StateDTO struct {
	NewlyMastered bool       `json:"newly_mastered,omitempty"`
	KpID          int64      `json:"kp_id"`
	Status        string     `json:"status"`
	Streak        int        `json:"streak"`
	Attempts      int        `json:"attempts"`
	Accuracy      float64    `json:"accuracy"`
	IntervalDays  int        `json:"interval_days"`
	DueAt         *time.Time `json:"due_at"`
}

// ApplyOne records one attempt and all derived learning state in the caller's transaction.
// applied is false when child_id + client_id has already been processed.
func (s *Service) ApplyOne(tx *gorm.DB, childID int64, in AttemptInput) (StateDTO, bool, error) {
	if in.At.IsZero() {
		in.At = time.Now()
	}
	if in.Source == "" {
		in.Source = mastery.SourceQuiz
	}

	// children.id exists before any per-KP mastery row, so it is the stable
	// cross-service lock for all learning-state writes for one child.
	var child model.Child
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&child, childID).Error; err != nil {
		return StateDTO{}, false, err
	}

	var kp model.KnowledgePoint
	if err := tx.First(&kp, in.KpID).Error; err != nil {
		return StateDTO{}, false, err
	}

	attempt := model.Attempt{
		ChildID: childID, KpID: in.KpID, QuestionID: in.QuestionID,
		IsCorrect: in.IsCorrect, CostMs: in.CostMs, Source: in.Source,
		ClientID: in.ClientID, CreatedAt: in.At,
	}
	if in.PlanItemID != nil && tx.Migrator().HasColumn("attempts", "plan_item_id") {
		attempt.PlanItemID = in.PlanItemID
	}
	if tx.Migrator().HasColumn("attempts", "selected") {
		attempt.Selected = strings.TrimSpace(in.Selected)
	}
	created := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "child_id"}, {Name: "client_id"}},
		DoNothing: true,
	}).Create(&attempt)
	if created.Error != nil {
		if isUniqueViolation(created.Error) {
			return StateDTO{}, false, nil
		}
		return StateDTO{}, false, created.Error
	}
	if created.RowsAffected == 0 {
		return StateDTO{}, false, nil
	}

	scope, err := scopeForKp(tx, in.KpID)
	if err != nil {
		return StateDTO{}, false, err
	}

	skillSet := mastery.SkillsFor(scope.SubjectCode, scope.ModuleCode)
	skillCode := strings.TrimSpace(in.SkillCode)
	if skillCode != "" {
		skillCode = mastery.SkillFromQuestionCode(scope.SubjectCode, skillCode)
		if !mastery.SkillIsTracked(scope.SubjectCode, scope.ModuleCode, skillCode) {
			return StateDTO{}, false, errors.New("skill code is not valid for knowledge point module")
		}
	} else if in.QuestionID != nil && *in.QuestionID > 0 {
		var question model.Question
		if err := tx.First(&question, *in.QuestionID).Error; err == nil {
			skillCode = mastery.SkillFromQuestionCode(scope.SubjectCode, question.Code)
		}
	}

	if scope.SubjectCode == "pinyin" && skillCode != "" {
		valid := false
		for _, required := range skillSet {
			valid = valid || required == skillCode
		}
		if !valid {
			return StateDTO{}, false, errors.New("question skill does not apply to this pinyin module")
		}
	}
	if scope.SubjectCode == "pinyin" && (skillCode == "" || in.Source == mastery.SourceParentMark) {
		// Legacy unclassified facts remain readable but cannot award mastery.
		in.Assisted = true
	}
	before, err := getMastery(tx, childID, in.KpID)
	if err != nil {
		return StateDTO{}, false, err
	}
	beforeEngine := stateToEngine(before)

	var after mastery.State
	if in.Assisted {
		after = beforeEngine
	} else if len(skillSet) > 0 && (skillCode != "" || in.Source == mastery.SourceParentMark) {
		if err := s.applySubjectSkills(tx, childID, in, kp.Difficulty, skillCode, skillSet); err != nil {
			return StateDTO{}, false, err
		}
		after, err = s.rollupSubjectMastery(tx, childID, in.KpID, skillSet)
		if err != nil {
			return StateDTO{}, false, err
		}
	} else {
		after = mastery.Apply(beforeEngine, mastery.Attempt{
			Correct: in.IsCorrect, At: in.At, Source: in.Source,
		}, kp.Difficulty, s.cfg)
		next := stateFromEngine(childID, in.KpID, after)
		if err := upsertMastery(tx, &next); err != nil {
			return StateDTO{}, false, err
		}
	}

	newlyMastered := 0
	if beforeEngine.MasteredAt == nil && after.MasteredAt != nil {
		newlyMastered = 1
	}
	if newlyMastered == 0 &&
		mastery.Display(beforeEngine, in.At) != mastery.StatusMastered &&
		mastery.Display(after, in.At) == mastery.StatusMastered {
		newlyMastered = 1
	}
	if scope.SubjectCode == "pinyin" && !in.Assisted {
		completed, e := finalizePinyin(tx, childID, in.KpID, &after, in.At)
		if e != nil {
			return StateDTO{}, false, e
		}
		newlyMastered = boolToInt(completed)
	}

	formalMath := scope.SubjectCode == "math" && !in.Assisted && in.Source != mastery.SourceParentMark && len(skillSet) > 0 && skillCode != ""
	if formalMath {
		completed, e := finalizeMath(tx, childID, in.KpID, beforeEngine, &after, in.At)
		if e != nil {
			return StateDTO{}, false, e
		}
		newlyMastered = boolToInt(completed)
	}

	reviewDone := 0
	if beforeEngine.Status == mastery.StatusMastered && in.IsCorrect && !in.Assisted {
		reviewDone = 1
	}

	if in.Source != mastery.SourceParentMark {
		if err := bumpDailyStat(tx, childID, in.At, model.DailyStat{
			PracticeSec: maxInt(in.CostMs/1000, 1), Attempts: 1,
			Correct: boolToInt(in.IsCorrect), NewlyMastered: newlyMastered,
			ReviewDone: reviewDone,
		}); err != nil {
			return StateDTO{}, false, err
		}
	}

	grantReward := newlyMastered == 1
	if grantReward && scope.SubjectCode == "pinyin" {
		exists, e := pinyinRewardExists(tx, childID, in.KpID)
		if e != nil {
			return StateDTO{}, false, e
		}
		grantReward = !exists
	}
	if grantReward {
		kpID := in.KpID
		var err error
		if formalMath {
			err = addFlowersAt(tx, childID, 1, "mastered", "knowledge_point", &kpID, in.At)
		} else {
			err = addFlowers(tx, childID, 1, "mastered", "knowledge_point", &kpID)
		}
		if err != nil {
			return StateDTO{}, false, err
		}
	}

	result := stateDTO(in.KpID, after)
	result.NewlyMastered = newlyMastered == 1
	return result, true, nil
}

func (s *Service) applySubjectSkills(
	tx *gorm.DB, childID int64, in AttemptInput, difficulty int, skillCode string, skillSet []string,
) error {
	codes := []string{skillCode}
	if in.Source == mastery.SourceParentMark {
		codes = append([]string{}, skillSet...)
	}
	for _, code := range codes {
		if code == "" {
			continue
		}
		current, err := getMasterySkill(tx, childID, in.KpID, code)
		if err != nil {
			return err
		}
		after := mastery.Apply(skillToEngine(current), mastery.Attempt{
			Correct: in.IsCorrect, At: in.At, Source: in.Source,
		}, difficulty, s.cfg)
		next := skillFromEngine(childID, in.KpID, code, after)
		if err := upsertMasterySkill(tx, &next); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) rollupSubjectMastery(
	tx *gorm.DB, childID, kpID int64, skillSet []string,
) (mastery.State, error) {
	now := time.Now()
	statuses := make([]mastery.Status, 0, len(skillSet))
	best := mastery.State{}
	anyMasteredAt := false
	for _, code := range skillSet {
		row, err := getMasterySkill(tx, childID, kpID, code)
		if err != nil {
			return mastery.State{}, err
		}
		engine := skillToEngine(row)
		statuses = append(statuses, mastery.Display(engine, now))
		if engine.Attempts > best.Attempts {
			best = engine
		}
		if engine.MasteredAt != nil {
			anyMasteredAt = true
		}
	}

	rolled := mastery.RollupSkills(statuses)
	out := best
	if out.Ease == 0 {
		out = mastery.NewState()
	}
	out.Status = rolled
	if rolled == mastery.StatusMastered || rolled == mastery.StatusReviewDue {
		if !anyMasteredAt && out.MasteredAt == nil {
			at := now
			out.MasteredAt = &at
		}
	} else {
		out.MasteredAt = nil
	}

	next := stateFromEngine(childID, kpID, out)
	next.Status = string(rolled)
	if err := upsertMastery(tx, &next); err != nil {
		return mastery.State{}, err
	}
	return out, nil
}

func getMastery(tx *gorm.DB, childID, kpID int64) (model.MasteryState, error) {
	var state model.MasteryState
	err := tx.Where("child_id = ? AND kp_id = ?", childID, kpID).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.MasteryState{
			ChildID: childID, KpID: kpID, Status: string(mastery.StatusNotStarted), Ease: 2.5,
		}, nil
	}
	return state, err
}

func upsertMastery(tx *gorm.DB, state *model.MasteryState) error {
	state.UpdatedAt = time.Now()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "child_id"}, {Name: "kp_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "attempts", "correct", "streak", "best_streak", "ease",
			"interval_days", "due_at", "first_seen_at", "mastered_at", "updated_at",
		}),
	}).Create(state).Error
}

func getMasterySkill(tx *gorm.DB, childID, kpID int64, skillCode string) (model.MasterySkill, error) {
	var state model.MasterySkill
	err := tx.Where("child_id = ? AND kp_id = ? AND skill_code = ?", childID, kpID, skillCode).
		First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.MasterySkill{
			ChildID: childID, KpID: kpID, SkillCode: skillCode,
			Status: string(mastery.StatusNotStarted), Ease: 2.5,
		}, nil
	}
	return state, err
}

func upsertMasterySkill(tx *gorm.DB, state *model.MasterySkill) error {
	state.UpdatedAt = time.Now()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "child_id"}, {Name: "kp_id"}, {Name: "skill_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "attempts", "correct", "streak", "best_streak", "ease",
			"interval_days", "due_at", "first_seen_at", "mastered_at", "updated_at",
		}),
	}).Create(state).Error
}

type kpScope struct {
	SubjectCode string
	ModuleCode  string
}

func scopeForKp(tx *gorm.DB, kpID int64) (kpScope, error) {
	var scope kpScope
	err := tx.Raw(`
		SELECT s.code AS subject_code, m.code AS module_code FROM subjects s
		JOIN modules m ON m.subject_id = s.id
		JOIN knowledge_points kp ON kp.module_id = m.id
		WHERE kp.id = ?`, kpID).Scan(&scope).Error
	return scope, err
}

func bumpDailyStat(tx *gorm.DB, childID int64, at time.Time, delta model.DailyStat) error {
	return tx.Exec(`
		INSERT INTO daily_stats
			(child_id, stat_date, practice_sec, attempts, correct, newly_mastered, review_done, checked_in)
		VALUES (?, ?, ?, ?, ?, ?, ?, TRUE)
		ON CONFLICT(child_id, stat_date) DO UPDATE SET
			practice_sec = daily_stats.practice_sec + EXCLUDED.practice_sec,
			attempts = daily_stats.attempts + EXCLUDED.attempts,
			correct = daily_stats.correct + EXCLUDED.correct,
			newly_mastered = daily_stats.newly_mastered + EXCLUDED.newly_mastered,
			review_done = daily_stats.review_done + EXCLUDED.review_done,
			checked_in = TRUE`,
		childID, at.Format("2006-01-02"), delta.PracticeSec, delta.Attempts,
		delta.Correct, delta.NewlyMastered, delta.ReviewDone,
	).Error
}

func addFlowers(tx *gorm.DB, childID int64, delta int, reason, refType string, refID *int64) error {
	return addFlowersAt(tx, childID, delta, reason, refType, refID, time.Now())
}

func addFlowersAt(tx *gorm.DB, childID int64, delta int, reason, refType string, refID *int64, at time.Time) error {
	if err := tx.Create(&model.FlowerLedger{
		ChildID: childID, Delta: delta, Reason: reason,
		RefType: refType, RefID: refID, CreatedAt: at,
	}).Error; err != nil {
		return err
	}
	return tx.Model(&model.Child{}).Where("id = ?", childID).
		UpdateColumn("flowers", gorm.Expr("flowers + ?", delta)).Error
}

func stateToEngine(state model.MasteryState) mastery.State {
	out := mastery.State{
		Status: mastery.Status(state.Status), Attempts: state.Attempts, Correct: state.Correct,
		Streak: state.Streak, BestStreak: state.BestStreak, Ease: state.Ease,
		IntervalDays: state.IntervalDays, MasteredAt: state.MasteredAt,
	}
	if out.Ease == 0 {
		out.Ease = 2.5
	}
	if state.DueAt != nil {
		out.DueAt = *state.DueAt
	}
	if state.FirstSeenAt != nil {
		out.FirstSeenAt = *state.FirstSeenAt
	}
	return out
}

func stateFromEngine(childID, kpID int64, state mastery.State) model.MasteryState {
	out := model.MasteryState{
		ChildID: childID, KpID: kpID, Status: string(state.Status),
		Attempts: state.Attempts, Correct: state.Correct, Streak: state.Streak,
		BestStreak: state.BestStreak, Ease: state.Ease, IntervalDays: state.IntervalDays,
		MasteredAt: state.MasteredAt,
	}
	if !state.DueAt.IsZero() {
		at := state.DueAt
		out.DueAt = &at
	}
	if !state.FirstSeenAt.IsZero() {
		at := state.FirstSeenAt
		out.FirstSeenAt = &at
	}
	return out
}

func skillToEngine(state model.MasterySkill) mastery.State {
	out := mastery.State{
		Status: mastery.Status(state.Status), Attempts: state.Attempts, Correct: state.Correct,
		Streak: state.Streak, BestStreak: state.BestStreak, Ease: state.Ease,
		IntervalDays: state.IntervalDays, MasteredAt: state.MasteredAt,
	}
	if out.Ease == 0 {
		out.Ease = 2.5
	}
	if state.Status == "" || state.Status == string(mastery.StatusNotStarted) {
		out.Status = mastery.StatusNotStarted
	}
	if state.DueAt != nil {
		out.DueAt = *state.DueAt
	}
	if state.FirstSeenAt != nil {
		out.FirstSeenAt = *state.FirstSeenAt
	}
	return out
}

func skillFromEngine(childID, kpID int64, skillCode string, state mastery.State) model.MasterySkill {
	out := model.MasterySkill{
		ChildID: childID, KpID: kpID, SkillCode: skillCode, Status: string(state.Status),
		Attempts: state.Attempts, Correct: state.Correct, Streak: state.Streak,
		BestStreak: state.BestStreak, Ease: state.Ease, IntervalDays: state.IntervalDays,
		MasteredAt: state.MasteredAt,
	}
	if !state.DueAt.IsZero() {
		at := state.DueAt
		out.DueAt = &at
	}
	if !state.FirstSeenAt.IsZero() {
		at := state.FirstSeenAt
		out.FirstSeenAt = &at
	}
	return out
}

func stateDTO(kpID int64, state mastery.State) StateDTO {
	out := StateDTO{
		KpID: kpID, Status: string(mastery.Display(state, time.Now())),
		Streak: state.Streak, Attempts: state.Attempts, Accuracy: state.Accuracy(),
		IntervalDays: state.IntervalDays,
	}
	if !state.DueAt.IsZero() {
		at := state.DueAt
		out.DueAt = &at
	}
	return out
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "UNIQUE constraint failed") ||
		strings.Contains(message, "constraint failed: UNIQUE") ||
		strings.Contains(message, "duplicate key value violates unique constraint")
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
