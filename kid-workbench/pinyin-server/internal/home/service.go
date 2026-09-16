package home

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/conchi/pinyin-server/internal/plan"
	"github.com/conchi/pinyin-server/internal/progress"
	"github.com/conchi/study-learning/mastery"
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
	Child       Child           `json:"child"`
	CurrentPlan *plan.StudyPlan `json:"currentPlan"`
	DueCount    int             `json:"dueCount"`
	Modules     []ModuleSummary `json:"modules"`
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
	query = s.db.WithContext(ctx).Where("child_id = ? AND subject_code = ? AND status <> ?", childID, "pinyin", "done").
		Order("plan_date DESC, seq_no DESC").First(&current)
	if query.Error == nil {
		result.CurrentPlan = &current
	} else if !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return Summary{}, query.Error
	}

	items, err := progress.NewService(s.db).Get(ctx, childID)
	if err != nil {
		return Summary{}, err
	}
	result.Modules = []ModuleSummary{}
	moduleIndex := map[string]int{}
	for _, item := range items {
		index, exists := moduleIndex[item.ModuleCode]
		if !exists {
			index = len(result.Modules)
			moduleIndex[item.ModuleCode] = index
			result.Modules = append(result.Modules, ModuleSummary{Code: item.ModuleCode, Name: item.ModuleName})
		}
		result.Modules[index].Total++
		if mastery.IsSkillDone(mastery.Status(item.Status)) {
			result.Modules[index].Mastered++
		}
		if item.Status == string(mastery.StatusReviewDue) {
			result.DueCount++
		}
	}
	return result, nil
}
