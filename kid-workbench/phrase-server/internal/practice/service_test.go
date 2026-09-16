package practice_test

import (
	"context"
	"testing"

	"github.com/conchi/phrase-server/internal/plan"
	"github.com/conchi/phrase-server/internal/practice"
	"github.com/conchi/phrase-server/internal/testdb"
	"github.com/conchi/study-learning/mastery"
	"github.com/stretchr/testify/require"
)

func TestAnswerKeepsWrongThenCorrectSelections(t *testing.T) {
	db := testdb.Open(t, "phrase-practice")
	detail, err := plan.NewService(db).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "listen_zh", Count: 2})
	require.NoError(t, err)
	require.Equal(t, "phrase", detail.Plan.SubjectCode)
	require.Len(t, detail.Items, 2)
	item := detail.Items[0]
	order, err := plan.ParseOrder(item.OptionOrder)
	require.NoError(t, err)
	correct := 0
	for display, original := range order {
		if original == 0 {
			correct = display
		}
	}
	wrong := (correct + 1) % len(order)
	svc := practice.NewService(db, mastery.DefaultConfig())
	first, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-1", OptionIndex: wrong, CostMs: 1000})
	require.NoError(t, err)
	require.False(t, first.Correct)
	require.True(t, first.CanRetry)
	require.Equal(t, "pending", first.Status)
	require.Nil(t, first.AnswerIndex)
	second, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-2", OptionIndex: correct, CostMs: 800})
	require.NoError(t, err)
	require.True(t, second.Correct)
	require.False(t, second.CanRetry)
	require.Equal(t, "correct", second.Status)
	again, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-1", OptionIndex: wrong, CostMs: 1000})
	require.NoError(t, err)
	require.False(t, again.Correct)
	require.Equal(t, "correct", again.Status)
	var rows []struct {
		Selected  string
		IsCorrect bool
	}
	require.NoError(t, db.Table("attempts").Select("selected,is_correct").Where("plan_item_id=?", item.ID).Order("id").Scan(&rows).Error)
	require.Len(t, rows, 2)
	require.False(t, rows[0].IsCorrect)
	require.True(t, rows[1].IsCorrect)
	require.NotEqual(t, rows[0].Selected, rows[1].Selected)
	require.NotEmpty(t, rows[0].Selected)
	_, err = svc.Finish(context.Background(), 1, detail.Plan.ID)
	require.ErrorIs(t, err, practice.ErrPlanIncomplete)
	rest := detail.Items[1]
	restOrder, err := plan.ParseOrder(rest.OptionOrder)
	require.NoError(t, err)
	restCorrect := 0
	for display, original := range restOrder {
		if original == 0 {
			restCorrect = display
		}
	}
	done, err := svc.Answer(context.Background(), 1, detail.Plan.ID, rest.ID, practice.AnswerInput{ClientID: "try-3", OptionIndex: restCorrect, CostMs: 400})
	require.NoError(t, err)
	require.True(t, done.Correct)
	require.False(t, done.CanRetry)
	require.Nil(t, done.AnswerIndex)
	finished, err := svc.Finish(context.Background(), 1, detail.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, "phrase", finished.Plan.SubjectCode)
	require.Equal(t, "done", finished.Plan.Status)
	require.Equal(t, 2, finished.Plan.CorrectCount)
}

func TestAnswerRecordsReplyWrongThenCorrect(t *testing.T) {
	db := testdb.Open(t, "phrase-practice-reply")
	detail, err := plan.NewService(db).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "reply", Count: 1})
	require.NoError(t, err)
	item := detail.Items[0]
	order, err := plan.ParseOrder(item.OptionOrder)
	require.NoError(t, err)
	correct := 0
	for display, original := range order {
		if original == 0 {
			correct = display
		}
	}
	wrong := (correct + 1) % len(order)
	svc := practice.NewService(db, mastery.DefaultConfig())
	first, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "reply-wrong", OptionIndex: wrong, CostMs: 900})
	require.NoError(t, err)
	require.False(t, first.Correct)
	second, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "reply-right", OptionIndex: correct, CostMs: 800})
	require.NoError(t, err)
	require.True(t, second.Correct)
	var rows []struct {
		Selected  string
		IsCorrect bool
	}
	require.NoError(t, db.Table("attempts").Select("selected,is_correct").Where("plan_item_id=?", item.ID).Order("id").Scan(&rows).Error)
	require.Len(t, rows, 2)
	require.False(t, rows[0].IsCorrect)
	require.True(t, rows[1].IsCorrect)
	require.NotEqual(t, rows[0].Selected, rows[1].Selected)
}
