package taskgen

import (
	"encoding/json"
	"github.com/conchi/study-task-admin/internal/generation"
	"gorm.io/gorm"
	"time"
)

type Task struct {
	SourceReviewSuggestionID *int64            `json:"sourceReviewSuggestionId,omitempty" gorm:"-"`
	ID                       int64             `gorm:"primaryKey" json:"id"`
	SubjectCode              string            `json:"subjectCode"`
	Title                    string            `json:"title"`
	ModuleCode               string            `json:"moduleCode"`
	ModuleName               string            `json:"moduleName"`
	TargetCount              int               `json:"targetCount"`
	Status                   string            `json:"status"`
	Kind                     string            `gorm:"default:practice" json:"kind"`
	SourceMode               string            `gorm:"default:legacy_pool" json:"sourceMode"`
	SpecJSON                 string            `gorm:"column:spec_json;default:'{}'" json:"-"`
	RowVersion               int64             `gorm:"default:1" json:"rowVersion"`
	ActiveRevisionID         *int64            `json:"activeRevisionId"`
	PublishedRevisionID      *int64            `json:"publishedRevisionId"`
	TargetChildID            *int64            `json:"targetChildId"`
	ParentTaskID             *int64            `json:"parentTaskId"`
	ReviewKey                *string           `gorm:"uniqueIndex" json:"-"`
	CreatedAt                time.Time         `json:"createdAt"`
	UpdatedAt                time.Time         `json:"updatedAt"`
	Spec                     generation.Spec   `gorm:"-" json:"spec"`
	Items                    []QuestionVersion `gorm:"-" json:"items"`
	Revisions                []Revision        `gorm:"-" json:"revisions"`
	LastError                string            `gorm:"-" json:"lastError,omitempty"`
}

func (Task) TableName() string { return "question_tasks" }

type Revision struct {
	ID              int64             `gorm:"primaryKey" json:"id"`
	TaskID          int64             `gorm:"uniqueIndex:uq_task_revision" json:"taskId"`
	RevisionNo      int               `gorm:"uniqueIndex:uq_task_revision" json:"revisionNo"`
	SpecJSON        string            `json:"-"`
	Seed            int64             `json:"seed"`
	TemplateVersion string            `json:"templateVersion"`
	CreatedAt       time.Time         `json:"createdAt"`
	Items           []QuestionVersion `gorm:"-" json:"items,omitempty"`
}

func (Revision) TableName() string { return "question_task_revisions" }

type QuestionVersion struct {
	ID                      int64               `gorm:"primaryKey" json:"id"`
	RevisionID              int64               `gorm:"uniqueIndex:uq_revision_seq;uniqueIndex:uq_revision_fp" json:"revisionId"`
	Seq                     int                 `gorm:"uniqueIndex:uq_revision_seq" json:"seq"`
	KpID                    int64               `gorm:"column:kp_id" json:"kpId"`
	QuestionType            string              `json:"questionType"`
	SkillCode               string              `json:"skillCode"`
	Fingerprint             string              `gorm:"uniqueIndex:uq_revision_fp" json:"fingerprint"`
	SnapshotJSON            string              `json:"-"`
	SourceQuestionVersionID *int64              `json:"sourceQuestionVersionId,omitempty"`
	Snapshot                generation.Snapshot `gorm:"-" json:"snapshot"`
}

func (QuestionVersion) TableName() string { return "question_versions" }

type Run struct {
	ID          int64  `gorm:"primaryKey"`
	TaskID      int64  `gorm:"uniqueIndex:uq_generation_key"`
	Key         string `gorm:"uniqueIndex:uq_generation_key"`
	RequestHash string
	State       string
	Seed        int64
	RevisionID  *int64
	Error       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Run) TableName() string { return "question_task_generation_runs" }

type ReviewSource struct {
	ID                      int64 `gorm:"primaryKey"`
	TaskID                  int64 `gorm:"uniqueIndex:uq_review_source"`
	SourceReceiptID         int64 `gorm:"uniqueIndex:uq_review_source"`
	SourceQuestionVersionID int64
	Reason                  string
}

func (ReviewSource) TableName() string { return "question_task_review_sources" }
func Migrate(g *gorm.DB) error {
	for _, name := range []string{"Kind", "SourceMode", "SpecJSON", "RowVersion", "ActiveRevisionID", "PublishedRevisionID", "TargetChildID", "ParentTaskID", "ReviewKey"} {
		if !g.Migrator().HasColumn(&Task{}, name) {
			if err := g.Migrator().AddColumn(&Task{}, name); err != nil {
				return err
			}
		}
	}
	if err := g.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uq_question_task_review_key ON question_tasks(review_key)").Error; err != nil {
		return err
	}
	return g.AutoMigrate(&Revision{}, &QuestionVersion{}, &Run{}, &ReviewSource{}, &ReviewJob{}, &WorkerState{})
}
func decodeTask(t *Task) {
	_ = json.Unmarshal([]byte(t.SpecJSON), &t.Spec)
	if t.Items == nil {
		t.Items = []QuestionVersion{}
	}
}
func decodeItems(items []QuestionVersion) error {
	for i := range items {
		if err := json.Unmarshal([]byte(items[i].SnapshotJSON), &items[i].Snapshot); err != nil {
			return err
		}
	}
	return nil
}
