package practice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/conchi/chengyu-server/internal/plan"
	"github.com/conchi/chengyu-server/internal/quiz"
	"github.com/conchi/study-learning/chengyucontent"
	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPlanNotFound        = errors.New("plan not found")
	ErrItemNotFound        = errors.New("item not found")
	ErrItemCompleted       = errors.New("item completed")
	ErrPlanCompleted       = errors.New("plan completed")
	ErrPlanIncomplete      = errors.New("plan incomplete")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)

type AnswerInput struct {
	ClientID    string `json:"clientId"`
	OptionIndex int    `json:"optionIndex"`
	CostMs      int    `json:"costMs"`
}

type AnswerResult struct {
	Correct     bool              `json:"correct"`
	AnswerIndex *int              `json:"answerIndex,omitempty"`
	CanRetry    bool              `json:"canRetry"`
	Tries       int               `json:"tries"`
	Status      string            `json:"status"`
	Mastery     learning.StateDTO `json:"mastery"`
}

type WeakChengyu struct {
	KpID    int64  `json:"kpId"`
	Chengyu string `json:"chengyu"`
	Meaning string `json:"meaning"`
}

type FinishResult struct {
	Plan        plan.StudyPlan `json:"plan"`
	Stars       int            `json:"stars"`
	Flowers     int            `json:"flowers"`
	WeakChengyu []WeakChengyu  `json:"weakChengyu"`
}

type Service struct {
	db       *gorm.DB
	learning *learning.Service
}

func NewService(db *gorm.DB, cfg mastery.Config) *Service {
	return &Service{db, learning.NewService(cfg)}
}

func (s *Service) Answer(ctx context.Context, childID, planID, itemID int64, input AnswerInput) (AnswerResult, error) {
	if strings.TrimSpace(input.ClientID) == "" {
		return AnswerResult{}, errors.New("client id required")
	}
	var out AnswerResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p plan.StudyPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "chengyu").First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanNotFound
			}
			return err
		}
		if p.Status == "done" {
			return ErrPlanCompleted
		}
		var item learningmodel.PlanItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND plan_id=?", itemID, planID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		answerRaw, optionsRaw := item.QuestionAnswer, item.QuestionOptions
		if answerRaw == "" || optionsRaw == "" {
			var q struct{ Answer, Options string }
			if err := tx.Table("questions").Where("id=?", item.QuestionID).Take(&q).Error; err != nil {
				return err
			}
			answerRaw, optionsRaw = q.Answer, q.Options
		}
		order, err := plan.ParseOrder(item.OptionOrder)
		if err != nil {
			return err
		}
		if len(order) == 0 {
			order = identityOrder(optionCount(optionsRaw))
		}
		if input.OptionIndex < 0 || input.OptionIndex >= len(order) {
			return errors.New("option index out of range")
		}
		original, err := parseAnswer(answerRaw)
		if err != nil {
			return err
		}
		correct := order[input.OptionIndex] == original
		selectedID, err := chengyucontent.OptionIDForDisplayIndex(optionsRaw, item.OptionOrder, input.OptionIndex)
		if err != nil {
			return err
		}
		var existing learningmodel.Attempt
		if err := tx.Where("child_id=? AND client_id=?", childID, input.ClientID).First(&existing).Error; err == nil {
			if existing.KpID != item.KpID || existing.QuestionID == nil || *existing.QuestionID != item.QuestionID {
				return ErrIdempotencyConflict
			}
			out = resultFor(existing.IsCorrect, item)
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		firstTry := item.Status == "pending"
		if !firstTry && (item.Status != "wrong" || item.Tries >= 2) {
			return ErrItemCompleted
		}
		state, applied, err := s.learning.ApplyOne(tx, childID, learning.AttemptInput{
			ClientID: input.ClientID, KpID: item.KpID, QuestionID: &item.QuestionID, PlanItemID: &item.ID,
			Selected: selectedID, SkillCode: skillCode(item), IsCorrect: correct, CostMs: input.CostMs, Source: mastery.SourceQuiz, At: time.Now(),
		})
		if err != nil {
			return err
		}
		if !applied {
			return errors.New("attempt not applied")
		}
		tries := item.Tries + 1
		status := "wrong"
		if correct {
			status = "correct"
		}
		updates := map[string]any{"tries": tries, "cost_ms": item.CostMs + input.CostMs, "picks": selectedID, "status": status}
		if firstTry {
			now := time.Now()
			updates["answered_at"] = &now
		}
		if err := tx.Model(&learningmodel.PlanItem{}).Where("id=?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		pu := map[string]any{"status": "doing"}
		if p.StartedAt == nil {
			pu["started_at"] = time.Now()
		}
		if firstTry {
			pu["done_count"] = gorm.Expr("done_count+1")
		}
		if correct && (firstTry || item.Status != "correct") {
			pu["correct_count"] = gorm.Expr("correct_count+1")
		}
		if err := tx.Model(&plan.StudyPlan{}).Where("id=?", planID).Updates(pu).Error; err != nil {
			return err
		}
		out = AnswerResult{Correct: correct, CanRetry: !correct && tries < 2, Tries: tries, Status: status, Mastery: state}
		return nil
	})
	return out, err
}

