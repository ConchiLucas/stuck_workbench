package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/catalog"
)

var (
	ErrChildNotFound = errors.New("child not found")
	ErrPlanNotFound  = errors.New("plan not found")
	ErrNoQuestions   = errors.New("no math questions")
	ErrInvalidScope  = errors.New("invalid plan scope")
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Create(ctx context.Context, childID int64, input CreateInput) (Detail, error) {
	if err := validateScope(input); err != nil {
		return Detail{}, err
	}
	var childCount int64
	if err := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Count(&childCount).Error; err != nil {
		return Detail{}, err
	}
	if childCount == 0 {
		return Detail{}, ErrChildNotFound
	}
	date := time.Now().Format("2006-01-02")
	if input.Kind == "daily" {
		var existing StudyPlan
		err := s.db.WithContext(ctx).
			Where("child_id = ? AND plan_date = ? AND subject_code = ? AND plan_kind = ?", childID, date, "math", "daily").
			Order("id DESC").First(&existing).Error
		if err == nil {
			return s.Get(ctx, childID, existing.ID)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return Detail{}, err
		}
	}

	candidates, err := s.loadCandidates(ctx, childID, input)
	if err != nil {
		return Detail{}, err
	}
	selected := selectCandidates(candidates, input)
	if len(selected) == 0 {
		return Detail{}, ErrNoQuestions
	}

	row := StudyPlan{
		ChildID: childID, PlanDate: date, SubjectCode: "math", PlanKind: input.Kind,
		ModuleCode: input.ModuleCode, StageCode: input.StageCode,
		Status: "pending", TargetCount: len(selected), CreatedAt: time.Now(),
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("study_plans").Where("child_id = ? AND plan_date = ?", childID, date).
			Select("COALESCE(MAX(seq_no), 0)").Scan(&row.SeqNo).Error; err != nil {
			return err
		}
		row.SeqNo++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for index, candidate := range selected {
			snapshot, err := BuildSnapshot(candidate)
			if err != nil {
				return err
			}
			rawSnapshot, err := json.Marshal(snapshot)
			if err != nil {
				return err
			}
			item := learningmodel.PlanItem{
				PlanID: row.ID, Seq: index + 1, KpID: candidate.KpID, QuestionID: candidate.QuestionID,
				Bucket: candidate.Bucket, Status: "pending", QuestionStem: candidate.Stem,
				QuestionOptions: candidate.Options, QuestionAnswer: candidate.Answer,
				QuestionVisual: candidate.Visual, QuestionSnapshot: string(rawSnapshot), ContentSnapshotVersion: 1,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, childID, row.ID)
}

func validateScope(input CreateInput) error {
	switch input.Kind {
	case "daily", "review":
		if input.ModuleCode != "" || input.StageCode != "" {
			return ErrInvalidScope
		}
		return nil
	case "module":
		if input.ModuleCode == "" || input.StageCode == "" {
			return ErrInvalidScope
		}
		valid := false
		if input.ModuleCode == "shape" {
			valid = input.StageCode == "basic-shapes"
		} else if input.ModuleCode == "add10" || input.ModuleCode == "sub10" {
			valid = input.StageCode == "within5" || input.StageCode == "within10" || input.StageCode == "within20"
		}
		if !valid {
			return ErrInvalidScope
		}
		return nil
	default:
		return ErrInvalidScope
	}
}

func ValidateCreateInput(input CreateInput) error { return validateScope(input) }

type candidateRow struct {
	QuestionID, KpID                                          int64
	ModuleCode, Code, Stem, Options, Answer, Visual, MediaURL string
	Payload, MasteryStatus                                    string
	DueAt                                                     *time.Time
	RecentWrong                                               bool
}

func (s *Service) loadCandidates(ctx context.Context, childID int64, input CreateInput) ([]Candidate, error) {
	var rows []candidateRow
	now := time.Now()
	recentWrongCutoff := now.AddDate(0, 0, -7)
	err := s.db.WithContext(ctx).Table("questions q").
		Select(`q.id AS question_id, q.kp_id, m.code AS module_code, q.code, q.stem,
			q.options, q.answer, q.visual, q.media_url, kp.payload,
			COALESCE(ms.status, 'not_started') AS mastery_status, ms.due_at,
			EXISTS(SELECT 1 FROM attempts a WHERE a.child_id = ? AND a.kp_id = kp.id AND a.is_correct = FALSE AND a.created_at >= ?) AS recent_wrong`, childID, recentWrongCutoff).
		Joins("JOIN knowledge_points kp ON kp.id = q.kp_id").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Joins("LEFT JOIN mastery_states ms ON ms.child_id = ? AND ms.kp_id = kp.id", childID).
		Where("s.code = ? AND m.code IN ?", "math", []string{"add10", "sub10", "shape"}).
		Order("m.order_no, kp.order_no, kp.id, q.id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(rows))
	for _, row := range rows {
		candidate := Candidate{
			QuestionID: row.QuestionID, KpID: row.KpID, ModuleCode: row.ModuleCode,
			Code: row.Code, Stem: row.Stem, Options: row.Options, Answer: row.Answer,
			Visual: row.Visual, MediaURL: row.MediaURL,
		}
		if _, err := BuildSnapshot(candidate); err != nil {
			continue
		}
		if row.ModuleCode != "shape" {
			var payload struct{ A, B int }
			if json.Unmarshal([]byte(row.Payload), &payload) != nil {
				continue
			}
			candidate.A, candidate.B = payload.A, payload.B
		}
		stage := catalog.StageFor(row.ModuleCode, candidate.A, candidate.B)
		if input.Kind == "module" && (row.ModuleCode != input.ModuleCode || stage != input.StageCode) {
			continue
		}
		due := row.DueAt != nil && row.DueAt.Before(now)
		switch {
		case row.MasteryStatus == "shaky":
			candidate.Bucket, candidate.Priority = "shaky", 1
		case row.MasteryStatus == "review_due" || due:
			candidate.Bucket, candidate.Priority = "review_due", 0
		case row.RecentWrong:
			candidate.Bucket, candidate.Priority = "recent_wrong", 1
		case row.MasteryStatus == "learning":
			candidate.Bucket, candidate.Priority = "learning", 2
		default:
			candidate.Bucket, candidate.Priority = "new", 3
		}
		if input.Kind == "review" && candidate.Priority >= 2 {
			continue
		}
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		if out[i].KpID != out[j].KpID {
			return out[i].KpID < out[j].KpID
		}
		return out[i].QuestionID < out[j].QuestionID
	})
	return out, nil
}

func selectCandidates(candidates []Candidate, input CreateInput) []Candidate {
	if input.Kind == "review" || input.Kind == "module" {
		return takeAlternating(candidates, 10)
	}
	quotas := []struct {
		module string
		count  int
	}{{"add10", 4}, {"sub10", 4}, {"shape", 2}}
	selected := make([]Candidate, 0, 10)
	used := map[int64]bool{}
	for _, quota := range quotas {
		var moduleCandidates []Candidate
		for _, candidate := range candidates {
			if candidate.ModuleCode == quota.module {
				moduleCandidates = append(moduleCandidates, candidate)
			}
		}
		for _, candidate := range takeAlternating(moduleCandidates, quota.count) {
			selected = append(selected, candidate)
			used[candidate.QuestionID] = true
		}
	}
	for _, candidate := range candidates {
		if len(selected) >= 10 {
			break
		}
		if !used[candidate.QuestionID] {
			selected = append(selected, candidate)
			used[candidate.QuestionID] = true
		}
	}
	return selected
}

func takeAlternating(candidates []Candidate, limit int) []Candidate {
	if len(candidates) <= limit {
		return append([]Candidate(nil), candidates...)
	}
	byCode := map[string][]Candidate{}
	var codeOrder []string
	for _, candidate := range candidates {
		if _, ok := byCode[candidate.Code]; !ok {
			codeOrder = append(codeOrder, candidate.Code)
		}
		byCode[candidate.Code] = append(byCode[candidate.Code], candidate)
	}
	out := make([]Candidate, 0, limit)
	for len(out) < limit {
		added := false
		for _, code := range codeOrder {
			if len(byCode[code]) == 0 || len(out) >= limit {
				continue
			}
			out = append(out, byCode[code][0])
			byCode[code] = byCode[code][1:]
			added = true
		}
		if !added {
			break
		}
	}
	return out
}

func (s *Service) Get(ctx context.Context, childID, planID int64) (Detail, error) {
	var row StudyPlan
	err := s.db.WithContext(ctx).
		Where("id = ? AND child_id = ? AND subject_code = ? AND plan_kind <> ''", planID, childID, "math").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Detail{}, ErrPlanNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	type itemRow struct {
		ID, KpID                         int64
		Seq, Tries                       int
		Bucket, Status, QuestionSnapshot string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("plan_items").
		Select("id, kp_id, seq, tries, bucket, status, question_snapshot").
		Where("plan_id = ?", planID).Order("seq").Scan(&rows).Error; err != nil {
		return Detail{}, err
	}
	items := make([]Item, 0, len(rows))
	for _, value := range rows {
		var snapshot QuestionSnapshot
		if err := json.Unmarshal([]byte(value.QuestionSnapshot), &snapshot); err != nil {
			return Detail{}, fmt.Errorf("plan item %d snapshot: %w", value.ID, err)
		}
		audioURL := fmt.Sprintf("/api/v1/children/%d/math/plans/%d/items/%d/audio.mp3", childID, planID, value.ID)
		items = append(items, Item{
			ID: value.ID, Seq: value.Seq, KpID: value.KpID, Bucket: value.Bucket,
			Status: value.Status, Tries: value.Tries, Question: PublicQuestion(snapshot, audioURL),
		})
	}
	return Detail{Plan: row, Items: items}, nil
}

func (s *Service) Start(ctx context.Context, childID, planID int64) (Detail, error) {
	result := s.db.WithContext(ctx).Model(&StudyPlan{}).
		Where("id = ? AND child_id = ? AND subject_code = ? AND plan_kind <> ''", planID, childID, "math").
		Where("status = ?", "pending").Updates(map[string]any{"status": "doing", "started_at": time.Now()})
	if result.Error != nil {
		return Detail{}, result.Error
	}
	return s.Get(ctx, childID, planID)
}
