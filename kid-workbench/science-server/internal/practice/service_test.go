package practice_test

import (
	"context"
	"testing"

	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-science/internal/plan"
	"github.com/conchi/study-science/internal/practice"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAnswerUsesSnapshotHidesFirstAnswerAndReplaysClientID(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(
		&learningmodel.Child{}, &learningmodel.KnowledgePoint{}, &learningmodel.Question{},
		&learningmodel.Attempt{}, &learningmodel.MasteryState{}, &learningmodel.MasterySkill{},
		&learningmodel.DailyStat{}, &learningmodel.FlowerLedger{}, &learningmodel.PlanItem{}, &plan.StudyPlan{},
		&practice.AttemptReceipt{},
	))
	require.NoError(t, gdb.Exec(`CREATE UNIQUE INDEX ux_attempt_client ON attempts(child_id, client_id)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE science_assets(kp_id INTEGER PRIMARY KEY, explanation TEXT, review_status TEXT)`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO subjects VALUES(1,'science'); INSERT INTO modules VALUES(1,1,'animal')`).Error)
	require.NoError(t, gdb.Create(&learningmodel.Child{ID: 1, Name: "小朋友"}).Error)
	require.NoError(t, gdb.Create(&learningmodel.KnowledgePoint{ID: 10, ModuleID: 1, Title: "冬眠", Difficulty: 1}).Error)
	require.NoError(t, gdb.Create(&learningmodel.Question{ID: 20, KpID: 10, Code: "recognize", Options: `[{"label":"熊"},{"label":"燕子"}]`, Answer: `{"index":1}`}).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO science_assets VALUES(10,'后来发布的新解释','published')`).Error)
	require.NoError(t, gdb.Create(&plan.StudyPlan{ID: 30, ChildID: 1, SubjectCode: "science", Status: "active", TargetCount: 1}).Error)
	require.NoError(t, gdb.Create(&learningmodel.PlanItem{
		ID: 40, PlanID: 30, KpID: 10, QuestionID: 20, Status: "pending", OptionOrder: "0,1",
		QuestionOptions: `[{"label":"熊"},{"label":"燕子"}]`, QuestionAnswer: `{"index":0}`,
		Explanation: "", ContentSnapshotVersion: 1,
	}).Error)

	svc := practice.NewService(gdb, mastery.DefaultConfig())
	first, err := svc.Answer(context.Background(), 1, 30, 40, practice.AnswerInput{ClientID: "try-1", OptionIndex: 1, CostMs: 100})
	require.NoError(t, err)
	require.False(t, first.Correct)
	require.True(t, first.CanRetry)
	require.Equal(t, -1, first.AnswerIndex)
	require.Empty(t, first.Explanation)

	second, err := svc.Answer(context.Background(), 1, 30, 40, practice.AnswerInput{ClientID: "try-2", OptionIndex: 0, CostMs: 100})
	require.NoError(t, err)
	require.True(t, second.Correct)
	require.Empty(t, second.Explanation)

	firstReplay, err := svc.Answer(context.Background(), 1, 30, 40, practice.AnswerInput{ClientID: "try-1", OptionIndex: 1, CostMs: 100})
	require.NoError(t, err)
	require.Equal(t, first.Correct, firstReplay.Correct)
	require.Equal(t, first.AnswerIndex, firstReplay.AnswerIndex)
	require.Equal(t, first.CanRetry, firstReplay.CanRetry)
	require.Equal(t, first.Tries, firstReplay.Tries)
	require.Equal(t, first.Status, firstReplay.Status)
	require.Equal(t, first.Mastery.KpID, firstReplay.Mastery.KpID)
	require.Equal(t, first.Mastery.Status, firstReplay.Mastery.Status)
	require.Equal(t, first.Mastery.Attempts, firstReplay.Mastery.Attempts)
	require.True(t, first.Mastery.DueAt.Equal(*firstReplay.Mastery.DueAt))

	replay, err := svc.Answer(context.Background(), 1, 30, 40, practice.AnswerInput{ClientID: "try-2", OptionIndex: 0, CostMs: 100})
	require.NoError(t, err)
	require.Equal(t, second.Correct, replay.Correct)
	require.Equal(t, second.Explanation, replay.Explanation)

	require.NoError(t, gdb.Create(&learningmodel.PlanItem{
		ID: 41, PlanID: 30, KpID: 10, QuestionID: 21, Status: "pending", OptionOrder: "0,1",
		QuestionOptions: `[{"label":"熊"},{"label":"燕子"}]`, QuestionAnswer: `{"index":0}`,
		ContentSnapshotVersion: 1,
	}).Error)
	_, err = svc.Answer(context.Background(), 1, 30, 41, practice.AnswerInput{ClientID: "try-1", OptionIndex: 0, CostMs: 100})
	require.ErrorIs(t, err, practice.ErrClientIDConflict)

	var attempts int64
	require.NoError(t, gdb.Model(&learningmodel.Attempt{}).Count(&attempts).Error)
	require.EqualValues(t, 2, attempts)
	var got plan.StudyPlan
	require.NoError(t, gdb.First(&got, 30).Error)
	require.Equal(t, 1, got.DoneCount)
	require.Equal(t, 1, got.CorrectCount)
}
