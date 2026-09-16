package model

import (
	"time"

	learningmodel "github.com/conchi/study-learning/model"
)

type User struct {
	ID           int64 `gorm:"primaryKey"`
	Phone        string
	Nickname     string
	PasswordHash string
	CreatedAt    time.Time
}

func (User) TableName() string { return "users" }

type Child = learningmodel.Child

type Subject struct {
	ID      int64 `gorm:"primaryKey"`
	Code    string
	Name    string
	Icon    string
	OrderNo int
	// QuizEnabled 标记该学科能否自动出题。科普/古诗这类题面依赖真实内容库，
	// 暂时进不了答题计划，但在家长看板照常显示。
	QuizEnabled bool
}

func (Subject) TableName() string { return "subjects" }

type Module struct {
	ID        int64 `gorm:"primaryKey"`
	SubjectID int64
	Code      string
	Name      string
	OrderNo   int
}

func (Module) TableName() string { return "modules" }

type KnowledgePoint = learningmodel.KnowledgePoint
type Question = learningmodel.Question
type Attempt = learningmodel.Attempt
type MasteryState = learningmodel.MasteryState
type MasterySkill = learningmodel.MasterySkill
type DailyStat = learningmodel.DailyStat

type DailyTask struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	ChildID       int64     `json:"child_id"`
	TaskDate      time.Time `gorm:"type:date" json:"task_date"`
	Title         string    `json:"title"`
	SubjectID     *int64    `json:"subject_id"`
	TargetCount   int       `json:"target_count"`
	DoneCount     int       `json:"done_count"`
	RewardFlowers int       `json:"reward_flowers"`
	Status        string    `json:"status"`
}

func (DailyTask) TableName() string { return "daily_tasks" }

// StudyPlan 是某个孩子某一天的一份答题计划。
// SeqNo 为 1 是当天的主计划，2 及以上是家长手动加的"加餐"。
type StudyPlan struct {
	ID      int64 `gorm:"primaryKey"`
	ChildID int64
	// PlanDate 用字符串存 YYYY-MM-DD。两个方言对 DATE 的读写行为不一致，
	// 统一走文本避免时区把日期带偏一天，与 daily_stats 的处理方式一致。
	PlanDate     string `gorm:"type:date"`
	SeqNo        int
	SubjectCode  string
	PlanKind     string
	ModuleCode   string
	StageCode    string
	Status       string
	TargetCount  int
	DoneCount    int
	CorrectCount int
	Stars        int
	DurationSec  int
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

func (StudyPlan) TableName() string { return "study_plans" }

// PlanItem 是计划里的一道题。生成时就把 question_id 固定下来，
// 这样中途刷新页面、换设备继续，看到的都是同一份题。
type PlanItem = learningmodel.PlanItem

type Reward struct {
	ID      int64  `gorm:"primaryKey" json:"id"`
	ChildID int64  `json:"child_id"`
	Name    string `json:"name"`
	Cost    int    `json:"cost"`
	Stock   int    `json:"stock"`
}

func (Reward) TableName() string { return "rewards" }

type FlowerLedger = learningmodel.FlowerLedger
