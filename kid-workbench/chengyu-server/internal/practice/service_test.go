package practice_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/conchi/chengyu-server/internal/plan"
	"github.com/conchi/chengyu-server/internal/practice"
	"github.com/conchi/chengyu-server/internal/testdb"
	"github.com/conchi/study-learning/mastery"
	"github.com/stretchr/testify/require"
)

func TestAnswerKeepsWrongThenRightOnSameItem(t *testing.T) {
	db := testdb.Open(t, "chengyu-practice")
	detail, err := plan.NewService(db).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "meaning", Count: 2})
	require.NoError(t, err)
	require.Equal(t, "chengyu", detail.Plan.SubjectCode)
	require.Len(t, detail.Items, 2)
	item := detail.Items[0]
	order, err := plan.ParseOrder(item.OptionOrder)
	require.NoError(t, err)
	var answerRaw string
	require.NoError(t, db.Table("plan_items").Where("id=?", item.ID).Select("question_answer").Scan(&answerRaw).Error)
	var ans struct{ Index int }
	require.NoError(t, json.Unmarshal([]byte(answerRaw), &ans))
	correct := 0
	for display, original := range order {
		if original == ans.Index {
			correct = display
		}
	}
	wrong := (correct + 1) % len(order)
	svc := practice.NewService(db, mastery.DefaultConfig())
	first, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-1", OptionIndex: wrong, CostMs: 1000})
	require.NoError(t, err)
	require.False(t, first.Correct)
	require.True(t, first.CanRetry)
	require.Equal(t, "wrong", first.Status)
	require.Nil(t, first.AnswerIndex)
	second, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-2", OptionIndex: correct, CostMs: 800})
	require.NoError(t, err)
	require.True(t, second.Correct)
	require.Equal(t, "correct", second.Status)
	again, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-1", OptionIndex: wrong})
	require.NoError(t, err)
	require.False(t, again.Correct)
	var attempts int64
	require.NoError(t, db.Table("attempts").Count(&attempts).Error)
	require.Equal(t, int64(2), attempts)
	type row struct {
		Selected  string
		IsCorrect bool
	}
	var saved []row
	require.NoError(t, db.Table("attempts").Select("selected,is_correct").Order("id").Scan(&saved).Error)
	require.Len(t, saved, 2)
	require.NotEqual(t, saved[0].Selected, saved[1].Selected)
	require.False(t, saved[0].IsCorrect)
	require.True(t, saved[1].IsCorrect)
	_, err = svc.Finish(context.Background(), 1, detail.Plan.ID)
	require.ErrorIs(t, err, practice.ErrPlanIncomplete)
	rest := detail.Items[1]
	restOrder, err := plan.ParseOrder(rest.OptionOrder)
	require.NoError(t, err)
	var restAnswerRaw string
	require.NoError(t, db.Table("plan_items").Where("id=?", rest.ID).Select("question_answer").Scan(&restAnswerRaw).Error)
	var restAns struct{ Index int }
	require.NoError(t, json.Unmarshal([]byte(restAnswerRaw), &restAns))
	restCorrect := 0
	for display, original := range restOrder {
		if original == restAns.Index {
			restCorrect = display
		}
	}
	_, err = svc.Answer(context.Background(), 1, detail.Plan.ID, rest.ID, practice.AnswerInput{ClientID: "try-3", OptionIndex: restCorrect, CostMs: 400})
	require.NoError(t, err)
	finished, err := svc.Finish(context.Background(), 1, detail.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, "chengyu", finished.Plan.SubjectCode)
	require.Equal(t, "done", finished.Plan.Status)
}

func TestFinishCompletesAfterFirstWrongWithoutRetry(t *testing.T) {
	db := testdb.Open(t, "chengyu-practice-finish")
	detail, err := plan.NewService(db).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "pick", Count: 1})
	require.NoError(t, err)
	item := detail.Items[0]
	var answerRaw string
	require.NoError(t, db.Table("plan_items").Where("id=?", item.ID).Select("question_answer").Scan(&answerRaw).Error)
	var ans struct{ Index int }
	require.NoError(t, json.Unmarshal([]byte(answerRaw), &ans))
	order, err := plan.ParseOrder(item.OptionOrder)
	require.NoError(t, err)
	wrong := 0
	for display, original := range order {
		if original != ans.Index {
			wrong = display
			break
		}
	}
	svc := practice.NewService(db, mastery.DefaultConfig())
	first, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "only-wrong", OptionIndex: wrong, CostMs: 400})
	require.NoError(t, err)
	require.Equal(t, "wrong", first.Status)
	finished, err := svc.Finish(context.Background(), 1, detail.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, "done", finished.Plan.Status)
	require.Equal(t, 1, finished.Plan.DoneCount)
	require.Equal(t, 0, finished.Plan.CorrectCount)
}
