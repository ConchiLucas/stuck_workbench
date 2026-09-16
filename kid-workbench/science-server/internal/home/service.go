package home

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/conchi/study-science/internal/plan"
)

var ErrChildNotFound = errors.New("child not found")

type Child struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Grade     string `json:"grade"`
	AvatarURL string `json:"avatarUrl"`
	Flowers   int    `json:"flowers"`
}

type ModuleSummary struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Mastered int    `json:"mastered"`
	Total    int    `json:"total"`
}

type Summary struct {
	Child         Child           `json:"child"`
	CurrentPlan   *plan.StudyPlan `json:"currentPlan"`
	DueCount      int             `json:"dueCount"`
	ExploredCount int             `json:"exploredCount"`
	Modules       []ModuleSummary `json:"modules"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, childID int64) (Summary, error) {
	var result Summary
	query := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Take(&result.Child)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return Summary{}, ErrChildNotFound
	}
	if query.Error != nil {
		return Summary{}, query.Error
	}

	var current plan.StudyPlan
	query = s.db.WithContext(ctx).Where("child_id = ? AND subject_code = ? AND status <> ?", childID, "science", "completed").
		Order("plan_date DESC, seq_no DESC").First(&current)
	if query.Error == nil {
		result.CurrentPlan = &current
	} else if !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return Summary{}, query.Error
	}

	var dueCount int64
	if err := s.db.WithContext(ctx).Table("mastery_states ms").
		Joins("JOIN knowledge_points kp ON kp.id = ms.kp_id").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects sub ON sub.id = m.subject_id").
		Joins("JOIN science_assets sa ON sa.kp_id = kp.id AND sa.review_status = 'published'").
		Where("ms.child_id = ? AND sub.code = ? AND (ms.status = ? OR (ms.status = ? AND ms.due_at <= CURRENT_TIMESTAMP))", childID, "science", "review_due", "mastered").
		Count(&dueCount).Error; err != nil {
		return Summary{}, err
	}
	result.DueCount = int(dueCount)
	var exploredCount int64
	if err := s.db.WithContext(ctx).Table("mastery_states ms").
		Joins("JOIN knowledge_points kp ON kp.id = ms.kp_id").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects sub ON sub.id = m.subject_id").
		Joins("JOIN science_assets sa ON sa.kp_id = kp.id AND sa.review_status = 'published'").
		Where("ms.child_id = ? AND sub.code = ? AND ms.attempts > 0", childID, "science").Count(&exploredCount).Error; err != nil {
		return Summary{}, err
	}
	result.ExploredCount = int(exploredCount)

	if err := s.db.WithContext(ctx).Table("modules m").
		Select(`m.code, m.name, COUNT(kp.id) AS total,
			SUM(CASE WHEN ms.status = 'mastered' THEN 1 ELSE 0 END) AS mastered`).
		Joins("JOIN subjects sub ON sub.id = m.subject_id").
		Joins("JOIN knowledge_points kp ON kp.module_id = m.id").
		Joins("JOIN science_assets sa ON sa.kp_id = kp.id AND sa.review_status = 'published'").
		Joins("LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?", childID).
		Where("sub.code = ?", "science").Group("m.id, m.code, m.name, m.order_no").
		Order("m.order_no, m.id").Scan(&result.Modules).Error; err != nil {
		return Summary{}, err
	}
	return result, nil
}
