package taskgen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"gorm.io/gorm"
	"sort"
	"strings"
	"time"
)

func (s *Service) ImportLegacy(ctx context.Context, id int64, key string) (Task, error) {
	if strings.TrimSpace(key) == "" || len(key) > 200 {
		return Task{}, fault(400, "missing_key", "缺少请求标识")
	}
	dedup := "import:" + hash([]any{id, key})
	var existing Task
	if e := s.DB.Where("review_key = ?", dedup).Take(&existing).Error; e == nil {
		return s.Get(ctx, existing.ID)
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return Task{}, e
	}
	var old Task
	if e := s.DB.WithContext(ctx).First(&old, id).Error; e != nil {
		return Task{}, e
	}
	if old.SourceMode != "legacy_pool" {
		return Task{}, fault(422, "not_legacy", "只支持导入旧题包")
	}
	if s.Materials == nil {
		return Task{}, fault(503, "materials_unavailable", "素材服务未配置")
	}
	var rows []struct {
		ID, KpID                          int64
		Code, Stem, Options, Answer, Text string
	}
	e := s.DB.WithContext(ctx).Raw(`SELECT q.id,q.kp_id,q.code,q.stem,q.options,q.answer,kp.title AS text FROM question_task_items i JOIN questions q ON q.id=i.question_id JOIN knowledge_points kp ON kp.id=q.kp_id WHERE i.task_id=? ORDER BY i.seq`, id).Scan(&rows).Error
	if e != nil {
		return Task{}, e
	}
	if len(rows) != old.TargetCount || len(rows) == 0 || len(rows) > 20 {
		return Task{}, fault(422, "legacy_incomplete", "旧题包缺失题目或题量超出范围")
	}
	materials, e := s.Materials.List(ctx, "")
	if e != nil {
		return Task{}, e
	}
	byText := map[string]generation.Material{}
	byID := map[int64]generation.Material{}
	for _, m := range materials {
		byText[m.Text] = m
		byID[m.KpID] = m
	}
	needed := map[int64]generation.Material{}
	type oldOption struct {
		KpID  int64  `json:"kpId"`
		Label string `json:"label"`
	}
	parsed := make([][]oldOption, len(rows))
	answers := make([]int, len(rows))
	for i, row := range rows {
		if row.Code != "glyph_sense" && row.Code != "sense_char" {
			return Task{}, fault(422, "legacy_type", "旧题包存在暂不支持的题型")
		}
		if e = json.Unmarshal([]byte(row.Options), &parsed[i]); e != nil || len(parsed[i]) != 4 {
			return Task{}, fault(422, "legacy_options", fmt.Sprintf("第%d题选项缺失，无法可靠导入", i+1))
		}
		var answer struct {
			Index *int `json:"index"`
		}
		if json.Unmarshal([]byte(row.Answer), &answer) != nil || answer.Index == nil || *answer.Index < 0 || *answer.Index > 3 {
			return Task{}, fault(422, "legacy_answer", "旧题答案不完整")
		}
		answers[i] = *answer.Index
		target, ok := byID[row.KpID]
		if !ok {
			return Task{}, fault(422, "legacy_material", "目标字素材不存在")
		}
		needed[target.KpID] = target
		for j, opt := range parsed[i] {
			m, ok := byID[opt.KpID]
			if opt.KpID == 0 {
				m, ok = byText[opt.Label]
			}
			if !ok || opt.Label != "" && m.Text != opt.Label {
				return Task{}, fault(422, "legacy_material", fmt.Sprintf("第%d题选项『%s』素材无法匹配", i+1, opt.Label))
			}
			parsed[i][j].KpID = m.KpID
			needed[m.KpID] = m
		}
	}
	toFreeze := []generation.Material{}
	for _, m := range needed {
		toFreeze = append(toFreeze, m)
	}
	sort.Slice(toFreeze, func(i, j int) bool { return toFreeze[i].KpID < toFreeze[j].KpID })
	frozen, e := s.Materials.Freeze(ctx, toFreeze)
	if e != nil {
		return Task{}, e
	}
	byID = map[int64]generation.Material{}
	for _, m := range frozen {
		byID[m.KpID] = m
	}
	qs := []generation.Snapshot{}
	counts := map[string]int{}
	for i, row := range rows {
		target := byID[row.KpID]
		q := generation.Snapshot{SchemaVersion: 1, SubjectCode: "literacy", KpID: row.KpID, TargetText: row.Text, QuestionType: row.Code, SkillCode: row.Code, TemplateVersion: "literacy-choice-v1", Prompt: row.Stem, SourceQuestionID: row.ID, Explanation: "按导入时素材保存；无法恢复原始历史媒体"}
		if row.Code == "glyph_sense" {
			q.Stem = generation.Stem{Text: row.Text, Image: target.Glyph}
		} else {
			q.Stem = generation.Stem{Image: target.Sense, Audio: target.Speech}
		}
		for j, opt := range parsed[i] {
			m := byID[opt.KpID]
			o := generation.Option{ID: fmt.Sprintf("kp:%d", m.KpID), KpID: m.KpID, Text: m.Text, Audio: m.Speech}
			if row.Code == "glyph_sense" {
				o.Image = m.Sense
			} else {
				o.Image = m.Glyph
			}
			q.Options = append(q.Options, o)
			q.MaterialRevisionIDs = append(q.MaterialRevisionIDs, m.RevisionID)
			if j == answers[i] {
				q.AnswerOptionID = o.ID
			}
		}
		sort.Strings(q.MaterialRevisionIDs)
		if e = generation.ValidateSnapshot(q); e != nil {
			return Task{}, fault(422, "legacy_invalid", fmt.Sprintf("第%d题：%s", i+1, e))
		}
		qs = append(qs, q)
		counts[row.Code]++
	}
	spec := generation.Spec{SubjectCode: "literacy", Kind: "practice", Scope: generation.Scope{ModuleCodes: []string{old.ModuleCode}}, TargetCount: len(qs), TypeCounts: counts, DistractorScope: "subject"}
	var result Task
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		local := New(tx, s.Materials)
		title := []rune("导入 · " + old.Title)
		if len(title) > 80 {
			title = title[:80]
		}
		var e error
		result, e = local.create(ctx, string(title), spec, nil, &id, &dedup)
		if e != nil {
			return e
		}
		return local.commit(ctx, result, Run{Seed: time.Now().UnixNano()}, qs, nil)
	})
	if e != nil {
		return Task{}, e
	}
	return s.Get(ctx, result.ID)
}
