package model

import "time"

type Child struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	Name      string     `json:"name"`
	Grade     string     `json:"grade"`
	AvatarURL string     `gorm:"column:avatar_url" json:"avatar_url"`
	Birthday  *time.Time `json:"birthday"`
	Flowers   int        `json:"flowers"`
	CreatedAt time.Time  `json:"created_at"`
}

func (Child) TableName() string { return "children" }

type KnowledgePoint struct {
	ID         int64 `gorm:"primaryKey"`
	ModuleID   int64
	Code       string
	Title      string
	Payload    string
	Difficulty int
	OrderNo    int
}

func (KnowledgePoint) TableName() string { return "knowledge_points" }

type Question struct {
	ID         int64 `gorm:"primaryKey"`
	KpID       int64 `gorm:"column:kp_id"`
	Code       string
	Type       string
	Stem       string
	Options    string
	Answer     string
	Visual     string
	Speech     string
	MediaURL   string `gorm:"column:media_url"`
	Difficulty int
}

func (Question) TableName() string { return "questions" }

type Attempt struct {
	ID         int64 `gorm:"primaryKey"`
	ChildID    int64
	KpID       int64 `gorm:"column:kp_id"`
	QuestionID *int64
	IsCorrect  bool
	CostMs     int
	Source     string
	ClientID   string `gorm:"column:client_id"`
	PlanItemID *int64 `gorm:"column:plan_item_id"`
	Selected   string `gorm:"column:selected"`
	CreatedAt  time.Time
}

func (Attempt) TableName() string { return "attempts" }

type MasteryState struct {
	ChildID      int64 `gorm:"primaryKey"`
	KpID         int64 `gorm:"primaryKey;column:kp_id"`
	Status       string
	Attempts     int
	Correct      int
	Streak       int
	BestStreak   int
	Ease         float64
	IntervalDays int
	DueAt        *time.Time
	FirstSeenAt  *time.Time
	MasteredAt   *time.Time
	UpdatedAt    time.Time
}

func (MasteryState) TableName() string { return "mastery_states" }

type MasterySkill struct {
	ChildID      int64  `gorm:"primaryKey"`
	KpID         int64  `gorm:"primaryKey;column:kp_id"`
	SkillCode    string `gorm:"primaryKey;column:skill_code"`
	Status       string
	Attempts     int
	Correct      int
	Streak       int
	BestStreak   int
	Ease         float64
	IntervalDays int
	DueAt        *time.Time
	FirstSeenAt  *time.Time
	MasteredAt   *time.Time
	UpdatedAt    time.Time
}

func (MasterySkill) TableName() string { return "mastery_skills" }

type DailyStat struct {
	ChildID       int64     `gorm:"primaryKey"`
	StatDate      time.Time `gorm:"primaryKey;type:date"`
	PracticeSec   int
	Attempts      int
	Correct       int
	NewlyMastered int
	ReviewDone    int
	CheckedIn     bool
}

func (DailyStat) TableName() string { return "daily_stats" }

type FlowerLedger struct {
	ID        int64 `gorm:"primaryKey"`
	ChildID   int64
	Delta     int
	Reason    string
	RefType   string
	RefID     *int64
	CreatedAt time.Time
}

func (FlowerLedger) TableName() string { return "flower_ledger" }

// PlanItem is shared by subject-specific practice servers. OptionOrder stores
// displayed-option indexes so a plan remains stable across reloads.
type PlanItem struct {
	ID                     int64 `gorm:"primaryKey"`
	PlanID                 int64
	Seq                    int
	KpID                   int64 `gorm:"column:kp_id"`
	QuestionID             int64
	Bucket                 string
	Status                 string
	Tries                  int
	CostMs                 int
	Picks                  string
	OptionOrder            string `gorm:"column:option_order"`
	QuestionStem           string `gorm:"column:question_stem"`
	QuestionOptions        string `gorm:"column:question_options"`
	QuestionAnswer         string `gorm:"column:question_answer"`
	QuestionVisual         string `gorm:"column:question_visual"`
	QuestionSpeech         string `gorm:"column:question_speech"`
	QuestionSnapshot       string `gorm:"column:question_snapshot;type:json"`
	Explanation            string `gorm:"column:explanation"`
	ContentSnapshotVersion int    `gorm:"column:content_snapshot_version"`
	AnsweredAt             *time.Time
}

func (PlanItem) TableName() string { return "plan_items" }
