package taskgen

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"time"
)

type WorkerState struct {
	ID    string `gorm:"primaryKey"`
	Since time.Time
}

func (WorkerState) TableName() string { return "question_task_worker_state" }

type ReviewJob struct {
	ID            int64      `gorm:"primaryKey" json:"id"`
	ChildID       int64      `gorm:"uniqueIndex:uq_auto_review_job" json:"childId"`
	SourcePlanID  int64      `gorm:"uniqueIndex:uq_auto_review_job" json:"sourcePlanId"`
	PolicyVersion string     `gorm:"uniqueIndex:uq_auto_review_job" json:"policyVersion"`
	SourceTaskID  int64      `json:"sourceTaskId"`
	State         string     `json:"state"`
	Attempts      int        `json:"attempts"`
	RetryRound    int        `json:"retryRound"`
	NextAttemptAt time.Time  `json:"nextAttemptAt"`
	LeaseUntil    *time.Time `json:"leaseUntil"`
	LeaseOwner    string     `json:"-"`
	TaskID        *int64     `json:"taskId"`
	Error         string     `json:"error"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (ReviewJob) TableName() string { return "review_generation_jobs" }

type Worker struct {
	Service *Service
	Enabled bool
}

func NewWorker(s *Service, enabled bool) *Worker { return &Worker{Service: s, Enabled: enabled} }
func (w *Worker) Run(ctx context.Context) {
	if !w.Enabled {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if e := w.Step(ctx, time.Now().UTC()); e != nil {
			log.Printf("review worker: %v", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w *Worker) Step(ctx context.Context, now time.Time) error {
	if !w.Enabled {
		return nil
	}
	s := w.Service
	if !s.DB.Migrator().HasTable("question_attempt_receipts") {
		return nil
	}
	state := WorkerState{ID: "auto_review", Since: now}
	if e := s.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&state).Error; e != nil {
		return e
	}
	if e := s.DB.First(&state, "id = ?", "auto_review").Error; e != nil {
		return e
	}
	var plans []struct{ ID, ChildID, SourceQuestionTaskID int64 }
	if e := s.DB.WithContext(ctx).Raw(`SELECT p.id,p.child_id,p.source_question_task_id FROM study_plans p JOIN question_tasks t ON t.id=p.source_question_task_id WHERE p.status='done' AND p.completed_at>=? AND t.kind='practice' AND t.source_mode='material_template'`, state.Since).Scan(&plans).Error; e != nil {
		return e
	}
	for _, p := range plans {
		job := ReviewJob{ChildID: p.ChildID, SourcePlanID: p.ID, SourceTaskID: p.SourceQuestionTaskID, PolicyVersion: "wrong-answer-v1", State: "pending", NextAttemptAt: now}
		if e := s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&job).Error; e != nil {
			return e
		}
	}
	for processed := 0; processed < 5; processed++ {
		var job ReviewJob
		claimed := false
		token := fmt.Sprintf("%d-%d", now.UnixNano(), processed)
		lease := now.Add(5 * time.Minute)
		e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			e := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("(state = ? AND next_attempt_at <= ?) OR (state = ? AND lease_until < ?)", "pending", now, "running", now).Order("id").First(&job).Error
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			if e != nil {
				return e
			}
			if job.Attempts >= 3 {
				return tx.Model(&job).Updates(map[string]any{"state": "failed", "error": "多次执行中断，请手动重试"}).Error
			}
			claimed = true
			return tx.Model(&job).Updates(map[string]any{"state": "running", "attempts": job.Attempts + 1, "lease_until": lease, "lease_owner": token}).Error
		})
		if e != nil {
			return e
		}
		if !claimed {
			break
		}
		task, e := s.CreateReview(ctx, job.SourceTaskID, ReviewInput{ChildID: job.ChildID, SourcePlanID: job.SourcePlanID, Mode: "mixed"})
		updates := map[string]any{"state": "done", "task_id": task.ID, "lease_until": nil, "error": ""}
		if e != nil {
			updates["task_id"] = nil
			updates["error"] = e.Error()
			var ge *Error
			if errors.As(e, &ge) && ge.Code == "no_wrong_answers" {
				updates["state"] = "skipped"
			} else if job.Attempts < 3 {
				updates["state"] = "pending"
				delay := time.Minute
				if job.Attempts >= 2 {
					delay = 5 * time.Minute
				}
				updates["next_attempt_at"] = now.Add(delay)
			} else {
				updates["state"] = "failed"
			}
		}
		if e = s.DB.Model(&ReviewJob{}).Where("id = ? AND lease_owner = ?", job.ID, token).Updates(updates).Error; e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) RetryReviewJob(ctx context.Context, id int64) error {
	res := s.DB.WithContext(ctx).Model(&ReviewJob{}).Where("id = ? AND state = ?", id, "failed").Updates(map[string]any{"state": "pending", "attempts": 0, "retry_round": gorm.Expr("retry_round + 1"), "next_attempt_at": time.Now().UTC(), "error": "", "lease_until": nil})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return fault(409, "job_not_failed", "只有失败任务可以重试")
	}
	return nil
}
