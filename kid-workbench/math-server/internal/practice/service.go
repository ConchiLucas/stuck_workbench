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

	"github.com/conchi/math-server/internal/plan"
)

var (
	ErrPlanNotFound        = errors.New("plan not found")
	ErrItemNotFound        = errors.New("plan item not found")
	ErrItemCompleted       = errors.New("plan item completed")
	ErrPlanIncomplete      = errors.New("plan incomplete")
	ErrIdempotencyConflict = errors.New("client id belongs to another attempt")
)

type Service struct {
	db       *gorm.DB
	learning *learning.Service
}

func NewService(db *gorm.DB, cfg mastery.Config) *Service {
	return &Service{db: db, learning: learning.NewService(cfg)}
}

func (s *Service) Answer(ctx context.Context, childID, planID, itemID int64, input AnswerInput) (AnswerResult, error) {
	if strings.TrimSpace(input.ClientID) == "" || input.OptionIndex < 0 || input.OptionIndex > 3 || input.CostMs < 0 || input.CostMs > 3600000 {
		return AnswerResult{}, errors.New("invalid answer input")
	}
	var output AnswerResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var planRow plan.StudyPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND child_id = ? AND subject_code = ? AND plan_kind <> ''", planID, childID, "math").
			First(&planRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanNotFound
			}
			return err
		}
		var item learningmodel.PlanItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND plan_id = ?", itemID, planID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		var snapshot plan.QuestionSnapshot
		if err := json.Unmarshal([]byte(item.QuestionSnapshot), &snapshot); err != nil {
			return fmt.Errorf("invalid question snapshot: %w", err)
		}
		if input.OptionIndex >= len(optionsArray(snapshot)) {
			return errors.New("option index out of range")
		}
		isCorrect := input.OptionIndex == snapshot.AnswerIndex

		var existing learningmodel.Attempt
		if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&existing).Error; err == nil {
			if existing.KpID != item.KpID || existing.QuestionID == nil || *existing.QuestionID != item.QuestionID {
				return ErrIdempotencyConflict
			}
			output = replayResult(tx, existing, item, snapshot.AnswerIndex, planRow)
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if item.Status != "pending" {
			return ErrItemCompleted
		}

		state, applied, err := s.learning.ApplyOne(tx, childID, learning.AttemptInput{
			ClientID: input.ClientID, KpID: item.KpID, QuestionID: &item.QuestionID,
			SkillCode: snapshot.Code,
			IsCorrect: isCorrect, CostMs: input.CostMs, Source: mastery.SourceQuiz, At: time.Now(),
		})
		if err != nil {
			return err
		}
		if !applied {
			return ErrIdempotencyConflict
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
			updates["status"], updates["answered_at"] = status, &now
		}
		if err := tx.Model(&learningmodel.PlanItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		planUpdates := map[string]any{"status": "doing"}
		if planRow.StartedAt == nil {
			planUpdates["started_at"] = time.Now()
		}
		if completedNow {
			planRow.DoneCount++
			planUpdates["done_count"] = planRow.DoneCount
			if isCorrect {
				planRow.CorrectCount++
				planUpdates["correct_count"] = planRow.CorrectCount
			}
		}
		planRow.Status = "doing"
		if err := tx.Model(&plan.StudyPlan{}).Where("id = ?", planID).Updates(planUpdates).Error; err != nil {
			return err
		}
		output = AnswerResult{
			Correct: isCorrect, AnswerIndex: snapshot.AnswerIndex, CanRetry: !isCorrect && tries < 2,
			Tries: tries, Status: status, Mastery: state, Plan: summary(planRow),
		}
		return nil
	})
	return output, err
}

func (s *Service) Finish(ctx context.Context, childID, planID int64) (plan.StudyPlan, error) {
	var row plan.StudyPlan
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND child_id = ? AND subject_code = ? AND plan_kind <> ''", planID, childID, "math").
			First(&row).Error; err != nil {
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
		stars := 1
		if row.TargetCount > 0 && row.CorrectCount*10 >= row.TargetCount*9 {
			stars = 3
		} else if row.TargetCount > 0 && row.CorrectCount*10 >= row.TargetCount*7 {
			stars = 2
		}
		now := time.Now()
		var costMs int
		if err := tx.Table("plan_items").Select("COALESCE(SUM(cost_ms), 0)").Where("plan_id = ?", planID).Scan(&costMs).Error; err != nil {
			return err
		}
		durationSec := costMs / 1000
		if err := tx.Model(&row).Updates(map[string]any{"status": "done", "completed_at": &now, "stars": stars, "duration_sec": durationSec}).Error; err != nil {
			return err
		}
		reward := 1 + stars
		if err := tx.Create(&learningmodel.FlowerLedger{
			ChildID: childID, Delta: reward, Reason: "plan_done", RefType: "study_plan", RefID: &row.ID, CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Table("children").Where("id = ?", childID).
			UpdateColumn("flowers", gorm.Expr("flowers + ?", reward)).Error; err != nil {
			return err
		}
		row.Status, row.CompletedAt, row.Stars, row.DurationSec = "done", &now, stars, durationSec
		return nil
	})
	return row, err
}

func optionsArray(snapshot plan.QuestionSnapshot) []json.RawMessage {
	var options []json.RawMessage
	_ = json.Unmarshal(snapshot.Options, &options)
	return options
}

func appendPick(existing string, option int) string {
	value := fmt.Sprintf("%d", option)
	if existing == "" {
		return value
	}
	return existing + "," + value
}

func replayResult(tx *gorm.DB, attempt learningmodel.Attempt, item learningmodel.PlanItem, answerIndex int, row plan.StudyPlan) AnswerResult {
	return AnswerResult{
		Correct: attempt.IsCorrect, AnswerIndex: answerIndex,
		CanRetry: item.Status == "pending" && !attempt.IsCorrect && item.Tries < 2,
		Tries:    item.Tries, Status: item.Status, Mastery: loadMastery(tx, attempt.ChildID, item.KpID), Plan: summary(row),
	}
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

func summary(row plan.StudyPlan) PlanSummary {
	return PlanSummary{Status: row.Status, TargetCount: row.TargetCount, DoneCount: row.DoneCount, CorrectCount: row.CorrectCount}
}
