package taskgen

import (
	"context"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"gorm.io/gorm"
	"math/rand"
	"sort"
	"time"
)

type EvidenceOriginal struct {
	VersionID int64
	Snapshot  generation.Snapshot
}
type EvidenceReviewTarget struct {
	Key                      string
	KpID                     int64
	QuestionType             string
	Mode                     string
	Count                    int
	PreferredDistractorKpIDs []int64
	Originals                []EvidenceOriginal
}
type EvidenceTargetMap struct {
	QuestionVersionID       int64  `json:"questionVersionId"`
	Seq                     int    `json:"seq"`
	TargetKey               string `json:"targetKey"`
	Kind                    string `json:"kind"`
	SourceQuestionVersionID int64  `json:"sourceQuestionVersionId"`
}
type GenerationWarning struct {
	TargetKey string `json:"targetKey"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}
type EvidenceReviewPrepared struct {
	Warnings  []GenerationWarning
	Questions []generation.Snapshot
	SourceIDs map[int]int64
	TargetMap []EvidenceTargetMap
}

// PrepareEvidenceReview is used only after an explicit suggestion run is claimed.
// It does not save tasks; callers must atomically commit the result with their ownership guard.
func (s *Service) PrepareEvidenceReview(ctx context.Context, module string, seed int64, targets []EvidenceReviewTarget) (EvidenceReviewPrepared, error) {
	out := EvidenceReviewPrepared{SourceIDs: map[int]int64{}, Questions: []generation.Snapshot{}, TargetMap: []EvidenceTargetMap{}}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Key < targets[j].Key })
	needsMaterials := false
	for _, t := range targets {
		if t.Count < 1 || t.Count > 10 || (t.Mode != "mixed" && t.Mode != "original_only") {
			return out, fault(422, "invalid_target", "invalid evidence review target")
		}
		if t.Mode == "mixed" && t.Count > 1 {
			needsMaterials = true
		}
	}
	var frozen []generation.Material
	if needsMaterials {
		if s.Materials == nil {
			return out, fault(503, "materials_unavailable", "素材服务未配置")
		}
		listCtx, listCancel := context.WithTimeout(ctx, 20*time.Second)
		rows, e := s.Materials.List(listCtx, module)
		listCancel()
		if e != nil {
			return out, e
		}
		spec := generation.Spec{SubjectCode: "literacy", Kind: "review", Scope: generation.Scope{ModuleCodes: []string{module}}, DistractorScope: "module"}
		spec.TypeCounts = map[string]int{}
		for _, t := range targets {
			spec.TypeCounts[t.QuestionType] += t.Count
		}
		ready := usableMaterials(rows, spec)
		if len(ready) > 80 {
			return out, fault(422, "material_limit", "复习素材超过80个，请缩小范围")
		}
		freezeCtx, freezeCancel := context.WithTimeout(ctx, 20*time.Second)
		frozen, e = s.Materials.Freeze(freezeCtx, ready)
		freezeCancel()
		if e != nil {
			return out, e
		}
	}
	byID := map[int64]generation.Material{}
	for _, m := range frozen {
		byID[m.KpID] = m
	}
	seen := map[string]bool{}
	rng := rand.New(rand.NewSource(seed))
	appendQuestion := func(t EvidenceReviewTarget, q generation.Snapshot, source int64, kind string) {
		idx := len(out.Questions)
		out.SourceIDs[idx] = source
		out.Questions = append(out.Questions, q)
		out.TargetMap = append(out.TargetMap, EvidenceTargetMap{Seq: idx + 1, TargetKey: t.Key, Kind: kind, SourceQuestionVersionID: source})
		seen[generation.Fingerprint(q)] = true
	}
	for _, t := range targets {
		originals := []EvidenceOriginal{}
		originalSeen := map[string]bool{}
		sort.Slice(t.Originals, func(i, j int) bool { return t.Originals[i].VersionID < t.Originals[j].VersionID })
		for _, o := range t.Originals {
			if o.VersionID < 1 || o.Snapshot.KpID != t.KpID || o.Snapshot.QuestionType != t.QuestionType {
				return out, fault(422, "source_mismatch", "original does not match target")
			}
			if e := generation.ValidateSnapshot(o.Snapshot); e != nil {
				return out, fault(422, "invalid_snapshot", e.Error())
			}
			fp := generation.Fingerprint(o.Snapshot)
			if !originalSeen[fp] {
				originalSeen[fp] = true
				originals = append(originals, o)
			}
		}
		if len(originals) == 0 {
			return out, fault(422, "missing_original", "没有可用于该题型的来源错题")
		}
		n := 1
		if t.Mode == "original_only" {
			n = t.Count
		}
		if len(originals) < n {
			return out, fault(422, "review_capacity", fmt.Sprintf("原题只有%d种独立内容，不能重复填充%d题", len(originals), n))
		}
		for i := 0; i < n; i++ {
			if seen[generation.Fingerprint(originals[i].Snapshot)] {
				return out, fault(422, "duplicate_original", "原题与同分片题目重复")
			}
			appendQuestion(t, originals[i].Snapshot, originals[i].VersionID, "original")
		}
		if t.Mode == "original_only" {
			continue
		}
		material, ok := byID[t.KpID]
		if t.Count > 1 && !ok {
			return out, fault(422, "missing_material", "缺少目标知识点可用素材")
		}
		for count := 1; count < t.Count; count++ {
			var candidate *generation.Snapshot
			for attempt := 0; attempt < 128; attempt++ {
				q, e := generation.Build(material, t.QuestionType, frozen, rng)
				if e != nil {
					return out, fault(422, "unsupported_question_type", e.Error())
				}
				if seen[generation.Fingerprint(q)] || originalSeen[generation.Fingerprint(q)] {
					continue
				}
				copy := q
				if candidate == nil {
					candidate = &copy
				}
				preferred := len(t.PreferredDistractorKpIDs) == 0
				for _, o := range q.Options {
					for _, id := range t.PreferredDistractorKpIDs {
						if o.KpID == id {
							preferred = true
						}
					}
				}
				if preferred {
					candidate = &copy
					break
				}
			}
			if candidate == nil {
				return out, fault(422, "review_capacity", "没有足够的不同变式，请减少题量")
			}
			if len(t.PreferredDistractorKpIDs) > 0 {
				retained := false
				for _, o := range candidate.Options {
					for _, id := range t.PreferredDistractorKpIDs {
						if o.KpID == id {
							retained = true
						}
					}
				}
				if !retained {
					out.Warnings = append(out.Warnings, GenerationWarning{TargetKey: t.Key, Code: "preferred_distractor_unavailable", Message: "可用变式未能保留优先错选内容，已按软偏好生成。"})
				}
			}
			appendQuestion(t, *candidate, originals[0].VersionID, "variant")
		}
	}
	if len(out.Questions) > 20 {
		return out, fault(422, "review_capacity", "分片题量超过20")
	}
	if s.Support != nil {
		if e := s.Support.Verify(ctx, out.Questions); e != nil {
			return out, e
		}
	}
	return out, nil
}

// CommitEvidenceReview retains the existing revision/source implementation, with no arbitrary parent task.
// Calling with a transaction lets the suggestion partition and mapping commit atomically with the task.
func (s *Service) CommitEvidenceReview(ctx context.Context, child int64, title, module, key string, seed int64, p EvidenceReviewPrepared, sources []ReviewSource) (Task, error) {
	spec := generation.Spec{SubjectCode: "literacy", Kind: "review", Scope: generation.Scope{ModuleCodes: []string{module}}, TargetCount: len(p.Questions), TypeCounts: map[string]int{}, DistractorScope: "module"}
	for _, q := range p.Questions {
		spec.TypeCounts[q.QuestionType]++
	}
	var task Task
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		svc := New(tx, s.Materials)
		var e error
		task, e = svc.create(ctx, title, spec, &child, nil, &key)
		if e != nil {
			return e
		}
		if e = svc.commit(ctx, task, Run{Seed: seed}, p.Questions, p.SourceIDs); e != nil {
			return e
		}
		seen := map[int64]bool{}
		for _, src := range sources {
			if seen[src.SourceReceiptID] {
				continue
			}
			seen[src.SourceReceiptID] = true
			src.TaskID = task.ID
			if e = tx.Create(&src).Error; e != nil {
				return e
			}
		}
		task, e = svc.Get(ctx, task.ID)
		return e
	})
	return task, e
}
