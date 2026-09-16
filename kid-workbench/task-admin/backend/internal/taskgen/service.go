package taskgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/conchi/study-task-admin/internal/generation"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math/rand"
	"strings"
	"time"
)

type Materials interface {
	List(context.Context, string) ([]generation.Material, error)
	Freeze(context.Context, []generation.Material) ([]generation.Material, error)
}
type Service struct {
	Support   *RuntimeSupport
	DB        *gorm.DB
	Materials Materials
}

func New(db *gorm.DB, m Materials) *Service { return &Service{DB: db, Materials: m} }

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string                 { return e.Message }
func fault(status int, code, msg string) error { return &Error{status, code, msg} }
func (s *Service) Create(ctx context.Context, title string, in generation.Spec) (Task, error) {
	spec, err := generation.Normalize(in)
	if err != nil {
		return Task{}, fault(422, "invalid_spec", err.Error())
	}
	if spec.Kind == "review" {
		return Task{}, fault(422, "review_source_required", "请从已完成练习生成复习任务")
	}
	return s.create(ctx, title, spec, nil, nil, nil)
}
func (s *Service) create(ctx context.Context, title string, spec generation.Spec, child, parent *int64, key *string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "识字 · " + spec.Scope.ModuleCodes[0]
	}
	if len([]rune(title)) > 80 {
		return Task{}, fault(422, "invalid_title", "标题不能超过80字")
	}
	b, _ := json.Marshal(spec)
	now := time.Now().UTC()
	t := Task{SubjectCode: spec.SubjectCode, Title: title, ModuleCode: spec.Scope.ModuleCodes[0], ModuleName: spec.Scope.ModuleCodes[0], TargetCount: spec.TargetCount, Status: "draft", Kind: spec.Kind, SourceMode: "material_template", SpecJSON: string(b), RowVersion: 1, CreatedAt: now, UpdatedAt: now, TargetChildID: child, ParentTaskID: parent, ReviewKey: key}
	err := s.DB.WithContext(ctx).Create(&t).Error
	decodeTask(&t)
	return t, err
}
func (s *Service) Get(ctx context.Context, id int64) (Task, error) {
	var t Task
	if err := s.DB.WithContext(ctx).First(&t, id).Error; err != nil {
		return t, err
	}
	decodeTask(&t)
	if s.DB.Migrator().HasTable("review_suggestion_tasks") {
		var link struct{ SuggestionID int64 }
		if e := s.DB.WithContext(ctx).Table("review_suggestion_tasks").Where("task_id=?", id).Take(&link).Error; e == nil {
			t.SourceReviewSuggestionID = &link.SuggestionID
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return t, e
		}
	}
	if t.ActiveRevisionID != nil {
		r, e := s.Revision(ctx, id, *t.ActiveRevisionID)
		if e != nil {
			return t, e
		}
		t.Items = r.Items
	}
	if e := s.DB.WithContext(ctx).Where("task_id = ?", id).Order("revision_no DESC").Find(&t.Revisions).Error; e != nil {
		return t, e
	}
	var run Run
	if s.DB.Where("task_id = ?", id).Order("id DESC").Take(&run).Error == nil && run.State == "failed" {
		t.LastError = run.Error
	}
	return t, nil
}
func (s *Service) List(ctx context.Context, subject, status, kind string) ([]Task, error) {
	q := s.DB.WithContext(ctx)
	if subject != "" {
		q = q.Where("subject_code = ?", subject)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	ts := []Task{}
	err := q.Order("updated_at DESC,id DESC").Find(&ts).Error
	for i := range ts {
		decodeTask(&ts[i])
	}
	return ts, err
}
func (s *Service) Revision(ctx context.Context, task, id int64) (Revision, error) {
	var r Revision
	if e := s.DB.WithContext(ctx).Where("id = ? AND task_id = ?", id, task).First(&r).Error; e != nil {
		return r, e
	}
	r.Items = []QuestionVersion{}
	e := s.DB.Where("revision_id = ?", r.ID).Order("seq").Find(&r.Items).Error
	if e == nil {
		e = decodeItems(r.Items)
	}
	return r, e
}
func lockTask(tx *gorm.DB, id, expected int64) (Task, error) {
	var t Task
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&t, id).Error
	if e != nil {
		return t, e
	}
	if expected < 1 || t.RowVersion != expected {
		return t, fault(409, "version_conflict", "任务已更新，请刷新后重试")
	}
	if t.SourceMode != "material_template" {
		return t, fault(422, "legacy_task", "请先将旧题包导入新任务")
	}
	decodeTask(&t)
	return t, nil
}
func (s *Service) Update(ctx context.Context, id, expected int64, title string, in generation.Spec) (Task, error) {
	spec, e := generation.Normalize(in)
	if e != nil {
		return Task{}, e
	}
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 80 {
		return Task{}, fault(422, "invalid_title", "请填写80字以内标题")
	}
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		t, e := lockTask(tx, id, expected)
		if e != nil {
			return e
		}
		if t.Status != "draft" {
			return fault(409, "not_draft", "撤回后才可编辑")
		}
		if t.Kind != spec.Kind {
			return fault(422, "invalid_kind", "不可改变任务类型")
		}
		if t.Kind == "review" {
			if hash(spec) != hash(t.Spec) {
				return fault(422, "review_spec_immutable", "复习出题要求不可修改，请从来源练习重新生成")
			}
			return tx.Model(&Task{}).Where("id = ? AND row_version = ?", id, expected).Updates(map[string]any{"title": title, "row_version": expected + 1, "updated_at": time.Now().UTC()}).Error
		}
		b, _ := json.Marshal(spec)
		return tx.Model(&Task{}).Where("id = ? AND row_version = ?", id, expected).Updates(map[string]any{"title": title, "spec_json": string(b), "module_code": spec.Scope.ModuleCodes[0], "target_count": spec.TargetCount, "active_revision_id": nil, "row_version": expected + 1, "updated_at": time.Now().UTC()}).Error
	})
	if e != nil {
		return Task{}, e
	}
	return s.Get(ctx, id)
}
func hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s *Service) Generate(ctx context.Context, id, expected int64, key string, replaceSeq int) (Task, error) {
	if strings.TrimSpace(key) == "" || len(key) > 200 {
		return Task{}, fault(400, "missing_key", "缺少有效请求标识")
	}
	requestHash := hash([]any{expected, replaceSeq})
	var existing Run
	e := s.DB.WithContext(ctx).Where("task_id = ? AND key = ?", id, key).First(&existing).Error
	if e == nil {
		if existing.RequestHash != requestHash {
			return Task{}, fault(409, "key_conflict", "请求标识已用于其他操作")
		}
		if existing.State == "done" {
			return s.Get(ctx, id)
		}
		return Task{}, fault(409, "generation_"+existing.State, existing.Error+"；请刷新或使用新的请求重试")
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return Task{}, e
	}
	var t Task
	run := Run{TaskID: id, Key: key, RequestHash: requestHash, State: "running", Seed: time.Now().UnixNano()}
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var e error
		t, e = lockTask(tx, id, expected)
		if e != nil {
			return e
		}
		if t.Status != "draft" {
			return fault(409, "not_draft", "撤回后才可重新生成")
		}
		if t.Kind == "review" {
			return fault(422, "review_source_required", "请从来源练习重新生成复习，避免丢失出题依据")
		}
		return tx.Create(&run).Error
	})
	if e != nil {
		return Task{}, e
	}
	fail := func(err error) (Task, error) {
		s.DB.Model(&Run{}).Where("id = ?", run.ID).Updates(map[string]any{"state": "failed", "error": err.Error(), "updated_at": time.Now().UTC()})
		return Task{}, err
	}
	if s.Materials == nil {
		return fail(fault(503, "materials_unavailable", "素材服务未配置"))
	}
	module := t.ModuleCode
	if t.Spec.DistractorScope == "subject" {
		module = ""
	}
	candidates, e := s.Materials.List(ctx, module)
	if e != nil {
		return fail(e)
	}
	usable := usableMaterials(candidates, t.Spec)
	if len(usable) > 80 { // Preserve selected targets; cap only the broad distractor pool.
		filtered := []generation.Material{}
		for _, m := range usable {
			if m.ModuleCode == t.ModuleCode {
				filtered = append(filtered, m)
			}
		}
		for _, m := range usable {
			if len(filtered) >= 80 {
				break
			}
			if m.ModuleCode != t.ModuleCode {
				filtered = append(filtered, m)
			}
		}
		usable = filtered
	}
	if len(usable) > 80 {
		return fail(fault(422, "material_limit", "本组超过80个素材，请缩小出题范围"))
	}
	frozen, e := s.Materials.Freeze(ctx, usable)
	if e != nil {
		return fail(e)
	}
	qs, e := generation.Generate(t.Spec, frozen, run.Seed)
	if e != nil {
		return fail(fault(422, "generation_failed", e.Error()))
	}
	if t.ActiveRevisionID != nil {
		old, e := s.Revision(ctx, t.ID, *t.ActiveRevisionID)
		if e != nil {
			return fail(e)
		}
		if replaceSeq > 0 {
			if replaceSeq > len(old.Items) {
				return fail(fault(422, "invalid_seq", "题目序号无效"))
			}
			target := old.Items[replaceSeq-1].Snapshot
			var material generation.Material
			for _, m := range frozen {
				if m.KpID == target.KpID {
					material = m
				}
			}
			changed := false
			r := rand.New(rand.NewSource(run.Seed))
			occupied := map[string]bool{}
			for _, q := range old.Items {
				occupied[q.Fingerprint] = true
			}
			var fresh generation.Snapshot
			for i := 0; i < 128; i++ {
				fresh, e = generation.Build(material, target.QuestionType, frozen, r)
				if e != nil {
					break
				}
				if !occupied[generation.Fingerprint(fresh)] {
					changed = true
					break
				}
			}
			if !changed {
				return fail(fault(422, "no_variant", "没有新的可用题目组合"))
			}
			qs = nil
			for i, q := range old.Items {
				if i == replaceSeq-1 {
					qs = append(qs, fresh)
				} else {
					qs = append(qs, q.Snapshot)
				}
			}
		} else {
			oldFP := map[string]bool{}
			for _, q := range old.Items {
				oldFP[q.Fingerprint] = true
			}
			changed := false
			for attempt := 0; attempt < 128; attempt++ {
				for _, q := range qs {
					if !oldFP[generation.Fingerprint(q)] {
						changed = true
						break
					}
				}
				if changed {
					break
				}
				if attempt < 127 {
					run.Seed++
					qs, e = generation.Generate(t.Spec, frozen, run.Seed)
					if e != nil {
						return fail(fault(422, "generation_failed", e.Error()))
					}
				}
			}
			if !changed {
				return fail(fault(422, "no_variant", "没有新的可用题目组合"))
			}
		}
	} else if replaceSeq > 0 {
		return fail(fault(422, "no_questions", "请先生成题目"))
	}
	e = s.commit(ctx, t, run, qs, nil)
	if e != nil {
		return fail(e)
	}
	return s.Get(ctx, id)
}
func (s *Service) commit(ctx context.Context, t Task, run Run, qs []generation.Snapshot, sources map[int]int64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, e := lockTask(tx, t.ID, t.RowVersion)
		if e != nil {
			return e
		}
		if current.Status != "draft" {
			return fault(409, "not_draft", "任务状态已改变")
		}
		var n int64
		tx.Model(&Revision{}).Where("task_id = ?", t.ID).Count(&n)
		rev := Revision{TaskID: t.ID, RevisionNo: int(n) + 1, SpecJSON: t.SpecJSON, Seed: run.Seed, TemplateVersion: "literacy-choice-v1"}
		if e = tx.Create(&rev).Error; e != nil {
			return e
		}
		for i, q := range qs {
			if e = generation.ValidateSnapshot(q); e != nil {
				return e
			}
			b, _ := json.Marshal(q)
			v := QuestionVersion{RevisionID: rev.ID, Seq: i + 1, KpID: q.KpID, QuestionType: q.QuestionType, SkillCode: q.SkillCode, Fingerprint: generation.Fingerprint(q), SnapshotJSON: string(b)}
			if source, ok := sources[i]; ok {
				v.SourceQuestionVersionID = &source
			}
			if e = tx.Create(&v).Error; e != nil {
				return e
			}
		}
		res := tx.Model(&Task{}).Where("id = ? AND row_version = ?", t.ID, t.RowVersion).Updates(map[string]any{"active_revision_id": rev.ID, "row_version": t.RowVersion + 1, "updated_at": time.Now().UTC()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fault(409, "version_conflict", "任务已更新")
		}
		if run.ID > 0 {
			return tx.Model(&Run{}).Where("id = ?", run.ID).Updates(map[string]any{"state": "done", "revision_id": rev.ID, "seed": run.Seed}).Error
		}
		return nil
	})
}
func (s *Service) Publish(ctx context.Context, id, expected int64, confirmed bool) (Task, error) {
	if s.Support != nil {
		t, e := s.Get(ctx, id)
		if e != nil {
			return Task{}, e
		}
		qs := []generation.Snapshot{}
		for _, v := range t.Items {
			qs = append(qs, v.Snapshot)
		}
		if e = s.Support.Verify(ctx, qs); e != nil {
			return Task{}, e
		}
	}
	if !confirmed {
		return Task{}, fault(422, "preview_required", "请先预览并确认题目")
	}
	if verifier, ok := s.Materials.(interface {
		Verify(context.Context, []generation.Snapshot) error
	}); ok {
		task, e := s.Get(ctx, id)
		if e != nil {
			return Task{}, e
		}
		snapshots := []generation.Snapshot{}
		for _, v := range task.Items {
			snapshots = append(snapshots, v.Snapshot)
		}
		if e = verifier.Verify(ctx, snapshots); e != nil {
			return Task{}, fault(503, "media_unavailable", "冻结素材不可读取："+e.Error())
		}
	}
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		t, e := lockTask(tx, id, expected)
		if e != nil {
			return e
		}
		if t.Status != "draft" || t.ActiveRevisionID == nil {
			return fault(409, "not_ready", "需要已生成的草稿")
		}
		var qs []QuestionVersion
		if e = tx.Where("revision_id = ?", *t.ActiveRevisionID).Find(&qs).Error; e != nil {
			return e
		}
		if len(qs) != t.TargetCount {
			return fault(422, "count_mismatch", "题量与要求不一致")
		}
		counts := map[string]int{}
		for _, v := range qs {
			var q generation.Snapshot
			if e = json.Unmarshal([]byte(v.SnapshotJSON), &q); e != nil {
				return e
			}
			if e = generation.ValidateSnapshot(q); e != nil {
				return e
			}
			counts[q.QuestionType]++
		}
		for typ, n := range t.Spec.TypeCounts {
			if counts[typ] != n {
				return fault(422, "type_mismatch", "题型配比不一致")
			}
		}
		return tx.Model(&Task{}).Where("id = ?", id).Updates(map[string]any{"status": "published", "published_revision_id": t.ActiveRevisionID, "row_version": expected + 1, "updated_at": time.Now().UTC()}).Error
	})
	if e != nil {
		return Task{}, e
	}
	return s.Get(ctx, id)
}
func (s *Service) Transition(ctx context.Context, id, expected int64, status string) (Task, error) {
	if status != "draft" && status != "archived" {
		return Task{}, fault(400, "invalid_status", "无效状态")
	}
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		t, e := lockTask(tx, id, expected)
		if e != nil {
			return e
		}
		if t.Status == "archived" || status == "draft" && t.Status != "published" {
			return fault(409, "invalid_status", "当前状态不支持此操作")
		}
		return tx.Model(&Task{}).Where("id = ?", id).Updates(map[string]any{"status": status, "row_version": expected + 1, "updated_at": time.Now().UTC()}).Error
	})
	if e != nil {
		return Task{}, e
	}
	return s.Get(ctx, id)
}
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var t Task
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&t, id).Error; e != nil {
			return e
		}
		if t.Status != "draft" || t.PublishedRevisionID != nil {
			return fault(409, "archive_required", "已发布过的任务只能归档")
		}
		return tx.Delete(&t).Error
	})
}
func (s *Service) Reorder(ctx context.Context, id, expected int64, order []int) (Task, error) {
	t, e := s.Get(ctx, id)
	if e != nil {
		return t, e
	}
	if t.RowVersion != expected {
		return t, fault(409, "version_conflict", "任务已更新")
	}
	if len(order) != len(t.Items) || len(order) == 0 {
		return t, fault(422, "invalid_order", "请传完整题序")
	}
	seen := map[int]bool{}
	qs := []generation.Snapshot{}
	sources := map[int]int64{}
	for _, n := range order {
		if n < 1 || n > len(t.Items) || seen[n] {
			return t, fault(422, "invalid_order", "题序重复或无效")
		}
		seen[n] = true
		if source := t.Items[n-1].SourceQuestionVersionID; source != nil {
			sources[len(qs)] = *source
		}
		qs = append(qs, t.Items[n-1].Snapshot)
	}
	if e = s.commit(ctx, t, Run{Seed: time.Now().UnixNano()}, qs, sources); e != nil {
		return t, e
	}
	return s.Get(ctx, id)
}
func (s *Service) RecoverRuns() error {
	return s.DB.Model(&Run{}).Where("state = ?", "running").Updates(map[string]any{"state": "failed", "error": "生成被服务重启中断，请重试"}).Error
}
