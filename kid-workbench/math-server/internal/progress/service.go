package progress

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/conchi/study-learning/mastery"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/catalog"
)

var ErrChildNotFound = errors.New("child not found")

type SkillProgress struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}

type ItemProgress struct {
	KpID   int64           `json:"kpId"`
	Title  string          `json:"title"`
	Status string          `json:"status"`
	Skills []SkillProgress `json:"skills"`
}

type StageProgress struct {
	Code  string         `json:"code"`
	Items []ItemProgress `json:"items"`
}

type ModuleProgress struct {
	Code   string          `json:"code"`
	Name   string          `json:"name"`
	Stages []StageProgress `json:"stages"`
}

type Result struct {
	Modules []ModuleProgress `json:"modules"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, childID int64) (Result, error) {
	var count int64
	if err := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Count(&count).Error; err != nil {
		return Result{}, err
	}
	if count == 0 {
		return Result{}, ErrChildNotFound
	}
	type itemRow struct {
		KpID       int64
		Title      string
		Payload    string
		ModuleCode string
		ModuleName string
	}
	var rows []itemRow
	err := s.db.WithContext(ctx).Table("knowledge_points kp").
		Select("kp.id AS kp_id, kp.title, kp.payload, m.code AS module_code, m.name AS module_name").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Where("s.code = ? AND m.code IN ?", "math", []string{"add10", "sub10", "shape"}).
		Order("m.order_no, kp.order_no, kp.id").Scan(&rows).Error
	if err != nil {
		return Result{}, err
	}

	result := Result{Modules: []ModuleProgress{}}
	moduleIndex := map[string]int{}
	stageIndex := map[string]map[string]int{}
	for _, row := range rows {
		stageCode, err := progressStage(row.ModuleCode, row.Payload)
		if err != nil {
			return Result{}, err
		}
		mi, ok := moduleIndex[row.ModuleCode]
		if !ok {
			mi = len(result.Modules)
			moduleIndex[row.ModuleCode] = mi
			stageIndex[row.ModuleCode] = map[string]int{}
			result.Modules = append(result.Modules, ModuleProgress{Code: row.ModuleCode, Name: row.ModuleName})
		}
		si, ok := stageIndex[row.ModuleCode][stageCode]
		if !ok {
			si = len(result.Modules[mi].Stages)
			stageIndex[row.ModuleCode][stageCode] = si
			result.Modules[mi].Stages = append(result.Modules[mi].Stages, StageProgress{Code: stageCode})
		}
		item, err := s.itemProgress(ctx, childID, row.KpID, row.Title, row.ModuleCode)
		if err != nil {
			return Result{}, err
		}
		result.Modules[mi].Stages[si].Items = append(result.Modules[mi].Stages[si].Items, item)
	}
	return result, nil
}

func progressStage(moduleCode, payloadRaw string) (string, error) {
	if moduleCode == "shape" {
		return catalog.StageFor(moduleCode, 0, 0), nil
	}
	var payload struct{ A, B int }
	if err := json.Unmarshal([]byte(payloadRaw), &payload); err != nil {
		return "", err
	}
	return catalog.StageFor(moduleCode, payload.A, payload.B), nil
}

func (s *Service) itemProgress(ctx context.Context, childID, kpID int64, title, moduleCode string) (ItemProgress, error) {
	codes := mastery.SkillsFor("math", moduleCode)
	statuses := make([]mastery.Status, 0, len(codes))
	skills := make([]SkillProgress, 0, len(codes))
	for _, code := range codes {
		var row struct {
			Status string
			DueAt  *time.Time
		}
		query := s.db.WithContext(ctx).Table("mastery_skills").
			Select("status, due_at").
			Where("child_id = ? AND kp_id = ? AND skill_code = ?", childID, kpID, code).Take(&row)
		status := mastery.StatusNotStarted
		if query.Error != nil && !errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return ItemProgress{}, query.Error
		}
		if query.Error == nil && row.Status != "" {
			state := mastery.State{Status: mastery.Status(row.Status)}
			if row.DueAt != nil {
				state.DueAt = *row.DueAt
			}
			status = mastery.Display(state, time.Now())
		}
		statuses = append(statuses, status)
		skills = append(skills, SkillProgress{Code: code, Status: string(status)})
	}
	return ItemProgress{KpID: kpID, Title: title, Status: string(mastery.RollupSkills(statuses)), Skills: skills}, nil
}