func resultFor(correct bool, item learningmodel.PlanItem) AnswerResult {
	return AnswerResult{Correct: correct, CanRetry: !correct && item.Tries < 2 && item.Status != "correct", Tries: item.Tries, Status: item.Status}
}

func (s *Service) Finish(ctx context.Context, childID, planID int64) (FinishResult, error) {
	var out FinishResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p plan.StudyPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "chengyu").First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanNotFound
			}
			return err
		}
		if p.Status != "done" {
			if p.DoneCount < p.TargetCount {
				return ErrPlanIncomplete
			}
			stars := 1
			if p.TargetCount > 0 && p.CorrectCount*100/p.TargetCount >= 80 {
				stars = 3
			} else if p.TargetCount > 0 && p.CorrectCount*100/p.TargetCount >= 60 {
				stars = 2
			}
			now := time.Now()
			if err := tx.Model(&p).Updates(map[string]any{"status": "done", "completed_at": &now, "stars": stars}).Error; err != nil {
				return err
			}
			p.Status = "done"
			p.CompletedAt = &now
			p.Stars = stars
		}
		type weakRow struct {
			KpID             int64
			Chengyu, Payload string
		}
		var rows []weakRow
		if err := tx.Table("plan_items pi").Select("DISTINCT pi.kp_id,kp.title AS chengyu,kp.payload").Joins("JOIN knowledge_points kp ON kp.id=pi.kp_id").Where("pi.plan_id=? AND pi.status=?", planID, "wrong").Scan(&rows).Error; err != nil {
			return err
		}
		weak := make([]WeakChengyu, 0, len(rows))
		for _, r := range rows {
			payload, _ := quiz.ParsePayload(r.Payload)
			weak = append(weak, WeakChengyu{r.KpID, r.Chengyu, payload.Meaning})
		}
		var flowers int64
		q := tx.Table("flower_ledger fl").Joins("JOIN plan_items pi ON pi.kp_id=fl.ref_id").Where("pi.plan_id=? AND fl.child_id=? AND fl.ref_type='knowledge_point'", planID, childID).Select("COALESCE(SUM(DISTINCT fl.delta),0)").Scan(&flowers)
		if q.Error != nil {
			return q.Error
		}
		out = FinishResult{p, p.Stars, int(flowers), weak}
		return nil
	})
	return out, err
}

func optionCount(raw string) int {
	var v []json.RawMessage
	if json.Unmarshal([]byte(raw), &v) != nil {
		return 0
	}
	return len(v)
}

func identityOrder(n int) []int {
	o := make([]int, n)
	for i := range o {
		o[i] = i
	}
	return o
}

func parseAnswer(raw string) (int, error) {
	var v struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return 0, fmt.Errorf("invalid answer: %w", err)
	}
	return v.Index, nil
}

func skillCode(item learningmodel.PlanItem) string {
	converted, err := chengyucontent.PlanExampleFromSnapshot(item.QuestionSnapshot, "")
	if err == nil && converted.Example.Kind != "" {
		return converted.Example.Kind
	}
	var snap struct{ Kind, SkillCode, Code string }
	if json.Unmarshal([]byte(item.QuestionSnapshot), &snap) == nil {
		if snap.SkillCode != "" {
			return snap.SkillCode
		}
		if snap.Kind != "" {
			return snap.Kind
		}
		if snap.Code != "" {
			return snap.Code
		}
	}
	return ""
}
