package progress

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/conchi/study-learning/mastery"
	"gorm.io/gorm"
	"time"
)

var ErrChildNotFound = errors.New("child not found")

type SkillProgress struct {
	Code     string  `json:"code"`
	Status   string  `json:"status"`
	Attempts int     `json:"attempts"`
	Accuracy float64 `json:"accuracy"`
	Streak   int     `json:"streak"`
}
type WordProgress struct {
	KpID      int64           `json:"kpId"`
	Word      string          `json:"word"`
	MeaningZh string          `json:"meaningZh"`
	Status    string          `json:"status"`
	Skills    []SkillProgress `json:"skills"`
}
type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, childID int64) ([]WordProgress, error) {
	var count int64
	if err := s.db.WithContext(ctx).Table("children").Where("id=?", childID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrChildNotFound
	}
	type wordRow struct {
		KpID          int64
		Word, Payload string
	}
	var rows []wordRow
	if err := s.db.WithContext(ctx).Table("knowledge_points kp").Select("kp.id AS kp_id,kp.title AS word,kp.payload").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects sub ON sub.id=m.subject_id").Where("sub.code=?", "english").Order("m.order_no,kp.order_no,kp.id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]WordProgress, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		var meta struct {
			MeaningZh string `json:"meaningZh"`
		}
		_ = json.Unmarshal([]byte(row.Payload), &meta)
		skills := make([]SkillProgress, 0, len(mastery.EnglishSkills))
		statuses := make([]mastery.Status, 0, len(mastery.EnglishSkills))
		for _, code := range mastery.EnglishSkills {
			var sk struct {
				Status                    string
				Attempts, Correct, Streak int
				DueAt                     *time.Time
			}
			q := s.db.WithContext(ctx).Table("mastery_skills").Where("child_id=? AND kp_id=? AND skill_code=?", childID, row.KpID, code).Take(&sk)
			if q.Error != nil && !errors.Is(q.Error, gorm.ErrRecordNotFound) {
				return nil, q.Error
			}
			status := mastery.StatusNotStarted
			if q.Error == nil && sk.Status != "" {
				state := mastery.State{Status: mastery.Status(sk.Status)}
				if sk.DueAt != nil {
					state.DueAt = *sk.DueAt
				}
				status = mastery.Display(state, now)
			}
			accuracy := 0.0
			if sk.Attempts > 0 {
				accuracy = float64(sk.Correct) / float64(sk.Attempts)
			}
			statuses = append(statuses, status)
			skills = append(skills, SkillProgress{code, string(status), sk.Attempts, accuracy, sk.Streak})
		}
		out = append(out, WordProgress{row.KpID, row.Word, meta.MeaningZh, string(mastery.RollupSkills(statuses)), skills})
	}
	return out, nil
}
