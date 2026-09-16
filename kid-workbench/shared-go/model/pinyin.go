package model

import "time"

type PinyinSyllableLink struct {
	AssetID      int64  `gorm:"primaryKey;autoIncrement:false"`
	KpID         int64  `gorm:"column:kp_id;uniqueIndex"`
	InitialText  string `gorm:"uniqueIndex:pinyin_identity"`
	FinalText    string `gorm:"uniqueIndex:pinyin_identity"`
	Tone         int    `gorm:"uniqueIndex:pinyin_identity"`
	SyllableText string
	Enabled      bool
	SyncedAt     time.Time
}

func (PinyinSyllableLink) TableName() string { return "pinyin_syllable_links" }

type PinyinQuizInstance struct {
	ID              string `gorm:"primaryKey"`
	ChildID         int64  `gorm:"index"`
	KpID            int64  `gorm:"column:kp_id"`
	SkillCode       string
	SnapshotVersion int
	PublicSnapshot  string
	AnswerOptionID  string
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

func (PinyinQuizInstance) TableName() string { return "pinyin_quiz_instances" }

type PinyinAnswerReceipt struct {
	ChildID          int64  `gorm:"primaryKey;autoIncrement:false"`
	ClientID         string `gorm:"primaryKey"`
	InstanceID       string `gorm:"uniqueIndex"`
	AttemptID        int64  `gorm:"uniqueIndex"`
	SkillCode        string
	SelectedOptionID string
	ResponseSnapshot string
	CreatedAt        time.Time
}

func (PinyinAnswerReceipt) TableName() string { return "pinyin_answer_receipts" }

type PinyinMasteryMilestone struct {
	ChildID          int64 `gorm:"primaryKey;autoIncrement:false"`
	KpID             int64 `gorm:"primaryKey;autoIncrement:false;column:kp_id"`
	RuleVersion      int   `gorm:"primaryKey;autoIncrement:false"`
	FirstCompletedAt time.Time
}

func (PinyinMasteryMilestone) TableName() string { return "pinyin_mastery_milestones" }

type PinyinUpgradeAudit struct {
	ChildID        int64 `gorm:"primaryKey;autoIncrement:false"`
	KpID           int64 `gorm:"primaryKey;autoIncrement:false;column:kp_id"`
	RuleVersion    int   `gorm:"primaryKey;autoIncrement:false"`
	BeforeSnapshot string
	HadReward      bool
	AppliedAt      time.Time
}

func (PinyinUpgradeAudit) TableName() string { return "pinyin_upgrade_audits" }
