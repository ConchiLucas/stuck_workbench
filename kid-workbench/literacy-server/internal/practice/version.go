package practice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/literacy-server/internal/plan"
	"github.com/conchi/study-learning/handwriting"
	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
	"time"
)

type Receipt struct {
	ResponseKind      string
	AnswerPayloadJSON string
	EvaluationJSON    string
	EvaluatorVersion  string
	ID                int64 `gorm:"primaryKey"`
	AttemptID         int64
	ChildID           int64
	ClientID          string
	PlanID            int64
	PlanItemID        int64
	QuestionVersionID int64
	KpID              int64
	SkillCode         string
	QuestionType      string
	SelectedOptionID  string
	DisplayIndex      int
	OriginalIndex     int
	IsCorrect         bool
	CostMs            int
	ResponseJSON      string
	CreatedAt         time.Time
}

func (Receipt) TableName() string { return "question_attempt_receipts" }
func (s *Service) answerVersion(tx *gorm.DB, childID int64, p plan.StudyPlan, item learningmodel.PlanItem, versionID int64, input AnswerInput) (AnswerResult, error) {
	var result AnswerResult
	snap, err := plan.ParseSnapshot(item.QuestionSnapshot)
	if err != nil {
		return result, err
	}
	if snap.KpID != item.KpID {
		return result, fmt.Errorf("snapshot knowledge point mismatch")
	}
	if snap.Interaction == "handwriting" {
		return s.answerHandwriting(tx, childID, p, item, versionID, snap, input)
	}
	if snap.SchemaVersion == 2 && (input.Response == nil || input.Response.Kind != "choice") {
		return result, ErrInvalidAnswer
	}
	order, err := plan.ParseOrder(item.OptionOrder)
	if err != nil {
		return result, err
	}
	if len(order) != len(snap.Options) {
		return result, fmt.Errorf("invalid frozen option order")
	}
	seen := map[int]bool{}
	answerIndex := -1
	for display, original := range order {
		if original < 0 || original >= len(snap.Options) || seen[original] {
			return result, fmt.Errorf("invalid frozen option order")
		}
		seen[original] = true
		if snap.Options[original].ID == snap.AnswerOptionID {
			answerIndex = display
		}
	}
	if input.Response != nil {
		if input.Response.Kind != "choice" || len(input.Response.Strokes) > 0 || input.Response.HintsUsed != 0 {
			return result, ErrInvalidAnswer
		}
		input.OptionIndex = -1
		for i, original := range order {
			if snap.Options[original].ID == input.Response.SelectedOptionID {
				input.OptionIndex = i
			}
		}
	}
	if input.OptionIndex < 0 || input.OptionIndex >= len(order) || input.CostMs < 0 || input.CostMs > 3600000 {
		return result, ErrInvalidAnswer
	}
	original := order[input.OptionIndex]
	selected := snap.Options[original].ID
	var receipt Receipt
	err = tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&receipt).Error
	if err == nil {
		if receipt.PlanID != p.ID || receipt.PlanItemID != item.ID || receipt.QuestionVersionID != versionID || receipt.SelectedOptionID != selected || receipt.DisplayIndex != input.OptionIndex {
			return result, ErrIdempotencyConflict
		}
		err = json.Unmarshal([]byte(receipt.ResponseJSON), &result)
		return result, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return result, err
	}
	var attemptCount int64
	if err := tx.Model(&learningmodel.Attempt{}).Where("child_id = ? AND client_id = ?", childID, input.ClientID).Count(&attemptCount).Error; err != nil {
		return result, err
	}
	if attemptCount > 0 {
		return result, ErrIdempotencyConflict
	}
	if item.Status != "pending" || item.Tries >= 2 {
		return result, ErrItemCompleted
	}
	correct := selected == snap.AnswerOptionID
	now := time.Now()
	state, applied, err := s.learning.ApplyOne(tx, childID, learning.AttemptInput{ClientID: input.ClientID, KpID: item.KpID, QuestionID: nil, SkillCode: snap.SkillCode, IsCorrect: correct, CostMs: input.CostMs, Source: mastery.SourceQuiz, At: now})
	if err != nil {
		return result, err
	}
	if !applied {
		return result, ErrIdempotencyConflict
	}
	tries := item.Tries + 1
	status := "pending"
	complete := correct || tries >= 2
	updates := map[string]any{"tries": tries, "cost_ms": item.CostMs + input.CostMs, "picks": appendPick(item.Picks, input.OptionIndex)}
	if complete {
		status = "wrong"
		if correct {
			status = "correct"
		}
		updates["answered_at"] = now
	}
	updates["status"] = status
	if err := tx.Model(&learningmodel.PlanItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return result, err
	}
	planUpdates := map[string]any{"status": "doing"}
	if p.StartedAt == nil {
		planUpdates["started_at"] = now
	}
	if complete {
		planUpdates["done_count"] = gorm.Expr("done_count + 1")
		if correct {
			planUpdates["correct_count"] = gorm.Expr("correct_count + 1")
		}
	}
	if err := tx.Model(&plan.StudyPlan{}).Where("id = ?", p.ID).Updates(planUpdates).Error; err != nil {
		return result, err
	}
	result = AnswerResult{AnswerOptionID: snap.AnswerOptionID, Correct: correct, AnswerIndex: answerIndex, CanRetry: !correct && tries < 2, Tries: tries, Status: status, Mastery: state}
	response, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	var attempt learningmodel.Attempt
	if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&attempt).Error; err != nil {
		return result, err
	}
	receipt = Receipt{AttemptID: attempt.ID, ChildID: childID, ClientID: input.ClientID, PlanID: p.ID, PlanItemID: item.ID, QuestionVersionID: versionID, KpID: item.KpID, SkillCode: snap.SkillCode, QuestionType: snap.QuestionType, SelectedOptionID: selected, DisplayIndex: input.OptionIndex, OriginalIndex: original, IsCorrect: correct, CostMs: input.CostMs, ResponseJSON: string(response), CreatedAt: now}
	if snap.SchemaVersion == 2 {
		receipt.ResponseKind = "choice"
		payload, _ := json.Marshal(ResponsePayload{Kind: "choice", SelectedOptionID: selected})
		receipt.AnswerPayloadJSON = string(payload)
		evaluation := handwriting.EvaluationResult{Outcome: "not_passed", Assistance: "none", EvaluatorVersion: "choice-v1", Metrics: map[string]float64{}}
		if correct {
			evaluation.Outcome = "passed"
		}
		evaluationJSON, _ := json.Marshal(evaluation)
		receipt.EvaluationJSON = string(evaluationJSON)
		receipt.EvaluatorVersion = evaluation.EvaluatorVersion
		result.Evaluation = &evaluation
		response, _ = json.Marshal(result)
		receipt.ResponseJSON = string(response)
		return result, tx.Create(&receipt).Error
	}
	return result, tx.Omit("ResponseKind", "AnswerPayloadJSON", "EvaluationJSON", "EvaluatorVersion").Create(&receipt).Error
}
