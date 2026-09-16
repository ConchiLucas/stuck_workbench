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
	KpID      int64           `json:"kpId"`
	Character string          `json:"character"`
	Status    string          `json:"status"`
	Skills    []SkillProgress `json:"skills"`
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
		KpID      int64
		Character string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("knowledge_points kp").
		Select("kp.id AS kp_id, kp.title AS character").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Where("s.code = ?", "literacy").Order("m.order_no, kp.order_no, kp.id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]ItemProgress, 0, len(rows))
	for _, row := range rows {
		skills := make([]SkillProgress, 0, len(mastery.LiteracySkills))
		statuses := make([]mastery.Status, 0, len(mastery.LiteracySkills))
		for _, code := range mastery.LiteracySkills {
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
				state := mastery.State{Status: mastery.Status(skill.Status)}
				if skill.DueAt != nil {
					state.DueAt = *skill.DueAt
				}
				status = mastery.Display(state, time.Now())
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
			KpID: row.KpID, Character: row.Character,
			Status: string(mastery.RollupSkills(statuses)), Skills: skills,
		})
	}
	return result, nil
}
