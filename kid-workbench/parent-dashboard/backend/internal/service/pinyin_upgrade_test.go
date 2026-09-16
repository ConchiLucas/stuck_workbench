package service_test

import (
	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPinyinUpgradeDryRunAndApplyPreserveFacts(t *testing.T) {
	_, d := newAttemptSvc(t)
	var id int64
	d.Raw(`SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='pinyin' ORDER BY kp.id LIMIT 1`).Scan(&id)
	at := time.Now().AddDate(0, 0, -2)
	require.NoError(t, d.Create(&model.MasteryState{ChildID: 1, KpID: id, Status: "mastered", MasteredAt: &at, Ease: 2.5}).Error)
	for _, code := range []string{"listen", "inword"} {
		require.NoError(t, d.Create(&model.MasterySkill{ChildID: 1, KpID: id, SkillCode: code, Status: "mastered", Attempts: 3, Correct: 3}).Error)
	}
	r, e := service.UpgradePinyin(d, false)
	require.NoError(t, e)
	require.Equal(t, 1, r.ChangedToPartial)
	var n int64
	d.Model(&learningmodel.PinyinUpgradeAudit{}).Count(&n)
	require.Zero(t, n)
	var ms model.MasteryState
	d.First(&ms)
	require.Equal(t, "mastered", ms.Status)
	require.NotNil(t, ms.MasteredAt)
	r, e = service.UpgradePinyin(d, true)
	require.NoError(t, e)
	require.Equal(t, 1, r.Points)
	ms = model.MasteryState{}
	d.First(&ms)
	require.Equal(t, "learning", ms.Status)
	require.Nil(t, ms.MasteredAt)
	r, e = service.UpgradePinyin(d, true)
	require.NoError(t, e)
	require.Equal(t, 0, r.Points)
	require.Equal(t, 1, r.AlreadyApplied)
	for _, table := range []string{"attempts", "daily_stats", "flower_ledger"} {
		d.Table(table).Count(&n)
		require.Zero(t, n, table)
	}
}
