package practice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/conchi/pinyin-server/internal/plan"
)

var (
	ErrPlanNotFound        = errors.New("plan not found")
	ErrItemNotFound        = errors.New("plan item not found")
	ErrItemCompleted       = errors.New("plan item completed")
	ErrPlanIncomplete      = errors.New("plan incomplete")
	ErrIdempotencyConflict = errors.New("client id belongs to another attempt")
)

type AnswerInput struct {
	ClientID    string `json:"clientId" binding:"required"`
	OptionIndex int    `json:"optionIndex" binding:"min=0,max=20"`
	CostMs      int    `json:"costMs" binding:"min=0,max=3600000"`
}

type AnswerResult struct {
	Correct     bool              `json:"correct"`
	AnswerIndex int               `json:"answerIndex"`
	CanRetry    bool              `json:"canRetry"`
	Tries       int               `json:"tries"`
	Status      string            `json:"status"`
	Mastery     learning.StateDTO `json:"mastery"`
}

type Service struct {
	db       *gorm.DB
	learning *learning.Service
}

func NewService(db *gorm.DB, cfg mastery.Config) *Service {
	return &Service{db: db, learning: learning.NewService(cfg)}
}

func (s *Service) Answer(ctx context.Context, childID, planID, itemID int64, input AnswerInput) (AnswerResult, error) {
	if strings.TrimSpace(input.ClientID) == "" {
		return AnswerResult{}, errors.New("client id is required")
	}
	var output AnswerResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var planRow plan.StudyPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "pinyin").First(&planRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanNotFound
			}
			return err
		}
		var item learningmodel.PlanItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND plan_id = ?", itemID, planID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		answerRaw, optionsRaw := item.QuestionAnswer, item.QuestionOptions
		if answerRaw == "" || optionsRaw == "" {
			var question struct{ Answer, Options string }
			if err := tx.Table("questions").Where("id = ?", item.QuestionID).Take(&question).Error; err != nil {
				return err
			}
			answerRaw, optionsRaw = question.Answer, question.Options
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
		correctOriginal, err := parseAnswer(answerRaw)
		if err != nil {
			return err
		}
		answerDisplay := displayIndex(order, correctOriginal)
		isCorrect := order[input.OptionIndex] == correctOriginal

		var existing learningmodel.Attempt
		if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&existing).Error; err == nil {
			if existing.KpID != item.KpID || existing.QuestionID == nil || *existing.QuestionID != item.QuestionID {
				return ErrIdempotencyConflict
			}
			output = AnswerResult{Correct: existing.IsCorrect, AnswerIndex: answerDisplay,
				CanRetry: item.Status == "pending" && !existing.IsCorrect && item.Tries < 2,
				Tries:    item.Tries, Status: item.Status, Mastery: loadMastery(tx, childID, item.KpID)}
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if item.Status != "pending" {
			return ErrItemCompleted
		}

		state, applied, err := s.learning.ApplyOne(tx, childID, learning.AttemptInput{
			ClientID: input.ClientID, KpID: item.KpID, QuestionID: &item.QuestionID,
			IsCorrect: isCorrect, CostMs: input.CostMs, Source: mastery.SourceQuiz, At: time.Now(),
		})
		if err != nil {
			return err
		}
		if !applied {
			if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&existing).Error; err != nil {
				return err
			}
			if existing.KpID != item.KpID || existing.QuestionID == nil || *existing.QuestionID != item.QuestionID {
				return ErrIdempotencyConflict
			}
			output = AnswerResult{Correct: existing.IsCorrect, AnswerIndex: answerDisplay,
				CanRetry: item.Status == "pending" && !existing.IsCorrect && item.Tries < 2,
				Tries:    item.Tries, Status: item.Status, Mastery: loadMastery(tx, childID, item.KpID)}
			return nil
		}
		tries := item.Tries + 1
		status := "pending"
		completedNow := isCorrect || tries >= 2
		updates := map[string]any{
			"tries": tries, "cost_ms": item.CostMs + input.CostMs,
			"picks": appendPick(item.Picks, input.OptionIndex), "status": status,
		}
		if completedNow {
			status = "wrong"
			if isCorrect {
				status = "correct"
			}
			now := time.Now()
			updates["status"] = status
			updates["answered_at"] = &now
		}
		if err := tx.Model(&learningmodel.PlanItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		planUpdates := map[string]any{"status": "doing"}
		if planRow.StartedAt == nil {
			planUpdates["started_at"] = time.Now()
		}
		if completedNow {
			planUpdates["done_count"] = gorm.Expr("done_count + 1")
			if isCorrect {
				planUpdates["correct_count"] = gorm.Expr("correct_count + 1")
			}
		}
		if err := tx.Model(&plan.StudyPlan{}).Where("id = ?", planID).Updates(planUpdates).Error; err != nil {
			return err
		}
		output = AnswerResult{Correct: isCorrect, AnswerIndex: answerDisplay,
			CanRetry: !isCorrect && tries < 2, Tries: tries, Status: status, Mastery: state}
		return nil
	})
	return output, err
}

func (s *Service) Finish(ctx context.Context, childID, planID int64) (plan.StudyPlan, error) {
	var row plan.StudyPlan
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "pinyin").First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanNotFound
			}
			return err
		}
		if row.Status == "done" {
			return nil
		}
		if row.DoneCount < row.TargetCount {
			return ErrPlanIncomplete
		}
		now := time.Now()
		stars := 1
		if row.TargetCount > 0 && row.CorrectCount*100/row.TargetCount >= 80 {
			stars = 3
		} else if row.TargetCount > 0 && row.CorrectCount*100/row.TargetCount >= 60 {
			stars = 2
		}
		if err := tx.Model(&row).Updates(map[string]any{"status": "done", "completed_at": &now, "stars": stars}).Error; err != nil {
			return err
		}
		row.Status, row.CompletedAt, row.Stars = "done", &now, stars
		return nil
	})
	return row, err
}

func optionCount(raw string) int {
	var values []json.RawMessage
	if json.Unmarshal([]byte(raw), &values) != nil {
		return 0
	}
	return len(values)
}

func identityOrder(count int) []int {
	order := make([]int, count)
	for index := range order {
		order[index] = index
	}
	return order
}

func parseAnswer(raw string) (int, error) {
	var value struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return 0, fmt.Errorf("invalid answer: %w", err)
	}
	return value.Index, nil
}

func displayIndex(order []int, original int) int {
	for index, value := range order {
		if value == original {
			return index
		}
	}
	return -1
}

func appendPick(existing string, option int) string {
	value := fmt.Sprintf("%d", option)
	if existing == "" {
		return value
	}
	return existing + "," + value
}

func loadMastery(tx *gorm.DB, childID, kpID int64) learning.StateDTO {
	var row learningmodel.MasteryState
	if tx.Where("child_id = ? AND kp_id = ?", childID, kpID).First(&row).Error != nil {
		return learning.StateDTO{KpID: kpID, Status: string(mastery.StatusNotStarted)}
	}
	accuracy := 0.0
	if row.Attempts > 0 {
		accuracy = float64(row.Correct) / float64(row.Attempts)
	}
	return learning.StateDTO{KpID: kpID, Status: row.Status, Streak: row.Streak,
		Attempts: row.Attempts, Accuracy: accuracy, IntervalDays: row.IntervalDays, DueAt: row.DueAt}
}
