package model_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-learning/model"
)

func TestSharedLearningTableNames(t *testing.T) {
	require.Equal(t, "children", (model.Child{}).TableName())
	require.Equal(t, "knowledge_points", (model.KnowledgePoint{}).TableName())
	require.Equal(t, "questions", (model.Question{}).TableName())
	require.Equal(t, "attempts", (model.Attempt{}).TableName())
	require.Equal(t, "mastery_states", (model.MasteryState{}).TableName())
	require.Equal(t, "mastery_skills", (model.MasterySkill{}).TableName())
	require.Equal(t, "daily_stats", (model.DailyStat{}).TableName())
	require.Equal(t, "flower_ledger", (model.FlowerLedger{}).TableName())
}

func TestPlanItemHasPersistedOptionOrder(t *testing.T) {
	got := model.PlanItem{OptionOrder: "2,0,3,1"}
	require.Equal(t, "2,0,3,1", got.OptionOrder)
	require.Equal(t, "plan_items", got.TableName())
}

func TestPlanItemPersistsQuestionSnapshot(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&model.PlanItem{}))

	want := `{"questionId":42,"code":"calc","answerIndex":2}`
	require.NoError(t, gdb.Create(&model.PlanItem{ID: 1, QuestionSnapshot: want}).Error)

	var got model.PlanItem
	require.NoError(t, gdb.First(&got, 1).Error)
	require.JSONEq(t, want, got.QuestionSnapshot)
}
