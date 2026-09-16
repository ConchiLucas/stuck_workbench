package learning_test

import (
	"fmt"
	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestPinyinThreeSkillsCompleteOnce(t *testing.T) {
	d := newLearningDB(t)
	require.NoError(t, d.AutoMigrate(&model.PinyinMasteryMilestone{}))
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	s := learning.NewService(cfg)
	at := time.Now()
	for i, skill := range []string{"listen", "inword", "shape", "shape"} {
		var result learning.StateDTO
		require.NoError(t, d.Transaction(func(tx *gorm.DB) error {
			var e error
			result, _, e = s.ApplyOne(tx, 1, learning.AttemptInput{ClientID: fmt.Sprint(i), KpID: 10, SkillCode: skill, IsCorrect: true, At: at.Add(time.Duration(i) * time.Second)})
			return e
		}))
		require.Equal(t, i == 2, result.NewlyMastered)
		if i < 2 {
			require.NotEqual(t, "mastered", result.Status)
		}
	}
	var ms model.MasteryState
	require.NoError(t, d.First(&ms).Error)
	require.WithinDuration(t, at.Add(2*time.Second), *ms.MasteredAt, time.Millisecond)
	var n int64
	d.Model(&model.FlowerLedger{}).Count(&n)
	require.Equal(t, int64(1), n)
	var day model.DailyStat
	d.First(&day)
	require.Equal(t, 4, day.Attempts)
	require.Equal(t, 1, day.NewlyMastered)
}
func TestPinyinCrossModuleSkillRollsBack(t *testing.T) {
	d := newLearningDB(t)
	s := learning.NewService(mastery.DefaultConfig())
	err := d.Transaction(func(tx *gorm.DB) error {
		_, _, e := s.ApplyOne(tx, 1, learning.AttemptInput{ClientID: "bad", KpID: 10, SkillCode: "blend", IsCorrect: true})
		return e
	})
	require.Error(t, err)
	var n int64
	d.Model(&model.Attempt{}).Count(&n)
	require.Zero(t, n)
}
func TestPinyinLegacyRewardNotRepeated(t *testing.T) {
	d := newLearningDB(t)
	require.NoError(t, d.AutoMigrate(&model.PinyinMasteryMilestone{}))
	id := int64(10)
	require.NoError(t, d.Create(&model.FlowerLedger{ChildID: 1, Delta: 1, Reason: "mastered", RefType: "knowledge_point", RefID: &id}).Error)
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	s := learning.NewService(cfg)
	for _, skill := range []string{"listen", "inword", "shape"} {
		require.NoError(t, d.Transaction(func(tx *gorm.DB) error {
			_, _, e := s.ApplyOne(tx, 1, learning.AttemptInput{ClientID: skill, KpID: 10, SkillCode: skill, IsCorrect: true})
			return e
		}))
	}
	var n int64
	d.Model(&model.FlowerLedger{}).Count(&n)
	require.Equal(t, int64(1), n)
}

func TestPinyinInferredQuestionSkillCannotCrossModule(t *testing.T) {
	d := newLearningDB(t)
	require.NoError(t, d.Model(&model.Question{}).Where("id=20").Update("code", "blend").Error)
	s := learning.NewService(mastery.DefaultConfig())
	q := int64(20)
	e := d.Transaction(func(tx *gorm.DB) error {
		_, _, e := s.ApplyOne(tx, 1, learning.AttemptInput{ClientID: "cross-question", KpID: 10, QuestionID: &q, IsCorrect: true})
		return e
	})
	require.Error(t, e)
	var n int64
	d.Model(&model.Attempt{}).Count(&n)
	require.Zero(t, n)
}
func TestPinyinUnclassifiedLegacyFactCannotCreateMastery(t *testing.T) {
	d := newLearningDB(t)
	s := learning.NewService(mastery.DefaultConfig())
	for i := 0; i < 8; i++ {
		require.NoError(t, d.Transaction(func(tx *gorm.DB) error {
			state, _, e := s.ApplyOne(tx, 1, learning.AttemptInput{ClientID: fmt.Sprint("legacy", i), KpID: 10, IsCorrect: true})
			require.False(t, state.NewlyMastered)
			return e
		}))
	}
	var n int64
	d.Model(&model.Attempt{}).Count(&n)
	require.Equal(t, int64(8), n)
	d.Model(&model.MasterySkill{}).Count(&n)
	require.Zero(t, n)
}
