package home

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrChildNotFound = errors.New("child not found")

type ChildSummary struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Flowers int    `json:"flowers"`
}

type PlanSummary struct {
	ID           int64  `json:"id"`
	Status       string `json:"status"`
	TargetCount  int    `json:"targetCount"`
	DoneCount    int    `json:"doneCount"`
	CorrectCount int    `json:"correctCount"`
}

type ModuleSummary struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	ItemCount     int    `json:"itemCount"`
	MasteredCount int    `json:"masteredCount"`
}

type Home struct {
	Child    ChildSummary    `json:"child"`
	Plan     *PlanSummary    `json:"plan"`
	DueCount int             `json:"dueCount"`
	Modules  []ModuleSummary `json:"modules"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, childID int64) (Home, error) {
	var child ChildSummary
	result := s.db.WithContext(ctx).Table("children").Select("id, name, flowers").Where("id = ?", childID).Scan(&child)
	if result.Error != nil {
		return Home{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Home{}, ErrChildNotFound
	}
	out := Home{Child: child}
	var active PlanSummary
	query := s.db.WithContext(ctx).Table("study_plans").
		Select("id, status, target_count, done_count, correct_count").
		Where("child_id = ? AND subject_code = ? AND plan_kind <> '' AND status IN ?", childID, "math", []string{"pending", "doing"}).
		Order("created_at DESC, id DESC").Limit(1).Scan(&active)
	if query.Error != nil {
		return Home{}, query.Error
	}
	if query.RowsAffected > 0 {
		out.Plan = &active
	}
	var dueCount int64
	if err := s.db.WithContext(ctx).Table("mastery_states ms").
		Joins("JOIN knowledge_points kp ON kp.id = ms.kp_id").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Where("ms.child_id = ? AND s.code = ?", childID, "math").
		Where("ms.status IN ? OR (ms.status = ? AND ms.due_at <= ?)", []string{"review_due", "shaky"}, "mastered", time.Now()).
		Count(&dueCount).Error; err != nil {
		return Home{}, err
	}
	out.DueCount = int(dueCount)
	if err := s.db.WithContext(ctx).Table("modules m").
		Select(`m.code, m.name, COUNT(kp.id) AS item_count,
			SUM(CASE WHEN ms.status IN ('mastered','review_due') THEN 1 ELSE 0 END) AS mastered_count`).
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Joins("JOIN knowledge_points kp ON kp.module_id = m.id").
		Joins("LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?", childID).
		Where("s.code = ? AND m.code IN ?", "math", []string{"add10", "sub10", "shape"}).
		Group("m.id, m.code, m.name, m.order_no").Order("m.order_no, m.id").Scan(&out.Modules).Error; err != nil {
		return Home{}, err
	}
	return out, nil
}
