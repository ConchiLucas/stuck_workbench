package plan

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/conchi/study-learning/logiccontent"
	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
)

var (
	ErrChildNotFound  = errors.New("child not found")
	ErrPlanNotFound   = errors.New("plan not found")
	ErrNoQuestions    = errors.New("no logic questions")
	ErrInvalidMode    = errors.New("invalid plan mode")
	ErrModuleNotFound = errors.New("logic module not found")
)

type CreateInput struct {
	Mode       string   `json:"mode"`
	ModuleCode string   `json:"moduleCode"`
	Count      int      `json:"count"`
	Types      []string `json:"types"`
}

type StudyPlan struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	ChildID      int64      `json:"-"`
	PlanDate     string     `gorm:"type:date" json:"planDate"`
	SeqNo        int        `json:"seqNo"`
	SubjectCode  string     `json:"subjectCode"`
	Status       string     `json:"status"`
	TargetCount  int        `json:"targetCount"`
	DoneCount    int        `json:"doneCount"`
	CorrectCount int        `json:"correctCount"`
	Stars        int        `json:"stars"`
	DurationSec  int        `json:"durationSec"`
	CreatedAt    time.Time  `json:"createdAt"`
	StartedAt    *time.Time `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

func (StudyPlan) TableName() string { return "study_plans" }

type Question struct {
	ID      int64           `json:"id"`
	Code    string          `json:"code"`
	Type    string          `json:"type"`
	Stem    string          `json:"stem"`
	Options json.RawMessage `json:"options"`
	Visual  json.RawMessage `json:"visual"`
	Speech  json.RawMessage `json:"speech"`
}

type Item struct {
	ID          int64                       `json:"id"`
	Seq         int                         `json:"seq"`
	KpID        int64                       `json:"kpId"`
	Title       string                       `json:"title"`
	Bucket      string                       `json:"bucket"`
	Status      string                       `json:"status"`
	Tries       int                         `json:"tries"`
	Picks       string                       `json:"picks"`
	OptionOrder string                       `json:"optionOrder"`
	Question    Question                     `json:"question"`
	Example     *logiccontent.LogicExample `json:"example,omitempty"`
}

type Detail struct {
	Plan  StudyPlan `json:"plan"`
	Items []Item    `json:"items"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB, _ ...string) *Service { return &Service{db: db} }

type questionCandidate struct {
	ID, KpID                                 int64
	Code, Stem, Options, Answer, Visual, Payload, Title string
}

