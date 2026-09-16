package service_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/seed"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
)

func TestMathProgressUsesFormalSkillsAndRealAttempts(t *testing.T) {
	attempts, d := newAttemptSvc(t)
	_, seedErr := seed.Questions(d)
	require.NoError(t, seedErr)
	svc := service.NewDashboardService(repo.New(d))
	now := time.Now()
	for _, module := range []string{"add10", "sub10", "shape"} {
		var kp int64
		require.NoError(t, d.Raw(`SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='math' AND m.code=? ORDER BY kp.id LIMIT 1`, module).Scan(&kp).Error)
		codes := []string{"calc", "story"}
		if module == "shape" {
			codes = []string{"find", "name"}
		}
		for i, code := range codes {
			var q model.Question
			require.NoError(t, d.Where("kp_id=? AND code=?", kp, code).First(&q).Error)
			for n := 0; n < 3; n++ {
				_, err := attempts.Report(1, []service.AttemptInput{{ClientID: fmt.Sprintf("math-plan-%s-%s-%d", module, code, n), KpID: kp, QuestionID: &q.ID, IsCorrect: true, Source: "plan", At: now.Add(time.Duration(i*3+n) * time.Second)}})
				require.NoError(t, err)
			}
			matrix, err := svc.Matrix(1, "math")
			require.NoError(t, err)
			var point service.MatrixPoint
			for _, m := range matrix.Modules {
				for _, p := range m.Points {
					if p.ID == kp {
						point = p
					}
				}
			}
			require.Len(t, point.Skills, 2, "module %s must expose formal skills", module)
			require.Equal(t, codes[0], point.Skills[0].Code)
			require.Equal(t, codes[1], point.Skills[1].Code)
			require.Equal(t, (i+1)*3, point.Attempts)
			if i == 0 {
				require.Equal(t, "learning", point.Status)
			} else {
				require.Contains(t, []string{"mastered", "review_due"}, point.Status)
			}
		}
		// Stale KP aggregates cannot inflate practice or override per-skill mastery.
		require.NoError(t, d.Model(&model.MasteryState{}).Where("child_id=1 AND kp_id=?", kp).Updates(map[string]any{"attempts": 99, "correct": 99, "status": "not_started"}).Error)
		detail, err := svc.KpDetail(1, kp)
		require.NoError(t, err)
		require.Equal(t, 6, detail.Attempts)
		require.Len(t, detail.Skills, 2)
		require.Len(t, detail.History, 6)
		require.Equal(t, codes[0], detail.History[0].SkillCode)
		require.Contains(t, []string{"mastered", "review_due"}, detail.Status)
	}
	matrix, err := svc.Matrix(1, "math")
	require.NoError(t, err)
	require.Equal(t, 3, matrix.Subject.Counts.Mastered+matrix.Subject.Counts.ReviewDue)
	require.Equal(t, 400, matrix.TypeProgress["calc"].Total)
	require.Equal(t, 8, matrix.TypeProgress["find"].Total)
	subjects, err := svc.Subjects(1)
	require.NoError(t, err)
	for _, subject := range subjects {
		if subject.Code == "math" {
			require.Equal(t, matrix.Subject, subject)
		}
	}
	overview, err := svc.Overview(1)
	require.NoError(t, err)
	require.Equal(t, 3, overview.Counts.Mastered+overview.Counts.ReviewDue)
}

func TestMathProgressMissingSkillsCannotBeCompletedByLegacyAggregate(t *testing.T) {
	_, d := newAttemptSvc(t)
	kp := mathKpID(t, d)
	now := time.Now()
	require.NoError(t, d.Create(&model.MasteryState{ChildID: 1, KpID: kp, Status: "mastered", Attempts: 99, Correct: 99, MasteredAt: &now}).Error)
	require.NoError(t, d.Create(&model.MasterySkill{ChildID: 1, KpID: kp, SkillCode: "calc", Status: "mastered", Attempts: 3, Correct: 3, UpdatedAt: now}).Error)
	svc := service.NewDashboardService(repo.New(d))
	matrix, err := svc.Matrix(1, "math")
	require.NoError(t, err)
	require.Zero(t, matrix.Subject.Counts.Mastered+matrix.Subject.Counts.ReviewDue)
	require.Zero(t, matrix.Subject.WeekNew)
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.Equal(t, "learning", detail.Status)
	require.Nil(t, detail.MasteredAt)
	require.Zero(t, detail.Attempts)
	overview, err := svc.Overview(1)
	require.NoError(t, err)
	require.Zero(t, overview.Counts.Mastered+overview.Counts.ReviewDue)
}
