package practice

import (
	"encoding/json"
	"errors"

	"github.com/conchi/literacy-server/internal/plan"
	"github.com/conchi/study-learning/handwriting"
	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
	"time"
)

func (s *Service) answerHandwriting(tx *gorm.DB, childID int64, p plan.StudyPlan, item learningmodel.PlanItem, versionID int64, snap plan.Snapshot, input AnswerInput) (AnswerResult, error) {
	var result AnswerResult
	r := input.Response
	if r == nil || r.Kind != "handwriting" || r.SelectedOptionID != "" || r.HintsUsed < 0 || r.HintsUsed > 100 || input.CostMs < 0 || input.CostMs > 3600000 {
		return result, ErrInvalidAnswer
	}
	if err := handwriting.ValidateStrokes(r.Strokes); err != nil {
		return result, ErrInvalidAnswer
	}
	payload, _ := json.Marshal(r)
	var receipt Receipt
	err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&receipt).Error
	if err == nil {
		if receipt.PlanID != p.ID || receipt.PlanItemID != item.ID || receipt.QuestionVersionID != versionID || receipt.ResponseKind != "handwriting" || receipt.AnswerPayloadJSON != string(payload) {
			return result, ErrIdempotencyConflict
		}
		err = json.Unmarshal([]byte(receipt.ResponseJSON), &result)
		return result, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return result, err
	}
	var n int64
	if err := tx.Model(&learningmodel.Attempt{}).Where("child_id = ? AND client_id = ?", childID, input.ClientID).Count(&n).Error; err != nil {
		return result, err
	}
	if n > 0 {
		return result, ErrIdempotencyConflict
	}
	if item.Status != "pending" || item.Tries >= 3 {
		return result, ErrItemCompleted
	}
	var cached plan.CachedTemplate
	if err := tx.Where("revision_id = ? AND sha256 = ?", snap.WritingTemplate.RevisionID, snap.WritingTemplate.SHA256).First(&cached).Error; err != nil {
		return result, plan.ErrTemplateUnavailable
	}
	var template handwriting.Template
	if json.Unmarshal([]byte(cached.TemplateJSON), &template) != nil || template.Character != snap.TargetText {
		return result, plan.ErrTemplateUnavailable
	}
	evaluation, err := handwriting.Evaluate(r.Strokes, template, snap.EvaluationPolicyVersion)
	if err != nil {
		return result, plan.ErrTemplateUnavailable
	}
	if r.HintsUsed > 0 {
		evaluation.Assistance = "hinted"
	}
	correct := evaluation.Outcome == "passed"
	now := time.Now()
	state, applied, err := s.learning.ApplyOne(tx, childID, learning.AttemptInput{ClientID: input.ClientID, KpID: item.KpID, SkillCode: snap.SkillCode, IsCorrect: correct, Assisted: r.HintsUsed > 0, CostMs: input.CostMs, Source: mastery.SourceQuiz, At: now})
	if err != nil {
		return result, err
	}
	if !applied {
		return result, ErrIdempotencyConflict
	}
	tries := item.Tries + 1
	status := "pending"
	complete := correct || tries >= 3
	updates := map[string]any{"tries": tries, "cost_ms": item.CostMs + input.CostMs}
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
	result = AnswerResult{Correct: correct, AnswerIndex: -1, CanRetry: !complete, Tries: tries, Status: status, Mastery: state, Evaluation: &evaluation}
	response, _ := json.Marshal(result)
	evaluationJSON, _ := json.Marshal(evaluation)
	var attempt learningmodel.Attempt
	if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&attempt).Error; err != nil {
		return result, err
	}
	values := map[string]any{"attempt_id": attempt.ID, "child_id": childID, "client_id": input.ClientID, "plan_id": p.ID, "plan_item_id": item.ID, "question_version_id": versionID, "kp_id": item.KpID, "skill_code": snap.SkillCode, "question_type": snap.QuestionType, "selected_option_id": nil, "display_index": nil, "original_index": nil, "is_correct": correct, "cost_ms": input.CostMs, "response_json": string(response), "created_at": now, "response_kind": "handwriting", "answer_payload_json": string(payload), "evaluation_json": string(evaluationJSON), "evaluator_version": evaluation.EvaluatorVersion}
	return result, tx.Table("question_attempt_receipts").Create(values).Error
}
