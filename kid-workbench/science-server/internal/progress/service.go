package progress

import (
	"context"
	"errors"
	"time"

	"github.com/conchi/study-learning/mastery"
	"gorm.io/gorm"
)

var ErrChildNotFound = errors.New("child not found")

type SkillProgress struct {
	Code     string  `json:"code"`
	Status   string  `json:"status"`
	Attempts int     `json:"attempts"`
	Accuracy float64 `json:"accuracy"`
	Streak   int     `json:"streak"`
}

type ItemProgress struct {
	KpID   int64           `json:"kpId"`
	Title  string          `json:"title"`
	Status string          `json:"status"`
	Skills []SkillProgress `json:"skills"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, childID int64) ([]ItemProgress, error) {
	var count int64
	if err := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrChildNotFound
	}

	type itemRow struct {
		KpID  int64
		Title string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("knowledge_points kp").
		Select("kp.id AS kp_id, kp.title").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Joins("JOIN science_assets sa ON sa.kp_id = kp.id AND sa.review_status = 'published'").
		Where("s.code = ?", "science").Order("m.order_no, kp.order_no, kp.id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]ItemProgress, 0, len(rows))
	for _, row := range rows {
		skills := make([]SkillProgress, 0, len(mastery.ScienceSkills))
		statuses := make([]mastery.Status, 0, len(mastery.ScienceSkills))
		for _, code := range mastery.ScienceSkills {
			var skill struct {
				Status   string
				Attempts int
				Correct  int
				Streak   int
				DueAt    *time.Time
			}
			query := s.db.WithContext(ctx).Table("mastery_skills").
				Where("child_id = ? AND kp_id = ? AND skill_code = ?", childID, row.KpID, code).
				Take(&skill)
			status := mastery.StatusNotStarted
			if query.Error != nil && !errors.Is(query.Error, gorm.ErrRecordNotFound) {
				return nil, query.Error
			}
			if query.Error == nil && skill.Status != "" {
				status = mastery.Status(skill.Status)
				if status == mastery.StatusMastered && skill.DueAt != nil && skill.DueAt.Before(time.Now()) {
					status = mastery.StatusReviewDue
				}
			}
			accuracy := 0.0
			if skill.Attempts > 0 {
				accuracy = float64(skill.Correct) / float64(skill.Attempts)
			}
			statuses = append(statuses, status)
			skills = append(skills, SkillProgress{
				Code: code, Status: string(status), Attempts: skill.Attempts,
				Accuracy: accuracy, Streak: skill.Streak,
			})
		}
		result = append(result, ItemProgress{
			KpID: row.KpID, Title: row.Title,
			Status: string(mastery.RollupSkills(statuses)), Skills: skills,
		})
	}
	return result, nil
}
