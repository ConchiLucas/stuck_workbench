package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-learning/handwriting"
	"github.com/conchi/study-learning/literacycontract"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

var ErrTaskConflict = errors.New("任务版本已变更或领取请求冲突")
var ErrTaskNotAvailable = errors.New("任务尚未发布或不属于这个孩子")

type ClaimInput struct {
	RevisionID int64  `json:"revisionId"`
	ClaimKey   string `json:"claimKey"`
}
type TaskSummary struct {
	QuestionTypes []string `json:"questionTypes" gorm:"-"`
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	Kind          string   `json:"kind"`
	RevisionID    int64    `json:"revisionId"`
	TargetCount   int      `json:"targetCount"`
	PlanID        *int64   `json:"planId,omitempty"`
	PlanStatus    string   `json:"planStatus,omitempty"`
}
type TaskList struct {
	Items []TaskSummary `json:"items"`
}
type MediaRef struct {
	RevisionID string `json:"revisionId"`
	Kind       string `json:"kind"`
	SHA256     string `json:"sha256"`
}
type SnapshotOption struct {
	ID    string    `json:"id"`
	KpID  int64     `json:"kpId"`
	Text  string    `json:"text"`
	Image *MediaRef `json:"image,omitempty"`
	Audio *MediaRef `json:"audio,omitempty"`
}
type SnapshotStem struct {
	Text  string    `json:"text,omitempty"`
	Image *MediaRef `json:"image,omitempty"`
	Audio *MediaRef `json:"audio,omitempty"`
}
type Snapshot struct {
	Interaction             string           `json:"interaction,omitempty"`
	ResponseSchemaVersion   int              `json:"responseSchemaVersion,omitempty"`
	EvaluationPolicyVersion string           `json:"evaluationPolicyVersion,omitempty"`
	Presentation            map[string]any   `json:"presentation,omitempty"`
	WritingTemplate         *MediaRef        `json:"writingTemplate,omitempty"`
	SchemaVersion           int              `json:"schemaVersion"`
	SubjectCode             string           `json:"subjectCode"`
	KpID                    int64            `json:"kpId"`
	TargetText              string           `json:"targetText,omitempty"`
	QuestionType            string           `json:"questionType"`
	SkillCode               string           `json:"skillCode"`
	TemplateVersion         string           `json:"templateVersion"`
	Prompt                  string           `json:"prompt"`
	Stem                    SnapshotStem     `json:"stem"`
	Options                 []SnapshotOption `json:"options"`
	AnswerOptionID          string           `json:"answerOptionId,omitempty"`
	Explanation             string           `json:"explanation,omitempty"`
	MaterialRevisionIDs     []string         `json:"materialRevisionIds"`
}

