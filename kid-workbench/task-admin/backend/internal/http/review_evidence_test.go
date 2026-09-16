package httpapi

import (
	"encoding/json"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/taskgen"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestReviewEvidenceShowsFrozenSelectedOption(t *testing.T) {
	g, e := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, e)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, taskgen.Migrate(g))
	child := int64(1)
	task := taskgen.Task{Title: "复习", SubjectCode: "literacy", ModuleCode: "g1", Kind: "review", SourceMode: "material_template", TargetChildID: &child, Status: "draft"}
	require.NoError(t, g.Create(&task).Error)
	require.NoError(t, g.Exec(`CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,child_id INTEGER,plan_id INTEGER,question_version_id INTEGER,selected_option_id TEXT,is_correct BOOLEAN,created_at DATETIME)`).Error)
	v := taskgen.QuestionVersion{RevisionID: 1, Seq: 1, KpID: 1, QuestionType: "glyph_sense", SnapshotJSON: `{"targetText":"春","questionType":"glyph_sense","answerOptionId":"kp:1","options":[{"id":"kp:1","kpId":1,"text":"春"},{"id":"kp:2","kpId":2,"text":"雨"}]}`}
	require.NoError(t, g.Create(&v).Error)
	require.NoError(t, g.Exec(`INSERT INTO question_attempt_receipts VALUES(1,1,9,?,'kp:2',false,CURRENT_TIMESTAMP)`, v.ID).Error)
	require.NoError(t, g.Create(&taskgen.ReviewSource{TaskID: task.ID, SourceReceiptID: 1, SourceQuestionVersionID: v.ID, Reason: "wrong-answer-v1"}).Error)
	r := NewRouter(Deps{Generation: taskgen.New(g, nil)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/question-tasks/1/review-evidence", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rows))
	require.Len(t, rows, 1)
	require.Equal(t, "雨", rows[0]["selectedOption"].(map[string]any)["text"])
	require.Equal(t, "春", rows[0]["correctOption"].(map[string]any)["text"])
	require.Equal(t, float64(9), rows[0]["planId"])
}
