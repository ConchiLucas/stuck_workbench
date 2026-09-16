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

type Question struct {
	ID      int64           `json:"id"`
	Code    string          `json:"code"`
	Type    string          `json:"type"`
	Stem    string          `json:"stem"`
	Options json.RawMessage `json:"options"`
	Visual  json.RawMessage `json:"visual"`
	Speech  json.RawMessage `json:"speech"`
}

type Item struct {
	ID          int64    `json:"id"`
	Seq         int      `json:"seq"`
	KpID        int64    `json:"kpId"`
	Tries       int      `json:"tries"`
	Phrase      string   `json:"phrase"`
	MeaningZh   string   `json:"meaningZh"`
	Scene       string   `json:"scene"`
	ReplyTo     string   `json:"replyTo"`
	Bucket      string   `json:"bucket"`
	Status      string   `json:"status"`
	Picks       string   `json:"picks"`
	OptionOrder string   `json:"optionOrder"`
	Question    Question `json:"question"`
}

type Detail struct {
	Plan  StudyPlan `json:"plan"`
	Items []Item    `json:"items"`
}

type CreateInput struct {
	Mode         string `json:"mode"`
	QuestionCode string `json:"questionCode"`
	Count        int    `json:"count"`
}
