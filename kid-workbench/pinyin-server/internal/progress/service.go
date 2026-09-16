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
	KpID       int64           `json:"kpId"`
	Letter     string          `json:"letter"`
	ModuleCode string          `json:"moduleCode"`
	ModuleName string          `json:"moduleName"`
	Status     string          `json:"status"`
	Skills     []SkillProgress `json:"skills"`
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
	var rows []struct {
		KpID                           int64
		Letter, ModuleCode, ModuleName string
	}
	if err := s.db.WithContext(ctx).Table("knowledge_points kp").Select("kp.id AS kp_id,kp.title AS letter,m.code AS module_code,m.name AS module_name").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects s ON s.id=m.subject_id").Where("s.code = ?", "pinyin").Where("m.code <> 'syllables' OR EXISTS (SELECT 1 FROM pinyin_syllable_links link WHERE link.kp_id=kp.id AND link.enabled = ?)", true).Order("m.order_no,kp.order_no,kp.id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	type skillRow struct {
		KpID                      int64
		SkillCode, Status         string
		Attempts, Correct, Streak int
		DueAt                     *time.Time
	}
	var skills []skillRow
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.KpID)
	}
	if len(ids) > 0 {
		if err := s.db.WithContext(ctx).Table("mastery_skills").Where("child_id = ? AND kp_id IN ?", childID, ids).Scan(&skills).Error; err != nil {
			return nil, err
		}
	}
	byKP := map[int64]map[string]skillRow{}
	for _, skill := range skills {
		if byKP[skill.KpID] == nil {
			byKP[skill.KpID] = map[string]skillRow{}
		}
		byKP[skill.KpID][skill.SkillCode] = skill
	}
	result := make([]ItemProgress, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		item := ItemProgress{KpID: row.KpID, Letter: row.Letter, ModuleCode: row.ModuleCode, ModuleName: row.ModuleName, Skills: []SkillProgress{}}
		statuses := []mastery.Status{}
		for _, code := range mastery.SkillsFor("pinyin", row.ModuleCode) {
			skill := byKP[row.KpID][code]
			status := mastery.StatusNotStarted
			if skill.Status != "" {
				state := mastery.State{Status: mastery.Status(skill.Status)}
				if skill.DueAt != nil {
					state.DueAt = *skill.DueAt
				}
				status = mastery.Display(state, now)
			}
			accuracy := 0.0
			if skill.Attempts > 0 {
				accuracy = float64(skill.Correct) / float64(skill.Attempts)
			}
			item.Skills = append(item.Skills, SkillProgress{Code: code, Status: string(status), Attempts: skill.Attempts, Accuracy: accuracy, Streak: skill.Streak})
			statuses = append(statuses, status)
		}
		item.Status = string(mastery.RollupSkills(statuses))
		result = append(result, item)
	}
	return result, nil
}
