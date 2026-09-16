package taskgen

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLegacyImportPreservesOriginalOptionsWithoutOverwriting(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	require.NoError(t, s.DB.Exec(`CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY,title TEXT);CREATE TABLE questions(id INTEGER PRIMARY KEY,kp_id INTEGER,code TEXT,stem TEXT,options TEXT,answer TEXT);INSERT INTO knowledge_points VALUES(1,'春');INSERT INTO questions VALUES(1,1,'glyph_sense','原题','[{"label":"雨"},{"label":"春"},{"label":"花"},{"label":"草"}]','{"index":1}');`).Error)
	old := Task{SubjectCode: "literacy", Title: "旧题包", ModuleCode: "g1", ModuleName: "第一组", TargetCount: 1, Status: "published", SourceMode: "legacy_pool"}
	require.NoError(t, s.DB.Create(&old).Error)
	require.NoError(t, s.DB.Exec(`INSERT INTO question_task_items(task_id,seq,kp_id,question_id) VALUES(?,1,1,1)`, old.ID).Error)
	fresh, e := s.ImportLegacy(ctx, old.ID, "import-1")
	require.NoError(t, e)
	require.Equal(t, "draft", fresh.Status)
	require.Len(t, fresh.Items, 1)
	q := fresh.Items[0].Snapshot
	require.Equal(t, "雨", q.Options[0].Text)
	require.Equal(t, "kp:1", q.AnswerOptionID)
	require.Equal(t, "原题", q.Prompt)
	require.Equal(t, int64(1), q.SourceQuestionID)
	again, e := s.ImportLegacy(ctx, old.ID, "import-1")
	require.NoError(t, e)
	require.Equal(t, fresh.ID, again.ID)
	var got Task
	require.NoError(t, s.DB.First(&got, old.ID).Error)
	require.Equal(t, "published", got.Status)
}
