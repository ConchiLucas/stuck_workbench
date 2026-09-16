// Package reviewsuggestion persists explicit review requirements. It never updates learning facts.
package reviewsuggestion

import (
	"encoding/json"
	"gorm.io/gorm"
	"time"
)

type Source struct {
	Kind              string `json:"kind"`
	ReceiptID         int64  `json:"receiptId,omitempty"`
	QuestionVersionID int64  `json:"questionVersionId,omitempty"`
	ClientID          string `json:"clientId,omitempty"`
	InstanceID        string `json:"instanceId,omitempty"`
	PlanID            int64  `json:"planId,omitempty"`
	ItemID            int64  `json:"itemId,omitempty"`
}
type EvidenceInput struct {
	AttemptID int64  `json:"attemptId"`
	Role      string `json:"role"`
	Source    Source `json:"source"`
}
type TargetInput struct {
	Key                      string          `json:"key"`
	KpID                     int64           `json:"kpId"`
	QuestionType             string          `json:"questionType"`
	ReasonCode               string          `json:"reasonCode"`
	Mode                     string          `json:"mode"`
	RequestedCount           int             `json:"requestedCount"`
	PreferredDistractorKpIDs []int64         `json:"preferredDistractorKpIds"`
	Evidence                 []EvidenceInput `json:"evidence"`
}
type Input struct {
	SupersedesID    *int64        `json:"supersedesId,omitempty"`
	SchemaVersion   int           `json:"schemaVersion"`
	Title           string        `json:"title"`
	AnalysisVersion string        `json:"analysisVersion"`
	AnalysisAsOf    time.Time     `json:"analysisAsOf"`
	SubjectCode     string        `json:"subjectCode"`
	Targets         []TargetInput `json:"targets"`
}
type CommandInput struct {
	ExpectedRowVersion int64    `json:"expectedRowVersion"`
	PartitionKeys      []string `json:"partitionKeys,omitempty"`
}
type Suggestion struct {
	SupersedesID     *int64      `json:"supersedesId,omitempty"`
	ID               int64       `json:"id" gorm:"primaryKey"`
	ChildID          int64       `json:"childId"`
	SubjectCode      string      `json:"subjectCode"`
	Title            string      `json:"title"`
	SchemaVersion    int         `json:"schemaVersion"`
	AnalysisVersion  string      `json:"analysisVersion"`
	AnalysisAsOf     time.Time   `json:"analysisAsOf"`
	SpecJSON         string      `json:"-"`
	ContentHash      string      `json:"-"`
	Lifecycle        string      `json:"lifecycle"`
	RowVersion       int64       `json:"rowVersion"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	GenerationStatus *string     `json:"generationStatus" gorm:"-"`
	RequestedCount   int         `json:"requestedCount" gorm:"-"`
	GeneratedCount   int         `json:"generatedCount" gorm:"-"`
	TaskCount        int         `json:"taskCount" gorm:"-"`
	Targets          []Target    `json:"targets" gorm:"-"`
	Partitions       []Partition `json:"partitions" gorm:"-"`
}

func (Suggestion) TableName() string { return "review_suggestions" }

type Target struct {
	Title          string `json:"title" gorm:"-"`
	ID             int64  `json:"id" gorm:"primaryKey"`
	SuggestionID   int64  `json:"-" gorm:"uniqueIndex:uq_rs_target"`
	TargetKey      string `json:"key" gorm:"uniqueIndex:uq_rs_target"`
	KpID           int64  `json:"kpId"`
	QuestionType   string `json:"questionType"`
	ModuleCode     string `json:"moduleCode"`
	ReasonCode     string `json:"reasonCode"`
	ReasonText     string `json:"reasonText"`
	Mode           string `json:"mode"`
	RequestedCount int    `json:"requestedCount"`
}

func (Target) TableName() string { return "review_suggestion_targets" }

type Evidence struct {
	SourceReceiptID         *int64          `json:"-"`
	SourceQuestionVersionID *int64          `json:"-"`
	SourcePlanID            *int64          `json:"-"`
	SourcePlanItemID        *int64          `json:"-"`
	SourcePinyinInstanceID  *string         `json:"-"`
	SourceChildID           *int64          `json:"-"`
	SourceClientID          *string         `json:"-"`
	ID                      int64           `json:"id" gorm:"primaryKey"`
	TargetID                int64           `json:"targetId" gorm:"uniqueIndex:uq_rs_evidence"`
	AttemptID               int64           `json:"attemptId" gorm:"uniqueIndex:uq_rs_evidence"`
	Role                    string          `json:"role" gorm:"uniqueIndex:uq_rs_evidence"`
	SourceKind              string          `json:"sourceKind"`
	SourceJSON              string          `json:"-"`
	EvidenceSummaryJSON     string          `json:"-"`
	Source                  Source          `json:"source" gorm:"-"`
	Summary                 json.RawMessage `json:"summary" gorm:"-"`
}

func (Evidence) TableName() string { return "review_suggestion_evidence" }

type Run struct {
	ID              int64      `json:"id" gorm:"primaryKey"`
	SuggestionID    int64      `json:"suggestionId"`
	State           string     `json:"state"`
	CancelRequested bool       `json:"cancelRequested"`
	ErrorCode       string     `json:"errorCode,omitempty"`
	ResultJSON      string     `json:"-"`
	CreatedAt       time.Time  `json:"createdAt"`
	FinishedAt      *time.Time `json:"finishedAt"`
}

func (Run) TableName() string { return "review_suggestion_runs" }

type Partition struct {
	WarningsJSON   string          `json:"-"`
	Warnings       json.RawMessage `json:"warnings,omitempty" gorm:"-"`
	ID             int64           `json:"id" gorm:"primaryKey"`
	SuggestionID   int64           `json:"-" gorm:"uniqueIndex:uq_rs_partition"`
	PartitionKey   string          `json:"key" gorm:"uniqueIndex:uq_rs_partition"`
	TargetKeysJSON string          `json:"-"`
	State          string          `json:"status"`
	Seed           int64           `json:"-"`
	RunID          int64           `json:"runId"`
	LeaseOwner     string          `json:"-"`
	LeaseUntil     *time.Time      `json:"-"`
	AttemptCount   int             `json:"attemptCount"`
	NextRunAt      time.Time       `json:"-"`
	ErrorJSON      string          `json:"-"`
	Error          json.RawMessage `json:"error,omitempty" gorm:"-"`
	TaskID         int64           `json:"taskId,omitempty" gorm:"-"`
	RevisionID     int64           `json:"revisionId,omitempty" gorm:"-"`
	GeneratedCount int             `json:"generatedCount" gorm:"-"`
}

func (Partition) TableName() string { return "review_suggestion_partitions" }

type TaskLink struct {
	TaskStatus          string          `json:"taskStatus" gorm:"-"`
	ActiveRevisionID    *int64          `json:"activeRevisionId" gorm:"-"`
	PublishedRevisionID *int64          `json:"publishedRevisionId" gorm:"-"`
	ID                  int64           `json:"id" gorm:"primaryKey"`
	SuggestionID        int64           `json:"suggestionId"`
	PartitionID         int64           `json:"partitionId" gorm:"uniqueIndex"`
	TaskID              int64           `json:"taskId" gorm:"uniqueIndex"`
	GeneratedRevisionID int64           `json:"generatedRevisionId"`
	TargetMapJSON       string          `json:"-"`
	TargetMap           json.RawMessage `json:"targetMap" gorm:"-"`
}

func (TaskLink) TableName() string { return "review_suggestion_tasks" }

type Command struct {
	ID             int64  `gorm:"primaryKey"`
	ChildID        int64  `gorm:"uniqueIndex:uq_rs_command"`
	Operation      string `gorm:"uniqueIndex:uq_rs_command"`
	IdempotencyKey string `gorm:"uniqueIndex:uq_rs_command"`
	RequestHash    string
	SuggestionID   int64
	RunID          *int64
	ResponseJSON   string
}

func (Command) TableName() string { return "review_suggestion_commands" }

// Migrate adds task-owned tables only; shared learning schemas are deliberately untouched.
func Migrate(g *gorm.DB) error {
	if e := g.AutoMigrate(&Suggestion{}, &Target{}, &Evidence{}, &Run{}, &Partition{}, &TaskLink{}, &Command{}); e != nil {
		return e
	}
	for _, sql := range []string{`CREATE UNIQUE INDEX IF NOT EXISTS uq_review_suggestion_open_content ON review_suggestions(child_id,content_hash) WHERE lifecycle='open'`, `CREATE UNIQUE INDEX IF NOT EXISTS uq_review_suggestion_active_run ON review_suggestion_runs(suggestion_id) WHERE state IN ('queued','running')`, `CREATE INDEX IF NOT EXISTS idx_review_suggestion_list ON review_suggestions(child_id,created_at DESC,id DESC)`, `CREATE INDEX IF NOT EXISTS idx_review_suggestion_claim ON review_suggestion_partitions(state,next_run_at,lease_until)`} {
		if e := g.Exec(sql).Error; e != nil {
			return e
		}
	}
	return migrateReferences(g)
}
