package taskgen

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestWorkerDisabledDoesNotTouchLearningTables(t *testing.T) {
	s := setup(t)
	w := NewWorker(s, false)
	require.NoError(t, w.Step(context.Background(), time.Now()))
}
func TestWorkerUsesDurableJobsAndDoesNotPublish(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, s.DB.Exec(`CREATE TABLE study_plans(id INTEGER PRIMARY KEY,child_id INTEGER,source_question_task_id INTEGER,status TEXT,completed_at DATETIME);CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,child_id INTEGER,plan_id INTEGER,question_version_id INTEGER,kp_id INTEGER,question_type TEXT,selected_option_id TEXT,is_correct BOOLEAN,created_at DATETIME);`).Error)
	task, e := s.Create(ctx, "普通", testSpec())
	require.NoError(t, e)
	require.NoError(t, s.DB.Create(&WorkerState{ID: "auto_review", Since: now.Add(-time.Hour)}).Error)
	require.NoError(t, s.DB.Exec(`INSERT INTO study_plans VALUES(1,1,?,'done',?)`, task.ID, now.Add(-time.Minute)).Error)
	w := NewWorker(s, true)
	require.NoError(t, w.Step(ctx, now))
	require.NoError(t, w.Step(ctx, now.Add(time.Minute)))
	var jobs []ReviewJob
	require.NoError(t, s.DB.Find(&jobs).Error)
	require.Len(t, jobs, 1)
	require.Equal(t, "skipped", jobs[0].State)
	var count int64
	s.DB.Model(&Task{}).Where("kind='review'").Count(&count)
	require.Zero(t, count)
}

func setupFailingWorker(t *testing.T) (*Service, *Worker) {
	t.Helper()
	s := setup(t)
	require.NoError(t, s.DB.Exec(`CREATE TABLE study_plans(id INTEGER PRIMARY KEY,child_id INTEGER,source_question_task_id INTEGER,status TEXT,completed_at DATETIME);CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,child_id INTEGER,plan_id INTEGER,question_version_id INTEGER,kp_id INTEGER,question_type TEXT,selected_option_id TEXT,is_correct BOOLEAN,created_at DATETIME);`).Error)
	return s, NewWorker(s, true)
}

func TestWorkerStopsAfterThreeAttemptsAndAllowsManualRetry(t *testing.T) {
	s, w := setupFailingWorker(t)
	ctx := context.Background()
	now := time.Now().UTC()
	job := ReviewJob{ChildID: 1, SourcePlanID: 99, SourceTaskID: 1, PolicyVersion: "wrong-answer-v1", State: "pending", NextAttemptAt: now}
	require.NoError(t, s.DB.Create(&job).Error)
	read := func() ReviewJob {
		t.Helper()
		var row ReviewJob
		require.NoError(t, s.DB.First(&row, job.ID).Error)
		return row
	}
	require.NoError(t, w.Step(ctx, now))
	row := read()
	require.Equal(t, 1, row.Attempts)
	require.Equal(t, "pending", row.State)
	require.WithinDuration(t, now.Add(time.Minute), row.NextAttemptAt, time.Millisecond)
	require.NoError(t, w.Step(ctx, now.Add(59*time.Second)))
	require.Equal(t, 1, read().Attempts)
	require.NoError(t, w.Step(ctx, now.Add(time.Minute)))
	row = read()
	require.Equal(t, 2, row.Attempts)
	require.Equal(t, "pending", row.State)
	require.WithinDuration(t, now.Add(6*time.Minute), row.NextAttemptAt, time.Millisecond)
	require.NoError(t, w.Step(ctx, now.Add(6*time.Minute)))
	row = read()
	require.Equal(t, 3, row.Attempts)
	require.Equal(t, "failed", row.State)
	require.NotEmpty(t, row.Error)
	require.NoError(t, w.Step(ctx, now.Add(time.Hour)))
	require.Equal(t, 3, read().Attempts)
	require.NoError(t, s.RetryReviewJob(ctx, job.ID))
	row = read()
	require.Equal(t, 0, row.Attempts)
	require.Equal(t, 1, row.RetryRound)
	require.Equal(t, "pending", row.State)
	require.NoError(t, w.Step(ctx, now.Add(2*time.Hour)))
	require.Equal(t, 1, read().Attempts)
}

func TestWorkerRecoversExpiredLeaseWithoutExceedingAttemptLimit(t *testing.T) {
	s, w := setupFailingWorker(t)
	ctx := context.Background()
	now := time.Now().UTC()
	expired := now.Add(-time.Minute)
	job := ReviewJob{ChildID: 1, SourcePlanID: 99, SourceTaskID: 1, PolicyVersion: "wrong-answer-v1", State: "running", Attempts: 1, LeaseUntil: &expired, LeaseOwner: "dead-process"}
	require.NoError(t, s.DB.Create(&job).Error)
	require.NoError(t, w.Step(ctx, now))
	var row ReviewJob
	require.NoError(t, s.DB.First(&row, job.ID).Error)
	require.Equal(t, 2, row.Attempts)
	require.Equal(t, "pending", row.State)
	require.NotEqual(t, "dead-process", row.LeaseOwner)
	require.Nil(t, row.LeaseUntil)
	require.WithinDuration(t, now.Add(5*time.Minute), row.NextAttemptAt, time.Millisecond)
	require.NoError(t, s.DB.Model(&ReviewJob{}).Where("id = ?", job.ID).Updates(map[string]any{"state": "running", "attempts": 3, "lease_until": expired}).Error)
	require.NoError(t, w.Step(ctx, now))
	require.NoError(t, s.DB.First(&row, job.ID).Error)
	require.Equal(t, 3, row.Attempts)
	require.Equal(t, "failed", row.State)
}
