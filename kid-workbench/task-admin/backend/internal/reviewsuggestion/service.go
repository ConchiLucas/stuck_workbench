package reviewsuggestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-task-admin/internal/taskgen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"strings"
	"time"
)

type Service struct {
	DB         *gorm.DB
	Generation *taskgen.Service
	Enabled    bool
}

func New(db *gorm.DB, gen *taskgen.Service, enabled bool) *Service { return &Service{db, gen, enabled} }

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string        { return e.Message }
func bad(code, msg string) error      { return &Error{422, code, msg} }
func conflict(code, msg string) error { return &Error{409, code, msg} }
func hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s *Service) enabled(key string) error {
	if !s.Enabled {
		return &Error{503, "review_suggestions_disabled", "review suggestions commands are disabled"}
	}
	if strings.TrimSpace(key) == "" || len(key) > 128 {
		return bad("idempotency_key_required", "Idempotency-Key is required (max 128 characters)")
	}
	return nil
}
func replayCommand(tx *gorm.DB, child int64, op, key, h string) (Suggestion, bool, error) {
	var c Command
	e := tx.Where("child_id=? AND operation=? AND idempotency_key=?", child, op, key).Take(&c).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return Suggestion{}, false, nil
	}
	if e != nil {
		return Suggestion{}, false, e
	}
	if c.RequestHash != h {
		return Suggestion{}, false, conflict("idempotency_conflict", "idempotency key has a different request")
	}
	var out Suggestion
	e = json.Unmarshal([]byte(c.ResponseJSON), &out)
	return out, true, e
}
func saveCommand(tx *gorm.DB, child int64, op, key, h string, out Suggestion, run int64) error {
	b, e := json.Marshal(out)
	if e != nil {
		return e
	}
	var runID *int64
	if run > 0 {
		runID = &run
	}
	return tx.Create(&Command{ChildID: child, Operation: op, IdempotencyKey: key, RequestHash: h, SuggestionID: out.ID, RunID: runID, ResponseJSON: string(b)}).Error
}
func (s *Service) Save(ctx context.Context, child int64, key string, in Input) (out Suggestion, replay bool, err error) {
	if err = s.enabled(key); err != nil {
		return
	}
	if child < 1 {
		return out, false, bad("invalid_child", "invalid child")
	}
	in, err = normalize(in)
	if err != nil {
		return
	}
	h := hash(in)
	content := in
	content.Title = ""
	content.SupersedesID = nil
	content.AnalysisAsOf = time.Time{}
	contentHash := hash(content)
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var e error
		if tx.Dialector.Name() == "postgres" {
			if e = tx.Exec("SELECT pg_advisory_xact_lock(19201, ?)", child).Error; e != nil {
				return e
			}
		}
		out, replay, e = replayCommand(tx, child, "save", key, h)
		if e != nil || replay {
			return e
		}
		if in.SupersedesID != nil {
			if *in.SupersedesID < 1 {
				return bad("invalid_supersedes", "invalid supersedesId")
			}
			var prior Suggestion
			if e := tx.Where("id=? AND child_id=?", *in.SupersedesID, child).Take(&prior).Error; e != nil {
				return bad("supersedes_not_found", "superseded suggestion must belong to same child")
			}
		}
		ts := []Target{}
		es := [][]Evidence{}
		for _, t := range in.Targets {
			row, ev, e := validateTarget(tx, child, in, t)
			if e != nil {
				return e
			}
			ts = append(ts, row)
			es = append(es, ev)
		}
		e = tx.Where("child_id=? AND content_hash=? AND lifecycle='open'", child, contentHash).Take(&out).Error
		if e == nil {
			replay = true
			out, e = New(tx, s.Generation, s.Enabled).Get(ctx, child, out.ID)
			if e != nil {
				return e
			}
			return saveCommand(tx, child, "save", key, h, out, 0)
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		b, _ := json.Marshal(in)
		now := time.Now().UTC()
		out = Suggestion{SupersedesID: in.SupersedesID, ChildID: child, SubjectCode: in.SubjectCode, Title: in.Title, SchemaVersion: in.SchemaVersion, AnalysisVersion: in.AnalysisVersion, AnalysisAsOf: in.AnalysisAsOf, SpecJSON: string(b), ContentHash: contentHash, Lifecycle: "open", RowVersion: 1, CreatedAt: now, UpdatedAt: now}
		if e = tx.Create(&out).Error; e != nil {
			return e
		}
		for i := range ts {
			ts[i].SuggestionID = out.ID
			if e = tx.Create(&ts[i]).Error; e != nil {
				return e
			}
			for j := range es[i] {
				es[i][j].TargetID = ts[i].ID
				if e = tx.Create(&es[i][j]).Error; e != nil {
					return e
				}
			}
		}
		out, e = New(tx, s.Generation, s.Enabled).Get(ctx, child, out.ID)
		if e != nil {
			return e
		}
		return saveCommand(tx, child, "save", key, h, out, 0)
	})
	// Unique constraints serialize concurrent identical saves without changing the stored request.
	if err != nil {
		var result Suggestion
		var found bool
		result, found, e := replayCommand(s.DB.WithContext(ctx), child, "save", key, h)
		if e == nil && found {
			return result, true, nil
		}
	}
	return
}
func (s *Service) Get(ctx context.Context, child, id int64) (Suggestion, error) {
	var out Suggestion
	q := s.DB.WithContext(ctx)
	if e := q.Where("id=? AND child_id=?", id, child).Take(&out).Error; e != nil {
		return out, e
	}
	out.Targets = []Target{}
	out.Partitions = []Partition{}
	if e := q.Where("suggestion_id=?", id).Order("target_key").Find(&out.Targets).Error; e != nil {
		return out, e
	}
	ids := []int64{}
	for _, t := range out.Targets {
		ids = append(ids, t.KpID)
	}
	var titles []struct {
		ID    int64
		Title string
	}
	if len(ids) > 0 && q.Migrator().HasColumn("knowledge_points", "title") {
		if e := q.Table("knowledge_points").Select("id,title").Where("id IN ?", ids).Scan(&titles).Error; e != nil {
			return out, e
		}
	}
	byTitle := map[int64]string{}
	for _, p := range titles {
		byTitle[p.ID] = p.Title
	}
	for i := range out.Targets {
		out.Targets[i].Title = byTitle[out.Targets[i].KpID]
	}
	for _, t := range out.Targets {
		out.RequestedCount += t.RequestedCount
	}
	var r Run
	if e := q.Where("suggestion_id=?", id).Order("id DESC").Take(&r).Error; e == nil {
		out.GenerationStatus = &r.State
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return out, e
	}
	if e := q.Where("suggestion_id=?", id).Order("partition_key").Find(&out.Partitions).Error; e != nil {
		return out, e
	}
	links, e := s.Tasks(ctx, child, id)
	if e != nil {
		return out, e
	}
	out.TaskCount = len(links)
	for i := range out.Partitions {
		p := &out.Partitions[i]
		if p.WarningsJSON != "" {
			p.Warnings = json.RawMessage(p.WarningsJSON)
		}
		if p.ErrorJSON != "" {
			p.Error = json.RawMessage(p.ErrorJSON)
		}
		for _, l := range links {
			if l.PartitionID == p.ID {
				p.TaskID = l.TaskID
				p.RevisionID = l.GeneratedRevisionID
				var entries []any
				_ = json.Unmarshal([]byte(l.TargetMapJSON), &entries)
				p.GeneratedCount = len(entries)
				out.GeneratedCount += len(entries)
			}
		}
	}
	return out, nil
}