func ParseSnapshot(raw string) (Snapshot, error) {
	var s Snapshot
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return s, err
	}
	if (s.SchemaVersion != 1 && s.SchemaVersion != 2) || s.SubjectCode != "literacy" || s.KpID <= 0 || !literacycontract.Supports(s.QuestionType) || s.SkillCode != s.QuestionType {
		return s, fmt.Errorf("invalid question snapshot")
	}
	if s.SchemaVersion == 2 {
		if s.ResponseSchemaVersion != 2 {
			return s, fmt.Errorf("invalid response schema")
		}
		if s.QuestionType == "write_char" {
			if s.Interaction != "handwriting" || s.EvaluationPolicyVersion != handwriting.PolicyV1 || len(s.Options) != 0 || s.AnswerOptionID != "" || s.WritingTemplate == nil || s.WritingTemplate.Kind != "writing_template" || s.Stem.Audio == nil {
				return s, fmt.Errorf("invalid handwriting snapshot")
			}
			return s, nil
		}
		if s.Interaction != "choice" {
			return s, fmt.Errorf("invalid choice interaction")
		}
	}
	if len(s.Options) != 4 || s.QuestionType == "write_char" {
		return s, fmt.Errorf("invalid choice snapshot")
	}
	found := false
	seen := map[string]bool{}
	for _, o := range s.Options {
		if o.ID == "" || seen[o.ID] {
			return s, fmt.Errorf("invalid option IDs")
		}
		seen[o.ID] = true
		if o.ID == s.AnswerOptionID {
			found = true
		}
	}
	if !found {
		return s, fmt.Errorf("snapshot answer missing")
	}
	return s, nil
}
func MediaURL(ref *MediaRef) string {
	if ref == nil {
		return ""
	}
	return "/api/v1/literacy/material-revisions/" + ref.RevisionID + "/media/" + ref.Kind
}
func (s *Service) ListTasks(ctx context.Context, childID int64) (TaskList, error) {
	out := TaskList{Items: []TaskSummary{}}
	var n int64
	if err := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Count(&n).Error; err != nil {
		return out, err
	}
	if n == 0 {
		return out, ErrChildNotFound
	}
	err := s.db.WithContext(ctx).Table("question_tasks t").Select("t.id,t.title,t.kind,t.published_revision_id AS revision_id,(SELECT COUNT(*) FROM question_versions q WHERE q.revision_id=t.published_revision_id) AS target_count").Where("t.subject_code = ? AND t.status = ? AND t.source_mode = ? AND t.published_revision_id IS NOT NULL AND (t.target_child_id IS NULL OR t.target_child_id = ?) AND (t.kind = 'practice' OR (t.kind = 'review' AND t.target_child_id = ?))", "literacy", "published", "material_template", childID, childID).Order("t.id DESC").Scan(&out.Items).Error
	if err != nil {
		return out, err
	}
	for i := range out.Items {
		if err := s.db.WithContext(ctx).Table("question_versions").Distinct("question_type").Where("revision_id = ?", out.Items[i].RevisionID).Order("question_type").Pluck("question_type", &out.Items[i].QuestionTypes).Error; err != nil {
			return out, err
		}
		var p StudyPlan
		err := s.db.WithContext(ctx).Where("child_id = ? AND source_question_task_revision_id = ?", childID, out.Items[i].RevisionID).Order("CASE WHEN status = 'done' THEN 1 ELSE 0 END, id DESC").First(&p).Error
		if err == nil {
			out.Items[i].PlanID = &p.ID
			out.Items[i].PlanStatus = p.Status
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return out, err
		}
	}
	return out, nil
}
func (s *Service) ClaimTask(ctx context.Context, childID, taskID int64, input ClaimInput) (Detail, error) {
	if input.RevisionID <= 0 || strings.TrimSpace(input.ClaimKey) == "" || len(input.ClaimKey) > 120 {
		return Detail{}, ErrTaskConflict
	}
	if err := s.cacheRevisionTemplates(ctx, input.RevisionID); err != nil {
		return Detail{}, err
	}
	var planID int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var child struct{ ID int64 }
		if err := tx.Table("children").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", childID).Take(&child).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrChildNotFound
			}
			return err
		}
		var task struct {
			ID                                    int64
			SubjectCode, Status, SourceMode, Kind string
			PublishedRevisionID                   *int64
			TargetChildID                         *int64
		}
		if err := tx.Table("question_tasks").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", taskID).Take(&task).Error; err != nil {
			return ErrTaskNotAvailable
		}
		var existing struct{ PlanID, TaskID, RevisionID int64 }
		err := tx.Table("study_plan_task_claims").Where("child_id = ? AND claim_key = ?", childID, input.ClaimKey).Take(&existing).Error
		if err == nil {
			if existing.TaskID != taskID || existing.RevisionID != input.RevisionID {
				return ErrTaskConflict
			}
			planID = existing.PlanID
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		bindClaim := func(id int64) error {
			return tx.Table("study_plan_task_claims").Create(map[string]any{"child_id": childID, "claim_key": input.ClaimKey, "task_id": taskID, "revision_id": input.RevisionID, "plan_id": id}).Error
		}
		if (task.Kind != "practice" && (task.Kind != "review" || task.TargetChildID == nil || *task.TargetChildID != childID)) || task.SubjectCode != "literacy" || task.Status != "published" || task.SourceMode != "material_template" || task.PublishedRevisionID == nil || (task.TargetChildID != nil && *task.TargetChildID != childID) {
			return ErrTaskNotAvailable
		}
		if *task.PublishedRevisionID != input.RevisionID {
			return ErrTaskConflict
		}
		var continued StudyPlan
		err = tx.Where("child_id = ? AND source_question_task_revision_id = ? AND status <> 'done'", childID, input.RevisionID).Order("id DESC").First(&continued).Error
		if err == nil {
			planID = continued.ID
			return bindClaim(planID)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var revision struct{ TaskID int64 }
		if err := tx.Table("question_task_revisions").Where("id = ?", input.RevisionID).Take(&revision).Error; err != nil {
			return err
		}
		if revision.TaskID != taskID {
			return ErrTaskConflict
		}
		var versions []struct {
			ID                                    int64
			KpID                                  int64
			Seq                                   int
			QuestionType, SkillCode, SnapshotJSON string
		}
		if err := tx.Table("question_versions").Where("revision_id = ?", input.RevisionID).Order("seq").Find(&versions).Error; err != nil {
			return err
		}
		if len(versions) == 0 || len(versions) > 20 {
			return ErrNoQuestions
		}
		row := StudyPlan{ChildID: childID, PlanDate: time.Now().Format("2006-01-02"), SubjectCode: "literacy", Status: "pending", TargetCount: len(versions), CreatedAt: time.Now()}
		if err := tx.Table("study_plans").Where("child_id = ? AND plan_date = ?", childID, row.PlanDate).Select("COALESCE(MAX(seq_no),0)").Scan(&row.SeqNo).Error; err != nil {
			return err
		}
		row.SeqNo++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		planID = row.ID
		if err := tx.Model(&row).Updates(map[string]any{"source_question_task_id": taskID, "source_question_task_revision_id": input.RevisionID, "task_claim_key": input.ClaimKey}).Error; err != nil {
			return err
		}
		for i, v := range versions {
			snap, err := ParseSnapshot(v.SnapshotJSON)
			if err != nil {
				return err
			}
			if snap.KpID != v.KpID || snap.QuestionType != v.QuestionType || snap.SkillCode != v.SkillCode {
				return fmt.Errorf("question metadata differs from snapshot")
			}
			options := []map[string]any{}
			correct := 0
			for j, o := range snap.Options {
				options = append(options, map[string]any{"id": o.ID, "kpId": o.KpID, "label": o.Text, "image": MediaURL(o.Image), "audio": MediaURL(o.Audio)})
				if o.ID == snap.AnswerOptionID {
					correct = j
				}
			}
			opts, _ := json.Marshal(options)
			answer, _ := json.Marshal(map[string]int{"index": correct})
			visual, _ := json.Marshal(map[string]string{"image": MediaURL(snap.Stem.Image)})
			speech, _ := json.Marshal(map[string]string{"audio": MediaURL(snap.Stem.Audio)})
			item := map[string]any{"plan_id": row.ID, "seq": i + 1, "kp_id": v.KpID, "question_id": nil, "question_version_id": v.ID, "bucket": "new", "status": "pending", "option_order": formatOrder(shuffledOrder(len(snap.Options), row.ID+v.ID)), "question_stem": snap.Prompt, "question_options": string(opts), "question_answer": string(answer), "question_visual": string(visual), "question_speech": string(speech), "question_snapshot": v.SnapshotJSON, "explanation": snap.Explanation, "content_snapshot_version": 2}
			if err := tx.Table("plan_items").Create(item).Error; err != nil {
				return err
			}
		}
		return bindClaim(planID)
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, childID, planID)
}
