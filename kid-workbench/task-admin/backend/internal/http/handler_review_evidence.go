package httpapi

import (
	"encoding/json"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

type reviewEvidence struct {
	ResponseKind            string             `json:"responseKind"`
	AnswerPayloadJSON       string             `json:"-"`
	EvaluationJSON          string             `json:"-"`
	AnswerPayload           json.RawMessage    `gorm:"-" json:"answerPayload,omitempty"`
	Evaluation              json.RawMessage    `gorm:"-" json:"evaluation,omitempty"`
	ReceiptID               int64              `json:"receiptId"`
	PlanID                  int64              `json:"planId"`
	SourceQuestionVersionID int64              `json:"sourceQuestionVersionId"`
	CreatedAt               time.Time          `json:"createdAt"`
	SelectedOptionID        string             `json:"selectedOptionId"`
	TargetText              string             `json:"targetText"`
	QuestionType            string             `json:"questionType"`
	SelectedOption          *generation.Option `gorm:"-" json:"selectedOption,omitempty"`
	CorrectOption           *generation.Option `gorm:"-" json:"correctOption,omitempty"`
	SnapshotJSON            string             `json:"-"`
}

func (h *handlers) reviewEvidence(c *gin.Context) {
	if h.deps.Generation == nil {
		c.Status(503)
		return
	}
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id < 1 {
		c.JSON(400, gin.H{"error": "无效任务编号"})
		return
	}
	s := h.deps.Generation
	task, e := s.Get(c.Request.Context(), id)
	if e != nil {
		genError(c, e)
		return
	}
	rows := []reviewEvidence{}
	if task.Kind != "review" || task.TargetChildID == nil {
		c.JSON(200, rows)
		return
	}
	extra := ""
	if s.DB.Migrator().HasColumn("question_attempt_receipts", "evaluation_json") {
		extra = ",a.response_kind,a.answer_payload_json,a.evaluation_json"
	}
	e = s.DB.WithContext(c.Request.Context()).Raw(`SELECT a.id AS receipt_id,a.plan_id,a.question_version_id AS source_question_version_id,a.created_at,a.selected_option_id,v.snapshot_json`+extra+` FROM question_task_review_sources s JOIN question_attempt_receipts a ON a.id=s.source_receipt_id AND a.question_version_id=s.source_question_version_id JOIN question_versions v ON v.id=a.question_version_id WHERE s.task_id=? AND a.child_id=? AND `+s.ReviewPredicate("a")+` ORDER BY a.created_at DESC,a.id DESC`, id, *task.TargetChildID).Scan(&rows).Error
	if e != nil {
		genError(c, e)
		return
	}
	for i := range rows {
		if json.Valid([]byte(rows[i].AnswerPayloadJSON)) {
			rows[i].AnswerPayload = json.RawMessage(rows[i].AnswerPayloadJSON)
		}
		if json.Valid([]byte(rows[i].EvaluationJSON)) {
			rows[i].Evaluation = json.RawMessage(rows[i].EvaluationJSON)
		}
		var snapshot generation.Snapshot
		if e = json.Unmarshal([]byte(rows[i].SnapshotJSON), &snapshot); e != nil {
			genError(c, e)
			return
		}
		rows[i].TargetText = snapshot.TargetText
		rows[i].QuestionType = snapshot.QuestionType
		for _, option := range snapshot.Options {
			option := option
			if option.ID == rows[i].SelectedOptionID {
				rows[i].SelectedOption = &option
			}
			if option.ID == snapshot.AnswerOptionID {
				rows[i].CorrectOption = &option
			}
		}
	}
	c.JSON(200, rows)
}
