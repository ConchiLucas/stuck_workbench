package service_test

import (
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPinyinMatrixMissingShapeAndEmptySyllableCatalog(t *testing.T) {
	_, d := newAttemptSvc(t)
	svc := service.NewDashboardService(repo.New(d))
	var id int64
	d.Raw(`SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='pinyin' ORDER BY kp.id LIMIT 1`).Scan(&id)
	for _, code := range []string{"listen", "inword"} {
		require.NoError(t, d.Create(&model.MasterySkill{ChildID: 1, KpID: id, SkillCode: code, Status: "mastered", Attempts: 3, Correct: 3, UpdatedAt: time.Now()}).Error)
	}
	out, e := svc.Matrix(1, "pinyin")
	require.NoError(t, e)
	require.Equal(t, 2, out.RuleVersion)
	require.False(t, out.CatalogAvailable)
	var matched bool
	for _, m := range out.Modules {
		for _, p := range m.Points {
			if p.ID == id {
				matched = true
				require.Len(t, p.Skills, 3)
				require.Equal(t, "learning", p.Status)
				require.Equal(t, "letter", p.Kind)
			}
		}
	}
	require.True(t, matched)
	detail, e := svc.KpDetail(1, id)
	require.NoError(t, e)
	require.Len(t, detail.Skills, 3)
	require.Nil(t, detail.MasteredAt)
}
