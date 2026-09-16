package home

import (
	"context"
	"errors"
	"gorm.io/gorm"
)

var ErrChildNotFound = errors.New("child not found")

type Child struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Grade     string `json:"grade"`
	AvatarURL string `json:"avatarUrl"`
	Flowers   int    `json:"flowers"`
}
type CurrentPlan struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	TargetCount int    `json:"targetCount"`
	DoneCount   int    `json:"doneCount"`
}
type ModuleSummary struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Mastered int    `json:"mastered"`
	Total    int    `json:"total"`
}
type Summary struct {
	Child       Child           `json:"child"`
	CurrentPlan *CurrentPlan    `json:"currentPlan"`
	DueCount    int             `json:"dueCount"`
	Modules     []ModuleSummary `json:"modules"`
}
type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db} }
func (s *Service) Get(ctx context.Context, childID int64) (Summary, error) {
	var out Summary
	q := s.db.WithContext(ctx).Table("children").Where("id=?", childID).Take(&out.Child)
	if errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return Summary{}, ErrChildNotFound
	}
	if q.Error != nil {
		return Summary{}, q.Error
	}
	var plan CurrentPlan
	q = s.db.WithContext(ctx).Table("study_plans").Where("child_id=? AND subject_code=? AND status<>?", childID, "english", "done").Order("plan_date DESC,seq_no DESC").First(&plan)
	if q.Error == nil {
		out.CurrentPlan = &plan
	} else if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return Summary{}, q.Error
	}
	var due int64
	if err := s.db.WithContext(ctx).Table("mastery_states ms").Joins("JOIN knowledge_points kp ON kp.id=ms.kp_id").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects sub ON sub.id=m.subject_id").Where("ms.child_id=? AND sub.code=? AND (ms.status='review_due' OR (ms.status='mastered' AND ms.due_at IS NOT NULL AND ms.due_at<=CURRENT_TIMESTAMP))", childID, "english").Count(&due).Error; err != nil {
		return Summary{}, err
	}
	out.DueCount = int(due)
	if err := s.db.WithContext(ctx).Table("modules m").Select("m.code,m.name,COUNT(kp.id) AS total,SUM(CASE WHEN ms.status IN ('mastered','review_due') THEN 1 ELSE 0 END) AS mastered").Joins("JOIN subjects sub ON sub.id=m.subject_id").Joins("JOIN knowledge_points kp ON kp.module_id=m.id").Joins("LEFT JOIN mastery_states ms ON ms.kp_id=kp.id AND ms.child_id=?", childID).Where("sub.code=?", "english").Group("m.id,m.code,m.name,m.order_no").Order("m.order_no,m.id").Scan(&out.Modules).Error; err != nil {
		return Summary{}, err
	}
	return out, nil
}
