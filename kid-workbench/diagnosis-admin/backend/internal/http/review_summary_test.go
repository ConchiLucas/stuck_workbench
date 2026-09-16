package httpapi

import (
	"encoding/json"
	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/http/httptest"
	"testing"
)

func reviewSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE review_suggestions(id INTEGER PRIMARY KEY,child_id INTEGER,lifecycle TEXT);
 CREATE TABLE review_suggestion_tasks(id INTEGER PRIMARY KEY,suggestion_id INTEGER,task_id INTEGER,generated_revision_id INTEGER);
 CREATE TABLE question_tasks(id INTEGER PRIMARY KEY,target_child_id INTEGER,published_revision_id INTEGER);
 CREATE TABLE question_task_revisions(id INTEGER PRIMARY KEY,task_id INTEGER);
 CREATE TABLE question_versions(id INTEGER PRIMARY KEY,revision_id INTEGER,kp_id INTEGER,question_type TEXT);
 CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,attempt_id INTEGER,child_id INTEGER,plan_id INTEGER,plan_item_id INTEGER,question_version_id INTEGER,kp_id INTEGER,question_type TEXT);
 ALTER TABLE study_plans ADD COLUMN source_question_task_id INTEGER;
 ALTER TABLE study_plans ADD COLUMN source_question_task_revision_id INTEGER;
 ALTER TABLE plan_items ADD COLUMN question_version_id INTEGER;`).Error)
}
func TestReviewSummaryMissingSchemaIsUnavailable(t *testing.T) {
	db := testdb.Open(t)
	r := NewRouter(Deps{Knowledge: knowledge.New(db)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge/review-status-summary", nil))
	require.Equal(t, 503, w.Code, w.Body.String())
}
func TestReviewSummaryStagesAndStrictAttribution(t *testing.T) {
	db := testdb.Open(t)
	reviewSchema(t, db)
	r := NewRouter(Deps{Knowledge: knowledge.New(db)})
	get := func() map[string]int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge/review-status-summary", nil))
		require.Equal(t, 200, w.Code, w.Body.String())
		var out map[string]int
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		return out
	}
	require.Equal(t, map[string]int{"pending": 0, "draft": 0, "awaiting": 0, "answered": 0, "total": 0}, get())
	require.NoError(t, db.Exec(`INSERT INTO review_suggestions VALUES(1,1,'open'),(2,1,'open'),(3,1,'open'),(4,1,'open'),(5,1,'archived'),(6,2,'open');
 INSERT INTO question_tasks VALUES(2,1,NULL),(3,1,30),(4,1,41),(5,1,50),(6,2,60);
 INSERT INTO question_task_revisions VALUES(20,2),(30,3),(40,4),(41,4),(50,5),(60,6);
 INSERT INTO review_suggestion_tasks VALUES(2,2,2,20),(3,3,3,30),(4,4,4,40),(5,5,5,50),(6,6,6,60),(7,4,3,30);
 INSERT INTO question_versions VALUES(400,40,10,'write_char'),(410,41,10,'write_char');
 INSERT INTO study_plans(id,child_id,source_question_task_id,source_question_task_revision_id) VALUES(40,1,4,40);
 INSERT INTO plan_items(id,plan_id,kp_id,question_version_id) VALUES(400,40,10,400);
 INSERT INTO question_attempt_receipts VALUES(1,3,1,40,400,400,10,'write_char');`).Error)
	require.Equal(t, map[string]int{"pending": 1, "draft": 1, "awaiting": 1, "answered": 1, "total": 4}, get())
	for _, mutation := range []string{
		`UPDATE question_attempt_receipts SET child_id=2`,
		`UPDATE question_attempt_receipts SET plan_item_id=90`,
		`UPDATE question_attempt_receipts SET question_type='glyph_sense'`,
		`UPDATE study_plans SET child_id=2 WHERE id=40`,
		`UPDATE study_plans SET source_question_task_revision_id=41 WHERE id=40`,
		`UPDATE question_task_revisions SET task_id=2 WHERE id=40`,
		`UPDATE question_attempt_receipts SET question_version_id=410`,
		`UPDATE question_attempt_receipts SET kp_id=20`,
		`UPDATE plan_items SET kp_id=20 WHERE id=400`,
		`UPDATE attempts SET source='parent_mark' WHERE id=3`,
		`UPDATE attempts SET child_id=2 WHERE id=3`,
	} {
		tx := db.Begin()
		require.NoError(t, tx.Error)
		require.NoError(t, tx.Exec(mutation).Error)
		rr := NewRouter(Deps{Knowledge: knowledge.New(tx)})
		w := httptest.NewRecorder()
		rr.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge/review-status-summary", nil))
		require.Equal(t, 200, w.Code)
		var out map[string]int
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		require.Equal(t, 0, out["answered"], mutation)
		require.Equal(t, 2, out["awaiting"], mutation)
		require.NoError(t, tx.Rollback().Error)
	}
	for i := 100; i < 225; i++ {
		require.NoError(t, db.Exec("INSERT INTO review_suggestions VALUES(?,1,'open')", i).Error)
	}
	require.Equal(t, 129, get()["total"])
	require.Equal(t, 126, get()["pending"])
}

func TestReviewStagesBatchAndIsolation(t *testing.T) {
	db := testdb.Open(t)
	reviewSchema(t, db)
	require.NoError(t, db.Exec(`INSERT INTO review_suggestions VALUES(1,1,'open'),(2,1,'open'),(3,2,'open'),(4,1,'archived');INSERT INTO question_tasks VALUES(2,1,NULL);INSERT INTO question_task_revisions VALUES(20,2);INSERT INTO review_suggestion_tasks VALUES(2,2,2,20)`).Error)
	r := NewRouter(Deps{Knowledge: knowledge.New(db)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge/review-stages?ids=1,2,3,4", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	require.JSONEq(t, `{"items":[{"id":1,"stage":"pending"},{"id":2,"stage":"draft"}]}`, w.Body.String())
	for _, q := range []string{"", "ids=0", "ids=1,x", "ids=-1"} {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge/review-stages?"+q, nil))
		require.Equal(t, 400, w.Code, w.Body.String())
	}
}
