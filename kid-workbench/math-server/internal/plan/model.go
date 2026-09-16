package plan

import (
	"encoding/json"
	"time"
)

type StudyPlan struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	ChildID      int64      `json:"-"`
	PlanDate     string     `gorm:"type:date" json:"planDate"`
	SeqNo        int        `json:"seqNo"`
	SubjectCode  string     `json:"subjectCode"`
	PlanKind     string     `json:"planKind"`
	ModuleCode   string     `json:"moduleCode,omitempty"`
	StageCode    string     `json:"stageCode,omitempty"`
	Status       string     `json:"status"`
	TargetCount  int        `json:"targetCount"`
	DoneCount    int        `json:"doneCount"`
	CorrectCount int        `json:"correctCount"`
	Stars        int        `json:"stars"`
	DurationSec  int        `json:"durationSec"`
	CreatedAt    time.Time  `json:"createdAt"`
	StartedAt    *time.Time `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt"`
}

func (StudyPlan) TableName() string { return "study_plans" }

type Candidate struct {
	QuestionID int64
	KpID       int64
	ModuleCode string
	Code       string
	Stem       string
	Options    string
	Answer     string
	Visual     string
	MediaURL   string
	Bucket     string
	Priority   int
	A          int
	B          int
}

type Visual struct {
	Kind       string `json:"kind"`
	A          int    `json:"a,omitempty"`
	B          int    `json:"b,omitempty"`
	Operator   string `json:"operator,omitempty"`
	LeftCount  int    `json:"leftCount,omitempty"`
	RightCount int    `json:"rightCount,omitempty"`
	Object     string `json:"object,omitempty"`
	Shape      string `json:"shape,omitempty"`
}

type QuestionSnapshot struct {
	QuestionID     int64           `json:"questionId"`
	Code           string          `json:"code"`
	Stem           string          `json:"stem"`
	Options        json.RawMessage `json:"options"`
	AnswerIndex    int             `json:"answerIndex"`
	Visual         Visual          `json:"visual"`
	AudioObjectKey string          `json:"audioObjectKey"`
}

type QuestionDTO struct {
	QuestionID int64           `json:"questionId"`
	Code       string          `json:"code"`
	Stem       string          `json:"stem"`
	Options    json.RawMessage `json:"options"`
	Visual     Visual          `json:"visual"`
	AudioURL   string          `json:"audioUrl"`
}

type Item struct {
	ID       int64       `json:"itemId"`
	Seq      int         `json:"seq"`
	KpID     int64       `json:"kpId"`
	Bucket   string      `json:"bucket"`
	Status   string      `json:"status"`
	Tries    int         `json:"tries"`
	Question QuestionDTO `json:"question"`
}

type Detail struct {
	Plan  StudyPlan `json:"plan"`
	Items []Item    `json:"items"`
}

type CreateInput struct {
	Kind       string `json:"kind"`
	ModuleCode string `json:"moduleCode"`
	StageCode  string `json:"stageCode"`
}
