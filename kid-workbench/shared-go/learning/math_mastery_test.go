package learning_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func mathAnswer(t *testing.T, d *gorm.DB, s *learning.Service, code, client string, correct bool, at time.Time) learning.StateDTO {
	t.Helper()
	var out learning.StateDTO
	require.NoError(t, d.Transaction(func(tx *gorm.DB) error {
		var err error
		out, _, err = s.ApplyOne(tx, 1, learning.AttemptInput{KpID: 10, SkillCode: code, ClientID: client, IsCorrect: correct, At: at, Source: "plan"})
		return err
	}))
	return out
}
func TestMathFullCompletionDateIsSecondSkillDayAndPracticeIsTotal(t *testing.T) {
	d := newMathLearningDB(t, "add10")
	s := learning.NewService(mastery.DefaultConfig())
	first := time.Now().UTC().AddDate(0, 0, -1).Truncate(time.Second)
	for i := 0; i < 3; i++ {
		out := mathAnswer(t, d, s, "calc", fmt.Sprintf("calc-%d", i), true, first.Add(time.Duration(i)*time.Minute))
		require.False(t, out.NewlyMastered)
	}
	var state model.MasteryState
	require.NoError(t, d.First(&state).Error)
	require.Nil(t, state.MasteredAt)
	second := first.AddDate(0, 0, 1)
	for i := 0; i < 3; i++ {
		out := mathAnswer(t, d, s, "story", fmt.Sprintf("story-%d", i), true, second.Add(time.Duration(i)*time.Minute))
		require.Equal(t, i == 2, out.NewlyMastered)
	}
	require.NoError(t, d.First(&state).Error)
	require.NotNil(t, state.MasteredAt)
	require.True(t, state.MasteredAt.Equal(second.Add(2*time.Minute)))
	require.Equal(t, 6, state.Attempts)
	require.Equal(t, 6, state.Correct)
	var ledger []model.FlowerLedger
	require.NoError(t, d.Find(&ledger).Error)
	require.Len(t, ledger, 1)
	require.True(t, ledger[0].CreatedAt.Equal(*state.MasteredAt), "new math reward is dated at the completion event")
	var days []model.DailyStat
	require.NoError(t, d.Order("stat_date").Find(&days).Error)
	require.Len(t, days, 2)
	require.Zero(t, days[0].NewlyMastered)
	require.Equal(t, 1, days[1].NewlyMastered)
}
func TestMathRegainingMasteryDoesNotRepeatNewLearningOrReward(t *testing.T) {
	d := newMathLearningDB(t, "shape")
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	s := learning.NewService(cfg)
	at := time.Now().UTC().Truncate(time.Second)
	require.False(t, mathAnswer(t, d, s, "find", "find", true, at).NewlyMastered)
	require.True(t, mathAnswer(t, d, s, "name", "name", true, at.Add(time.Minute)).NewlyMastered)
	completed := at.Add(time.Minute)
	require.False(t, mathAnswer(t, d, s, "name", "wrong", false, at.Add(2*time.Minute)).NewlyMastered)
	var state model.MasteryState
	require.NoError(t, d.First(&state).Error)
	require.Nil(t, state.MasteredAt)
	for i := 0; i < 5; i++ {
		require.False(t, mathAnswer(t, d, s, "name", fmt.Sprintf("recover-%d", i), true, at.Add(time.Duration(i+3)*time.Minute)).NewlyMastered)
	}
	require.NoError(t, d.First(&state).Error)
	require.Equal(t, "mastered", state.Status)
	require.NotNil(t, state.MasteredAt)
	require.True(t, state.MasteredAt.Equal(completed))
	require.Equal(t, 8, state.Attempts)
	require.Equal(t, 7, state.Correct)
	var n int64
	require.NoError(t, d.Model(&model.FlowerLedger{}).Count(&n).Error)
	require.EqualValues(t, 1, n)
	var daily model.DailyStat
	require.NoError(t, d.First(&daily).Error)
	require.Equal(t, 1, daily.NewlyMastered)
	var child model.Child
	require.NoError(t, d.First(&child).Error)
	require.Equal(t, 1, child.Flowers)
}
func TestMathLegacyCompletionReceiptPreventsRepeatAndKeepsItsDate(t *testing.T) {
	d := newMathLearningDB(t, "sub10")
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	s := learning.NewService(cfg)
	original := time.Now().UTC().AddDate(0, 0, -20).Truncate(time.Second)
	kp := int64(10)
	require.NoError(t, d.Create(&model.FlowerLedger{ChildID: 1, Reason: "mastered", RefType: "knowledge_point", RefID: &kp, Delta: 1, CreatedAt: original}).Error)
	now := time.Now().UTC()
	require.False(t, mathAnswer(t, d, s, "calc", "calc", true, now).NewlyMastered)
	require.False(t, mathAnswer(t, d, s, "story", "story", true, now.Add(time.Minute)).NewlyMastered)
	var state model.MasteryState
	require.NoError(t, d.First(&state).Error)
	require.NotNil(t, state.MasteredAt)
	require.True(t, state.MasteredAt.Equal(original))
	var n int64
	require.NoError(t, d.Model(&model.FlowerLedger{}).Count(&n).Error)
	require.EqualValues(t, 1, n)
	var daily model.DailyStat
	require.NoError(t, d.First(&daily).Error)
	require.Zero(t, daily.NewlyMastered)
}
func TestMathLegacyFullStateWithoutReceiptKeepsCompletionDate(t *testing.T) {
	d := newMathLearningDB(t, "add10")
	s := learning.NewService(mastery.DefaultConfig())
	original := time.Now().UTC().AddDate(0, 0, -20).Truncate(time.Second)
	require.NoError(t, d.Create(&model.MasteryState{ChildID: 1, KpID: 10, Status: "mastered", MasteredAt: &original}).Error)
	for _, code := range []string{"calc", "story"} {
		require.NoError(t, d.Create(&model.MasterySkill{ChildID: 1, KpID: 10, SkillCode: code, Status: "mastered", MasteredAt: &original, Attempts: 3, Correct: 3, Streak: 3}).Error)
	}
	require.False(t, mathAnswer(t, d, s, "story", "review", true, time.Now().UTC()).NewlyMastered)
	var state model.MasteryState
	require.NoError(t, d.First(&state).Error)
	require.NotNil(t, state.MasteredAt)
	require.True(t, state.MasteredAt.Equal(original))
	var n int64
	require.NoError(t, d.Model(&model.FlowerLedger{}).Count(&n).Error)
	require.Zero(t, n)
}
