package reviewsuggestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/conchi/study-task-admin/internal/taskgen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"time"
)

// Run processes only explicitly requested runs, independently of the old automatic review worker.
func (s *Service) Run(ctx context.Context) {
	if !s.Enabled {
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if e := s.Tick(ctx); e != nil {
			log.Printf("review suggestion worker: %v", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Service) Tick(ctx context.Context) error {
	if !s.Enabled {
		return nil
	}
	var p Partition
	now := time.Now().UTC()
	owner := fmt.Sprintf("%d", now.UnixNano())
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("(state='queued' AND next_run_at<=?) OR (state='running' AND lease_until<?)", now, now).Order("id").Limit(1).Find(&p)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		var r Run
		if e := tx.Where("id=? AND state IN ('queued','running') AND cancel_requested=false", p.RunID).Take(&r).Error; e != nil {
			return e
		}
		until := now.Add(time.Minute)
		if e := tx.Model(&Partition{}).Where("id=?", p.ID).Updates(map[string]any{"state": "running", "lease_owner": owner, "lease_until": until, "attempt_count": p.AttemptCount + 1}).Error; e != nil {
			return e
		}
		p.LeaseOwner = owner
		p.AttemptCount++
		return tx.Model(&Run{}).Where("id=?", p.RunID).Update("state", "running").Error
	})
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil
	}
	if e != nil {
		return e
	}
	// The lease bounds expensive generation; a stale owner cannot commit after another owner or cancellation.
	workCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	renewCtx, stopRenew := context.WithCancel(ctx)
	renewDone := make(chan struct{})
	go func() {
		defer close(renewDone)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				ok, err := s.renewLease(renewCtx, p.ID, owner)
				if err != nil || !ok {
					cancel()
					return
				}
			}
		}
	}()
	defer func() { stopRenew(); <-renewDone }()
	var row Suggestion
	if e = s.DB.WithContext(ctx).First(&row, p.SuggestionID).Error; e != nil {
		return e
	}
	var in Input
	if e = json.Unmarshal([]byte(row.SpecJSON), &in); e != nil {
		return e
	}
	var keys []string
	_ = json.Unmarshal([]byte(p.TargetKeysJSON), &keys)
	wanted := map[string]bool{}
	for _, k := range keys {
		wanted[k] = true
	}
	targets := []taskgen.EvidenceReviewTarget{}
	sources := []taskgen.ReviewSource{}
	module := ""
	for _, t := range in.Targets {
		if !wanted[t.Key] {
			continue
		}
		saved, _, err := validateTarget(s.DB.WithContext(workCtx), row.ChildID, in, t)
		if err != nil {
			e = err
			break
		}
		module = saved.ModuleCode
		target := taskgen.EvidenceReviewTarget{Key: t.Key, KpID: t.KpID, QuestionType: t.QuestionType, Mode: t.Mode, Count: t.RequestedCount, PreferredDistractorKpIDs: t.PreferredDistractorKpIDs}
		if in.SubjectCode != "literacy" {
			e = bad("unsupported_source", "v1 generation only supports literacy version evidence")
			break
		}
		for _, ev := range t.Evidence {
			if ev.Role == "supporting_strength" {
				continue
			}
			if ev.Source.Kind != "literacy_version" {
				e = bad("unsupported_source", "v1 generation requires literacy version evidence")
				break
			}
			v, err := verifyEvidence(s.DB.WithContext(workCtx), row.ChildID, t, ev, in.AnalysisAsOf)
			if err != nil {
				e = err
				break
			}
			var n int64
			if err = s.DB.WithContext(workCtx).Table("study_plans").Where("id=? AND child_id=? AND status='done'", v.PlanID, row.ChildID).Count(&n).Error; err != nil {
				e = err
				break
			}
			if n != 1 {
				e = bad("source_plan_incomplete", "source plan must be completed")
				break
			}
			sources = append(sources, taskgen.ReviewSource{SourceReceiptID: v.ReceiptID, SourceQuestionVersionID: v.QuestionVersionID, Reason: t.ReasonCode})
			// Supporting observations validate conclusions; only target errors and assisted completion seed originals.
			if ev.Role == "target_error" || ev.Role == "assisted_completion" {
				var q generation.Snapshot
				if err = json.Unmarshal([]byte(v.SnapshotJSON), &q); err != nil {
					e = err
					break
				}
				target.Originals = append(target.Originals, taskgen.EvidenceOriginal{VersionID: v.QuestionVersionID, Snapshot: q})
			}
		}
		if e != nil {
			break
		}
		targets = append(targets, target)
	}
	var prepared taskgen.EvidenceReviewPrepared
	if e == nil {
		if s.Generation == nil {
			e = &Error{503, "generation_unavailable", "generation service unavailable"}
		} else {
			prepared, e = s.Generation.PrepareEvidenceReview(workCtx, module, p.Seed, targets)
		}
	}
	stopRenew()
	<-renewDone
	finish := func(tx *gorm.DB) error {
		// Lock suggestion first, matching command lock order; the guard and task commit share this transaction.
		var current Suggestion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, row.ID).Error; err != nil {
			return err
		}
		var live Partition
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&live, p.ID).Error; err != nil {
			return err
		}
		var run Run
		if err := tx.First(&run, p.RunID).Error; err != nil {
			return err
		}
		if live.State != "running" || live.LeaseOwner != owner || live.LeaseUntil == nil || !live.LeaseUntil.After(time.Now()) || run.CancelRequested || run.State == "cancelled" || current.Lifecycle != "open" {
			return nil
		}
		if e != nil {
			code := "generation_failed"
			state := "failed"
			retryable := true
			var f *Error
			var tf *taskgen.Error
			if errors.As(e, &f) {
				code = f.Code
				if f.Status < 500 {
					state = "blocked"
					retryable = false
				}
			}
			if errors.As(e, &tf) {
				code = tf.Code
				if tf.Status < 500 {
					state = "blocked"
					retryable = false
				}
			}
			eb, _ := json.Marshal(map[string]any{"code": code, "message": e.Error(), "retryable": retryable})
			updates := map[string]any{"state": state, "error_json": string(eb), "lease_owner": "", "lease_until": nil}
			if retryable && p.AttemptCount < 3 {
				updates["state"] = "queued"
				delay := 5 * time.Second
				if p.AttemptCount == 2 {
					delay = 30 * time.Second
				}
				updates["next_run_at"] = time.Now().UTC().Add(delay)
			}
			if err := tx.Model(&Partition{}).Where("id=?", p.ID).Updates(updates).Error; err != nil {
				return err
			}
		} else {
			gen := taskgen.New(tx, s.Generation.Materials)
			task, err := gen.CommitEvidenceReview(ctx, row.ChildID, row.Title, module, fmt.Sprintf("suggestion:%d:partition:%d", row.ID, p.ID), p.Seed, prepared, sources)
			if err != nil {
				return err
			}
			for i := range prepared.TargetMap {
				for _, v := range task.Items {
					if v.Seq == prepared.TargetMap[i].Seq {
						prepared.TargetMap[i].QuestionVersionID = v.ID
					}
				}
			}
			warnings, _ := json.Marshal(prepared.Warnings)
			mapping, _ := json.Marshal(prepared.TargetMap)
			if err = tx.Create(&TaskLink{SuggestionID: row.ID, PartitionID: p.ID, TaskID: task.ID, GeneratedRevisionID: *task.ActiveRevisionID, TargetMapJSON: string(mapping)}).Error; err != nil {
				return err
			}
			if err = tx.Model(&Partition{}).Where("id=?", p.ID).Updates(map[string]any{"state": "succeeded", "error_json": "", "warnings_json": string(warnings), "lease_owner": "", "lease_until": nil}).Error; err != nil {
				return err
			}
		}
		var all []Partition
		if err := tx.Where("suggestion_id=?", row.ID).Order("partition_key").Find(&all).Error; err != nil {
			return err
		}
		success, blocked, failed, active := 0, 0, 0, 0
		for _, v := range all {
			switch v.State {
			case "succeeded":
				success++
			case "blocked":
				blocked++
			case "failed":
				failed++
			case "queued", "running":
				active++
			}
		}
		if active == 0 {
			state := "cancelled"
			if success == len(all) {
				state = "succeeded"
			} else if success > 0 {
				state = "partial"
			} else if failed > 0 {
				state = "failed"
			} else if blocked > 0 {
				state = "blocked"
			}
			b, _ := json.Marshal(all)
			if err := tx.Model(&Run{}).Where("id=?", p.RunID).Updates(map[string]any{"state": state, "finished_at": time.Now().UTC(), "result_json": string(b)}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Suggestion{}).Where("id=?", row.ID).Updates(map[string]any{"row_version": gorm.Expr("row_version+1"), "updated_at": time.Now().UTC()}).Error
	}
	commitError := s.DB.WithContext(ctx).Transaction(finish)
	if commitError != nil && ctx.Err() == nil {
		// The entire partition rolled back. Persist a bounded retry using the same lease guard.
		e = &Error{503, "task_commit_failed", "复习题包保存失败，可重试"}
		return s.DB.WithContext(ctx).Transaction(finish)
	}
	return commitError
}

func (s *Service) renewLease(ctx context.Context, id int64, owner string) (bool, error) {
	now := time.Now().UTC()
	r := s.DB.WithContext(ctx).Model(&Partition{}).Where("id=? AND state='running' AND lease_owner=? AND lease_until>? AND run_id IN (SELECT id FROM review_suggestion_runs WHERE state IN ('queued','running') AND cancel_requested=false)", id, owner, now).Update("lease_until", now.Add(time.Minute))
	return r.RowsAffected == 1, r.Error
}