type ListResult struct {
	Items      []Suggestion `json:"items"`
	NextCursor string       `json:"nextCursor"`
	HasMore    bool         `json:"hasMore"`
}

func (s *Service) List(ctx context.Context, child, before int64, limit int) (ListResult, error) {
	if limit < 1 || limit > 100 {
		limit = 30
	}
	out := ListResult{Items: []Suggestion{}}
	var rows []Suggestion
	q := s.DB.WithContext(ctx).Where("child_id=?", child)
	if before > 0 {
		q = q.Where("id<?", before)
	}
	if e := q.Order("id DESC").Limit(limit + 1).Find(&rows).Error; e != nil {
		return out, e
	}
	if len(rows) > limit {
		out.HasMore = true
		rows = rows[:limit]
	}
	for _, row := range rows {
		v, e := s.Get(ctx, child, row.ID)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	if out.HasMore {
		out.NextCursor = fmt.Sprint(rows[len(rows)-1].ID)
	}
	return out, nil
}
func (s *Service) owned(ctx context.Context, child, id int64) error {
	var row Suggestion
	return s.DB.WithContext(ctx).Select("id").Where("id=? AND child_id=?", id, child).Take(&row).Error
}
func (s *Service) Evidence(ctx context.Context, child, id int64) ([]Evidence, error) {
	out := []Evidence{}
	if e := s.owned(ctx, child, id); e != nil {
		return out, e
	}
	e := s.DB.WithContext(ctx).Table("review_suggestion_evidence e").Select("e.*").Joins("JOIN review_suggestion_targets t ON t.id=e.target_id").Where("t.suggestion_id=?", id).Order("e.id").Scan(&out).Error
	for i := range out {
		_ = json.Unmarshal([]byte(out[i].SourceJSON), &out[i].Source)
		out[i].Summary = json.RawMessage(out[i].EvidenceSummaryJSON)
	}
	return out, e
}
func (s *Service) Tasks(ctx context.Context, child, id int64) ([]TaskLink, error) {
	out := []TaskLink{}
	if e := s.owned(ctx, child, id); e != nil {
		return out, e
	}
	e := s.DB.WithContext(ctx).Where("suggestion_id=?", id).Order("id").Find(&out).Error
	for i := range out {
		out[i].TargetMap = json.RawMessage(out[i].TargetMapJSON)
		if e == nil {
			e = s.taskState(ctx, &out[i])
		}
	}
	return out, e
}
func (s *Service) Plans(ctx context.Context, child, id int64) ([]map[string]any, error) {
	out := []map[string]any{}
	if e := s.owned(ctx, child, id); e != nil {
		return out, e
	}
	if !s.DB.Migrator().HasTable("study_plans") {
		return out, nil
	}
	e := s.DB.WithContext(ctx).Table("study_plans p").Select("p.*").Joins("JOIN review_suggestion_tasks t ON t.task_id=p.source_question_task_id").Where("t.suggestion_id=? AND p.child_id=?", id, child).Order("p.id DESC").Find(&out).Error
	return out, e
}
func (s *Service) Command(ctx context.Context, child, id int64, operation, key string, in CommandInput) (out Suggestion, replay bool, err error) {
	if err = s.enabled(key); err != nil {
		return
	}
	if operation != "generate" && operation != "retry" && operation != "cancel" && operation != "archive" {
		return out, false, bad("invalid_command", "unknown command")
	}
	if in.ExpectedRowVersion < 1 {
		return out, false, bad("version_required", "expectedRowVersion is required")
	}
	sort.Strings(in.PartitionKeys)
	h := hash([]any{id, in})
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var e error
		if tx.Dialector.Name() == "postgres" {
			if e = tx.Exec("SELECT pg_advisory_xact_lock(19201, ?)", child).Error; e != nil {
				return e
			}
		}
		out, replay, e = replayCommand(tx, child, operation, key, h)
		if e != nil || replay {
			return e
		}
		var row Suggestion
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND child_id=?", id, child).Take(&row).Error; e != nil {
			return e
		}
		if row.RowVersion != in.ExpectedRowVersion {
			return conflict("version_conflict", "suggestion row version changed")
		}
		var active Run
		e = tx.Where("suggestion_id=? AND state IN ('queued','running')", id).Take(&active).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		hasActive := e == nil
		runID := int64(0)
		switch operation {
		case "generate", "retry":
			if row.Lifecycle != "open" {
				return conflict("archived", "archived suggestion cannot generate")
			}
			if hasActive {
				return conflict("run_active", "generation already active")
			}
			var ps []Partition
			if e = tx.Where("suggestion_id=?", id).Order("partition_key").Find(&ps).Error; e != nil {
				return e
			}
			if operation == "generate" && len(ps) > 0 {
				return conflict("already_generated", "use retry for existing partitions")
			}
			if operation == "retry" && len(in.PartitionKeys) == 0 {
				return bad("partitions_required", "retry requires partitionKeys")
			}
			if operation == "generate" {
				var targets []Target
				if e = tx.Where("suggestion_id=?", id).Order("module_code,target_key").Find(&targets).Error; e != nil {
					return e
				}
				groups := map[string][]string{}
				for _, t := range targets {
					groups[t.ModuleCode] = append(groups[t.ModuleCode], t.TargetKey)
				}
				modules := []string{}
				for m := range groups {
					modules = append(modules, m)
				}
				sort.Strings(modules)
				for _, m := range modules {
					b, _ := json.Marshal(groups[m])
					ps = append(ps, Partition{SuggestionID: id, PartitionKey: m + ":" + strings.Join(groups[m], ","), TargetKeysJSON: string(b), State: "queued", Seed: id, NextRunAt: time.Now().UTC()})
				}
			}
			chosen := map[string]bool{}
			for _, k := range in.PartitionKeys {
				if chosen[k] {
					return bad("invalid_partitions", "duplicate partition")
				}
				chosen[k] = true
			}
			if operation == "retry" {
				for _, p := range ps {
					if chosen[p.PartitionKey] {
						if p.State == "succeeded" {
							return conflict("partition_succeeded", "successful partition cannot be recreated")
						}
						delete(chosen, p.PartitionKey)
					}
				}
				if len(chosen) > 0 {
					return bad("invalid_partitions", "unknown partition")
				}
			}
			run := Run{SuggestionID: id, State: "queued", CreatedAt: time.Now().UTC()}
			if e = tx.Create(&run).Error; e != nil {
				return e
			}
			runID = run.ID
			for i := range ps {
				p := &ps[i]
				if operation == "retry" {
					match := false
					for _, k := range in.PartitionKeys {
						if p.PartitionKey == k {
							match = true
						}
					}
					if !match {
						continue
					}
				}
				p.RunID = run.ID
				p.State = "queued"
				p.ErrorJSON = ""
				p.LeaseUntil = nil
				p.LeaseOwner = ""
				p.NextRunAt = time.Now().UTC()
				p.AttemptCount = 0
				if p.ID == 0 {
					e = tx.Create(p).Error
				} else {
					e = tx.Save(p).Error
				}
				if e != nil {
					return e
				}
			}
		case "cancel":
			if !hasActive {
				return conflict("no_active_run", "no active generation to cancel")
			}
			now := time.Now().UTC()
			// Match claim lock order: partition before run. The suggestion lock guards cancellation against commits.
			if e = tx.Model(&Partition{}).Where("run_id=? AND state IN ('queued','running')", active.ID).Updates(map[string]any{"state": "cancelled", "lease_owner": "", "lease_until": nil}).Error; e != nil {
				return e
			}
			var snapshot []Partition
			if e = tx.Where("suggestion_id=?", id).Order("partition_key").Find(&snapshot).Error; e != nil {
				return e
			}
			result, _ := json.Marshal(snapshot)
			if e = tx.Model(&Run{}).Where("id=?", active.ID).Updates(map[string]any{"cancel_requested": true, "state": "cancelled", "finished_at": now, "result_json": string(result)}).Error; e != nil {
				return e
			}
			runID = active.ID
		case "archive":
			if hasActive {
				return conflict("run_active", "cancel active generation before archiving")
			}
			row.Lifecycle = "archived"
		}
		row.RowVersion++
		row.UpdatedAt = time.Now().UTC()
		if e = tx.Model(&Suggestion{}).Where("id=? AND row_version=?", id, in.ExpectedRowVersion).Updates(map[string]any{"row_version": row.RowVersion, "lifecycle": row.Lifecycle, "updated_at": row.UpdatedAt}).Error; e != nil {
			return e
		}
		out, e = New(tx, s.Generation, s.Enabled).Get(ctx, child, id)
		if e != nil {
			return e
		}
		return saveCommand(tx, child, operation, key, h, out, runID)
	})
	return
}

func (s *Service) taskState(ctx context.Context, link *TaskLink) error {
	if !s.DB.Migrator().HasTable("question_tasks") {
		return nil
	}
	var task struct {
		Status                                string
		ActiveRevisionID, PublishedRevisionID *int64
	}
	e := s.DB.WithContext(ctx).Table("question_tasks").Select("status,active_revision_id,published_revision_id").Where("id=?", link.TaskID).Take(&task).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		link.TaskStatus = "deleted"
		return nil
	}
	if e != nil {
		return e
	}
	link.TaskStatus = task.Status
	link.ActiveRevisionID = task.ActiveRevisionID
	link.PublishedRevisionID = task.PublishedRevisionID
	return nil
}
