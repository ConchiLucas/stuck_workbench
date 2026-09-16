package catalog

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

const mathSubject = "math"

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type itemRow struct {
	KpID        int64
	Title       string
	Payload     string
	ModuleCode  string
	ModuleName  string
	ModuleOrder int
	KpOrder     int
}

func (r *Repository) ListModules(ctx context.Context) ([]Module, error) {
	rows, err := r.rows(ctx, "")
	if err != nil {
		return nil, err
	}
	modules := make([]Module, 0, 3)
	byCode := map[string]int{}
	for _, row := range rows {
		item, err := normalize(row)
		if err != nil {
			return nil, err
		}
		index, ok := byCode[row.ModuleCode]
		if !ok {
			index = len(modules)
			byCode[row.ModuleCode] = index
			stages := make([]Stage, 0, len(stageCodes(row.ModuleCode)))
			for _, code := range stageCodes(row.ModuleCode) {
				stages = append(stages, Stage{Code: code, Name: stageName(code)})
			}
			modules = append(modules, Module{Code: row.ModuleCode, Name: row.ModuleName, OrderNo: row.ModuleOrder, Stages: stages})
		}
		modules[index].ItemCount++
		for i := range modules[index].Stages {
			if modules[index].Stages[i].Code == item.StageCode {
				modules[index].Stages[i].ItemCount++
			}
		}
	}
	return modules, nil
}

func (r *Repository) GetModule(ctx context.Context, moduleCode string) (Module, error) {
	modules, err := r.ListModules(ctx)
	if err != nil {
		return Module{}, err
	}
	for _, module := range modules {
		if module.Code == moduleCode {
			return module, nil
		}
	}
	return Module{}, gorm.ErrRecordNotFound
}

func (r *Repository) ListStageItems(ctx context.Context, moduleCode, stageCode string) ([]LearningItem, error) {
	valid := false
	for _, code := range stageCodes(moduleCode) {
		valid = valid || code == stageCode
	}
	if !valid {
		return nil, gorm.ErrRecordNotFound
	}
	rows, err := r.rows(ctx, moduleCode)
	if err != nil {
		return nil, err
	}
	items := make([]LearningItem, 0)
	for _, row := range rows {
		item, err := normalize(row)
		if err != nil {
			return nil, err
		}
		if item.StageCode == stageCode {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *Repository) rows(ctx context.Context, moduleCode string) ([]itemRow, error) {
	var rows []itemRow
	query := r.db.WithContext(ctx).Table("knowledge_points kp").
		Select(`kp.id AS kp_id, kp.title, kp.payload, kp.order_no AS kp_order,
			m.code AS module_code, m.name AS module_name, m.order_no AS module_order`).
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Where("s.code = ? AND m.code IN ?", mathSubject, []string{"add10", "sub10", "shape"})
	if moduleCode != "" {
		query = query.Where("m.code = ?", moduleCode)
	}
	err := query.Order("m.order_no, kp.order_no, kp.id").Scan(&rows).Error
	return rows, err
}

var shapeKeys = map[string]string{
	"圆形": "circle", "正方形": "square", "长方形": "rect", "三角形": "triangle",
	"椭圆形": "oval", "梯形": "trapezoid", "菱形": "rhombus", "五角星": "star",
}

func normalize(row itemRow) (LearningItem, error) {
	item := LearningItem{KpID: row.KpID, Title: row.Title, ModuleCode: row.ModuleCode}
	if row.ModuleCode == "shape" {
		shape, ok := shapeKeys[row.Title]
		if !ok {
			return LearningItem{}, fmt.Errorf("unsupported shape %q", row.Title)
		}
		item.Kind, item.Shape, item.StageCode = "shape", shape, "basic-shapes"
		return item, nil
	}
	var payload struct {
		Kind string `json:"kind"`
		A    int    `json:"a"`
		B    int    `json:"b"`
	}
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		return LearningItem{}, fmt.Errorf("invalid math payload for kp %d: %w", row.KpID, err)
	}
	if (row.ModuleCode == "add10" && payload.Kind != "add") ||
		(row.ModuleCode == "sub10" && payload.Kind != "sub") {
		return LearningItem{}, fmt.Errorf("invalid math kind %q for module %s", payload.Kind, row.ModuleCode)
	}
	item.Kind, item.A, item.B = payload.Kind, payload.A, payload.B
	item.StageCode = StageFor(row.ModuleCode, payload.A, payload.B)
	if item.StageCode == "" {
		return LearningItem{}, fmt.Errorf("out-of-range math payload for kp %d", row.KpID)
	}
	return item, nil
}