func (s *Service) Create(ctx context.Context, childID int64, input CreateInput) (Detail, error) {
	count := input.Count
	if count <= 0 {
		count = 8
	}
	if count > 24 {
		count = 24
	}
	var childCount int64
	if err := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Count(&childCount).Error; err != nil {
		return Detail{}, err
	}
	if childCount == 0 {
		return Detail{}, ErrChildNotFound
	}
	mode := strings.TrimSpace(input.Mode)
	if mode == "" {
		mode = "daily"
	}
	if mode != "daily" && mode != "module" && mode != "review" {
		return Detail{}, ErrInvalidMode
	}
	kinds := planKinds(input.Types)
	row := StudyPlan{ChildID: childID, PlanDate: time.Now().Format("2006-01-02"), SubjectCode: "logic", Status: "pending", CreatedAt: time.Now()}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []questionCandidate
		query := tx.Table("questions q").
			Select("q.id, q.kp_id, q.code, q.stem, q.options, q.answer, q.visual, COALESCE(kp.payload,'') AS payload, kp.title").
			Joins("JOIN knowledge_points kp ON kp.id = q.kp_id").
			Joins("JOIN modules m ON m.id = kp.module_id").
			Joins("JOIN subjects s ON s.id = m.subject_id").
			Where("s.code = ? AND q.code IN ?", "logic", kinds)
		if mode == "module" && strings.TrimSpace(input.ModuleCode) != "" {
			query = query.Where("m.code = ? OR q.code = ?", strings.TrimSpace(input.ModuleCode), strings.TrimSpace(input.ModuleCode))
		}
		if err := query.Order("m.order_no, kp.order_no, q.id").Scan(&candidates).Error; err != nil {
			return err
		}
		if len(candidates) == 0 {
			return ErrNoQuestions
		}
		candidates = pickByKinds(candidates, kinds, count)
		if len(candidates) == 0 {
			return ErrNoQuestions
		}
		row.TargetCount = len(candidates)
		if err := tx.Table("study_plans").Where("child_id = ? AND plan_date = ?", childID, row.PlanDate).
			Select("COALESCE(MAX(seq_no), 0)").Scan(&row.SeqNo).Error; err != nil {
			return err
		}
		row.SeqNo++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for index, candidate := range candidates {
			example, err := logiccontent.ExampleFromPayload(candidate.Payload, candidate.Visual, "", candidate.KpID)
			if err != nil {
				return err
			}
			example = logiccontent.ShuffleExample(example, row.ID+candidate.ID)
			snap := logiccontent.ExampleToSnapshot(example)
			if err := logiccontent.FreezeMedia(ctx, tx, "", &snap); err != nil {
				return err
			}
			example.ImageURLs = snap.ImageURLs
			raw, err := logiccontent.HistoryBytes(snap)
			if err != nil {
				return err
			}
			order := strings.Join(example.Options, ",")
			if example.Kind == "order" {
				order = strings.Join(example.DisplayOrder, ",")
			}
			item := learningmodel.PlanItem{
				PlanID: row.ID, Seq: index + 1, KpID: candidate.KpID, QuestionID: candidate.ID, Bucket: "new", Status: "pending",
				OptionOrder: order, ContentSnapshotVersion: 1, QuestionSnapshot: string(raw),
				QuestionStem: example.Prompt, QuestionOptions: candidate.Options, QuestionAnswer: candidate.Answer,
				QuestionVisual: candidate.Visual, Explanation: example.Rule.Explain,
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

func (s *Service) Get(ctx context.Context, childID, planID int64) (Detail, error) {
	var row StudyPlan
	err := s.db.WithContext(ctx).Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "logic").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Detail{}, ErrPlanNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	type itemRow struct {
		ID, KpID, QuestionID               int64
		Seq, Tries                       int
		Bucket, Status, Picks, OptionOrder, Title, Code, Stem, Options, Visual, Speech, QuestionSnapshot string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("plan_items pi").
		Select(`pi.id, pi.kp_id, pi.question_id, pi.seq, pi.tries, pi.bucket, pi.status, pi.picks, pi.option_order, pi.question_snapshot,
			kp.title, q.code, q.stem, q.options, q.visual, q.speech`).
		Joins("JOIN questions q ON q.id = pi.question_id").
		Joins("JOIN knowledge_points kp ON kp.id = pi.kp_id").
		Where("pi.plan_id = ?", row.ID).Order("pi.seq").Scan(&rows).Error; err != nil {
		return Detail{}, err
	}
	items := make([]Item, 0, len(rows))
	for _, value := range rows {
		item := Item{ID: value.ID, Seq: value.Seq, KpID: value.KpID, Title: value.Title,
			Bucket: value.Bucket, Status: value.Status, Tries: value.Tries, Picks: value.Picks, OptionOrder: value.OptionOrder,
			Question: Question{ID: value.QuestionID, Code: value.Code, Type: value.Code, Stem: value.Stem,
				Options: json.RawMessage(orJSON(value.Options, "[]")), Visual: json.RawMessage(orJSON(value.Visual, "{}")), Speech: json.RawMessage(orJSON(value.Speech, "{}"))}}
		if converted, err := logiccontent.PlanExampleFromSnapshot(value.QuestionSnapshot, ""); err == nil {
			ex := converted.Example
			item.Example = &ex
			item.Question.Stem = ex.Prompt
			item.Question.Code = ex.Kind
		}
		items = append(items, item)
	}
	return Detail{Plan: row, Items: items}, nil
}

func (s *Service) Start(ctx context.Context, childID, planID int64) (Detail, error) {
	result := s.db.WithContext(ctx).Model(&StudyPlan{}).
		Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "logic").
		Where("status = ?", "pending").Updates(map[string]any{"status": "active", "started_at": time.Now()})
	if result.Error != nil {
		return Detail{}, result.Error
	}
	return s.Get(ctx, childID, planID)
}

func (s *Service) FrozenMedia(ctx context.Context, file string) ([]byte, string, error) {
	return logiccontent.MediaBytes(s.db.WithContext(ctx), file)
}

func planKinds(types []string) []string {
	if len(types) == 0 {
		return append([]string{}, logiccontent.SkillCodes...)
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(types))
	for _, raw := range types {
		if logiccontent.KindNames[raw] == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, raw)
	}
	if len(out) == 0 {
		return append([]string{}, logiccontent.SkillCodes...)
	}
	return out
}

func pickByKinds(candidates []questionCandidate, kinds []string, count int) []questionCandidate {
	pools := map[string][]questionCandidate{}
	for _, row := range candidates {
		pools[row.Code] = append(pools[row.Code], row)
	}
	out := make([]questionCandidate, 0, count)
	used := map[int64]bool{}
	for len(out) < count {
		added := false
		for _, kind := range kinds {
			if len(out) >= count {
				break
			}
			for _, row := range pools[kind] {
				if used[row.ID] {
					continue
				}
				used[row.ID] = true
				out = append(out, row)
				added = true
				break
			}
		}
		if !added {
			break
		}
	}
	return out
}

func orJSON(value, fallback string) string {
	if !json.Valid([]byte(value)) {
		return fallback
	}
	return value
}
