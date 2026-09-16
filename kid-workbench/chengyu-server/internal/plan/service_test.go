package plan_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/conchi/chengyu-server/internal/plan"
	"github.com/conchi/chengyu-server/internal/testdb"
	"github.com/stretchr/testify/require"
)

func TestCreatePlanFiltersByChengyuQuestionCode(t *testing.T) {
	db := testdb.Open(t, "chengyu-plan")
	s := plan.NewService(db)
	created, err := s.Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "meaning", Count: 4})
	require.NoError(t, err)
	require.Equal(t, "chengyu", created.Plan.SubjectCode)
	require.Len(t, created.Items, 4)
	require.Equal(t, "meaning", created.Items[0].Question.Code)
	require.Equal(t, "一心一意", created.Items[0].Chengyu)
	require.NoError(t, db.Table("questions").Where("id=?", created.Items[0].Question.ID).Update("stem", "changed").Error)
	again, err := s.Get(context.Background(), 1, created.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, created.Items[0].Question.Stem, again.Items[0].Question.Stem)
}

func TestCreatePinyinFallsBackToKnowledgePointPayload(t *testing.T) {
	db := testdb.Open(t, "chengyu-plan-pinyin")
	created, err := plan.NewService(db).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "pinyin", Count: 4})
	require.NoError(t, err)
	require.Equal(t, "pinyin", created.Items[0].Question.Code)
	var visual struct{ Kind, Text string }
	require.NoError(t, json.Unmarshal(created.Items[0].Question.Visual, &visual))
	require.Equal(t, "pinyin", visual.Kind)
	require.NotEmpty(t, visual.Text)
	require.Equal(t, created.Items[0].Pinyin, visual.Text)
}

func TestCreateIgnoresOtherSubjects(t *testing.T) {
	created, err := plan.NewService(testdb.Open(t, "chengyu-plan-subject")).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "meaning", Count: 8})
	require.NoError(t, err)
	for _, item := range created.Items {
		require.NotEqual(t, "Good morning.", item.Chengyu)
	}
}

func TestCreatePlanWritesSchemaSnapshot(t *testing.T) {
	db := testdb.Open(t, "chengyu-plan-snapshot")
	created, err := plan.NewService(db).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "example", Count: 1})
	require.NoError(t, err)
	require.Equal(t, "example", created.Items[0].Question.Code)
	var visual struct {
		Text, Full, Blanked, Target string
		Start, Length               int
	}
	require.NoError(t, json.Unmarshal(created.Items[0].Question.Visual, &visual))
	require.Equal(t, "做作业要____。", visual.Text)
	require.Equal(t, "做作业要一心一意。", visual.Full)
	var raw string
	require.NoError(t, db.Table("plan_items").Select("question_snapshot").Where("id=?", created.Items[0].ID).Scan(&raw).Error)
	require.Contains(t, raw, `"schema":1`)
	require.Contains(t, raw, `"kind":"example"`)
	require.Contains(t, string(created.Items[0].Question.Options), `"id"`)
}

