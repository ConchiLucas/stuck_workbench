package taskgen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"gorm.io/gorm"
	"math/rand"
	"sort"
	"time"
)

type ReviewInput struct {
	ChildID      int64  `json:"childId"`
	SourcePlanID int64  `json:"sourcePlanId"`
	TargetCount  int    `json:"targetCount"`
	Mode         string `json:"mode"`
}
type ReviewPlan struct {
	PlanID             int64     `json:"planId"`
	CompletedAt        time.Time `json:"completedAt"`
	EligibleWrongCount int       `json:"eligibleWrongCount"`
}
type reviewReceipt struct {
	ID                int64
	QuestionVersionID int64
	KpID              int64
	QuestionType      string
	SelectedOptionID  string
	CreatedAt         time.Time
	SnapshotJSON      string
}
type reviewGroup struct {
	Receipts []reviewReceipt
	Snapshot generation.Snapshot
	SourceID int64
}
type ReviewPreview struct {
	Available  int                   `json:"available"`
	Requested  int                   `json:"requested"`
	Message    string                `json:"message"`
	Questions  []generation.Snapshot `json:"-"`
	SourceIDs  map[int]int64         `json:"-"`
	ReceiptIDs []int64               `json:"-"`
}

func (s *Service) ReviewPlans(ctx context.Context, task, child int64) ([]ReviewPlan, error) {
	if child < 1 {
		return nil, fault(400, "child_required", "孩子编号无效")
	}
	if !s.DB.Migrator().HasTable("question_attempt_receipts") {
		return nil, fault(503, "learning_not_ready", "作答回执尚未接入")
	}
	out := []ReviewPlan{}
	e := s.DB.WithContext(ctx).Raw(`SELECT p.id AS plan_id,p.completed_at,(SELECT COUNT(*) FROM question_attempt_receipts a WHERE a.plan_id=p.id AND a.child_id=p.child_id AND `+s.ReviewPredicate("a")+`) AS eligible_wrong_count FROM study_plans p WHERE p.child_id=? AND p.source_question_task_id=? AND p.status='done' ORDER BY p.completed_at DESC`, child, task).Scan(&out).Error
	return out, e
}
func (s *Service) reviewGroups(ctx context.Context, task int64, in ReviewInput) ([]reviewGroup, error) {
	if in.ChildID < 1 || in.SourcePlanID < 1 {
		return nil, fault(400, "invalid_source", "请选择来源练习")
	}
	if !s.DB.Migrator().HasTable("question_attempt_receipts") {
		return nil, fault(503, "learning_not_ready", "作答回执尚未接入")
	}
	var n int64
	e := s.DB.WithContext(ctx).Table("study_plans").Where("id = ? AND child_id = ? AND source_question_task_id = ? AND status = ?", in.SourcePlanID, in.ChildID, task, "done").Count(&n).Error
	if e != nil {
		return nil, e
	}
	if n != 1 {
		return nil, fault(404, "source_not_found", "找不到属于这个孩子的已完成来源练习")
	}
	var receipts []reviewReceipt
	e = s.DB.WithContext(ctx).Raw(`SELECT a.id,a.question_version_id,a.kp_id,a.question_type,a.selected_option_id,a.created_at,v.snapshot_json FROM question_attempt_receipts a JOIN question_versions v ON v.id=a.question_version_id WHERE a.plan_id=? AND a.child_id=? AND `+s.ReviewPredicate("a")+` ORDER BY a.created_at DESC,a.id DESC`, in.SourcePlanID, in.ChildID).Scan(&receipts).Error
	if e != nil {
		return nil, e
	}
	groups := []reviewGroup{}
	indexes := map[string]int{}
	for _, r := range receipts {
		key := fmt.Sprintf("%d:%s", r.KpID, r.QuestionType)
		if i, ok := indexes[key]; ok {
			groups[i].Receipts = append(groups[i].Receipts, r)
			continue
		}
		var q generation.Snapshot
		if e = json.Unmarshal([]byte(r.SnapshotJSON), &q); e != nil {
			return nil, e
		}
		if e = generation.ValidateSnapshot(q); e != nil {
			return nil, e
		}
		indexes[key] = len(groups)
		groups = append(groups, reviewGroup{Receipts: []reviewReceipt{r}, Snapshot: q, SourceID: r.QuestionVersionID})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		if len(a.Receipts) != len(b.Receipts) {
			return len(a.Receipts) > len(b.Receipts)
		}
		if !a.Receipts[0].CreatedAt.Equal(b.Receipts[0].CreatedAt) {
			return a.Receipts[0].CreatedAt.After(b.Receipts[0].CreatedAt)
		}
		if a.Snapshot.KpID != b.Snapshot.KpID {
			return a.Snapshot.KpID < b.Snapshot.KpID
		}
		return a.Snapshot.QuestionType < b.Snapshot.QuestionType
	})
	return groups, nil
}
func (s *Service) PreviewReview(ctx context.Context, task int64, in ReviewInput) (ReviewPreview, error) {
	out := ReviewPreview{SourceIDs: map[int]int64{}}
	if in.Mode == "" {
		in.Mode = "mixed"
	}
	if in.Mode != "mixed" && in.Mode != "original_only" {
		return out, fault(422, "invalid_mode", "无效复习方式")
	}
	groups, e := s.reviewGroups(ctx, task, in)
	if e != nil {
		return out, e
	}
	if len(groups) == 0 {
		return out, fault(422, "no_wrong_answers", "这次练习没有可用于复习的错题")
	}
	wanted := in.TargetCount
	if wanted == 0 {
		wanted = len(groups)
		if in.Mode == "mixed" {
			wanted *= 2
		}
		if wanted > 10 {
			wanted = 10
		}
	}
	if wanted < 1 || wanted > 20 {
		return out, fault(422, "invalid_count", "复习题量必须为1–20")
	}
	out.Requested = wanted
	for _, g := range groups {
		for _, r := range g.Receipts {
			out.ReceiptIDs = append(out.ReceiptIDs, r.ID)
		}
	}
	sort.Slice(out.ReceiptIDs, func(i, j int) bool { return out.ReceiptIDs[i] < out.ReceiptIDs[j] })
	seen := map[string]bool{}
	for _, g := range groups {
		if len(out.Questions) >= wanted {
			break
		}
		fp := generation.Fingerprint(g.Snapshot)
		if seen[fp] {
			continue
		}
		seen[fp] = true
		out.SourceIDs[len(out.Questions)] = g.SourceID
		out.Questions = append(out.Questions, g.Snapshot)
	}
	if in.Mode == "mixed" && len(out.Questions) < wanted {
		parent, e := s.Get(ctx, task)
		if e != nil {
			return out, e
		}
		if s.Materials == nil {
			return out, fault(503, "materials_unavailable", "素材服务未配置")
		}
		module := parent.ModuleCode
		if parent.Spec.DistractorScope == "subject" {
			module = ""
		}
		rows, e := s.Materials.List(ctx, module)
		if e != nil {
			return out, e
		}
		ready := usableMaterials(rows, parent.Spec)
		if len(ready) > 80 {
			priority := map[int64]bool{}
			wrong := map[string]bool{}
			for _, g := range groups {
				priority[g.Snapshot.KpID] = true
				for _, receipt := range g.Receipts {
					wrong[receipt.SelectedOptionID] = true
				}
			}
			limited := []generation.Material{}
			for _, m := range ready {
				if priority[m.KpID] || wrong[fmt.Sprintf("kp:%d", m.KpID)] {
					limited = append(limited, m)
					priority[m.KpID] = true
				}
			}
			if len(limited) > 80 {
				return out, fault(422, "material_limit", "复习来源素材超过80个，请缩小来源范围")
			}
			for _, m := range ready {
				if len(limited) >= 80 {
					break
				}
				if !priority[m.KpID] {
					limited = append(limited, m)
				}
			}
			ready = limited
		}
		frozen, e := s.Materials.Freeze(ctx, ready)
		if e != nil {
			return out, e
		}
		byID := map[int64]generation.Material{}
		for _, m := range frozen {
			byID[m.KpID] = m
		}
		r := rand.New(rand.NewSource(in.SourcePlanID))
		for _, g := range groups {
			if len(out.Questions) >= wanted {
				break
			}
			target, ok := byID[g.Snapshot.KpID]
			if !ok {
				continue
			}
			wrong := map[string]bool{}
			for _, receipt := range g.Receipts {
				wrong[receipt.SelectedOptionID] = true
			}
			var chosen *generation.Snapshot
			for attempt := 0; attempt < 128; attempt++ {
				q, e := generation.Build(target, g.Snapshot.QuestionType, frozen, r)
				if e != nil {
					break
				}
				if seen[generation.Fingerprint(q)] {
					continue
				}
				if chosen == nil {
					copy := q
					chosen = &copy
				}
				retained := false
				for _, o := range q.Options {
					if wrong[o.ID] {
						retained = true
					}
				}
				if retained {
					copy := q
					chosen = &copy
					break
				}
			}
			if chosen != nil {
				seen[generation.Fingerprint(*chosen)] = true
				out.SourceIDs[len(out.Questions)] = g.SourceID
				out.Questions = append(out.Questions, *chosen)
			}
		}
	}
	out.Available = len(out.Questions)
	if out.Available < wanted {
		out.Message = fmt.Sprintf("只能生成 %d 题，请减少题量或选择只重练原题", out.Available)
	} else {
		out.Message = "题目来源和素材已检查，可生成复习草稿"
	}
	return out, nil
}
func (s *Service) CreateReview(ctx context.Context, parentID int64, in ReviewInput) (Task, error) {
	if in.Mode == "" {
		in.Mode = "mixed"
	}
	groups, e := s.reviewGroups(ctx, parentID, in)
	if e != nil {
		return Task{}, e
	}
	ids := []int64{}
	for _, g := range groups {
		for _, r := range g.Receipts {
			ids = append(ids, r.ID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	key := hash([]any{in.ChildID, in.SourcePlanID, in.TargetCount, in.Mode, ids, "wrong-answer-v1"})
	var existing Task
	e = s.DB.Where("review_key = ?", key).Take(&existing).Error
	if e == nil {
		return s.Get(ctx, existing.ID)
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return Task{}, e
	}
	preview, e := s.PreviewReview(ctx, parentID, in)
	if e != nil {
		return Task{}, e
	}
	if preview.Available != preview.Requested {
		return Task{}, fault(422, "review_capacity", preview.Message)
	}
	parent, e := s.Get(ctx, parentID)
	if e != nil {
		return Task{}, e
	}
	spec := parent.Spec
	spec.Kind = "review"
	spec.SubjectCode = "literacy"
	spec.TargetCount = preview.Requested
	spec.TypeCounts = map[string]int{}
	spec.Scope = generation.Scope{ModuleCodes: []string{parent.ModuleCode}}
	for _, q := range preview.Questions {
		spec.TypeCounts[q.QuestionType]++
	}
	var t Task
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		service := New(tx, s.Materials)
		var e error
		t, e = service.create(ctx, "复习 · "+parent.Title, spec, &in.ChildID, &parentID, &key)
		if e != nil {
			return e
		}
		if e = service.commit(ctx, t, Run{Seed: in.SourcePlanID}, preview.Questions, preview.SourceIDs); e != nil {
			return e
		}
		for _, g := range groups {
			for _, r := range g.Receipts {
				if e = tx.Create(&ReviewSource{TaskID: t.ID, SourceReceiptID: r.ID, SourceQuestionVersionID: r.QuestionVersionID, Reason: "wrong-answer-v1"}).Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
	if e != nil {
		if s.DB.Where("review_key = ?", key).Take(&existing).Error == nil {
			return s.Get(ctx, existing.ID)
		}
		return Task{}, e
	}
	return s.Get(ctx, t.ID)
}

// ReviewPredicate is used only with internal SQL aliases.
func (s *Service) ReviewPredicate(alias string) string {
	base := alias + ".is_correct=false"
	if s.DB.Migrator().HasColumn("question_attempt_receipts", "evaluation_json") {
		return "(" + base + " OR (" + alias + ".response_kind='handwriting' AND " + alias + ".evaluation_json LIKE '%\"assistance\":\"hinted\"%'))"
	}
	return "(" + base + ")"
}
