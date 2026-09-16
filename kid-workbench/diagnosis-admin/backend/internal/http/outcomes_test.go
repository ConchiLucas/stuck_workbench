package httpapi

import (
	"encoding/json"
	"fmt"
	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/conchi/study-diagnosis-admin/internal/reviewclient"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOutcomesSavedSuggestionDoesNotClaimPractice(t *testing.T) {
	db := testdb.Open(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/children/1/review-suggestions/7" {
			fmt.Fprint(w, `{"createdAt":"2026-09-08T00:00:00Z","targets":[{"key":"10:write_char","kpId":10,"questionType":"write_char"}]}`)
		} else if r.URL.Path == "/api/v1/children/1/review-suggestions/7/tasks" {
			fmt.Fprint(w, `{"items":[],"hasMore":false}`)
		} else {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"error":{"code":"not_found"}}`)
		}
	}))
	defer upstream.Close()
	r := NewRouter(Deps{Knowledge: knowledge.New(db), Reviews: reviewclient.New(upstream.URL, true)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge/review-suggestions/7/outcomes", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	var result struct {
		Items []reviewOutcome `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Len(t, result.Items, 1)
	require.Equal(t, "not_generated", result.Items[0].State)
	require.Zero(t, result.Items[0].ObservedAttempts)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/2/knowledge/review-suggestions/7/outcomes", nil))
	require.Equal(t, 404, w.Code)
}

func TestOutcomesSeparatesChangedRevisionAndOtherPractice(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE study_plans ADD COLUMN source_question_task_id INTEGER;ALTER TABLE study_plans ADD COLUMN source_question_task_revision_id INTEGER;ALTER TABLE plan_items ADD COLUMN question_version_id INTEGER;
 CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,attempt_id INTEGER,child_id INTEGER,kp_id INTEGER,skill_code TEXT,question_type TEXT,plan_id INTEGER,plan_item_id INTEGER,question_version_id INTEGER);
 UPDATE study_plans SET source_question_task_id=40,source_question_task_revision_id=700 WHERE id=9;
 UPDATE plan_items SET question_version_id=501 WHERE id=90;
 INSERT INTO study_plans(id,child_id,status,source_question_task_id,source_question_task_revision_id)VALUES(10,1,'done',40,701);
 INSERT INTO plan_items(id,plan_id,kp_id,question_version_id)VALUES(91,10,20,502);
 INSERT INTO question_attempt_receipts VALUES(1,1,1,20,'listen','listen',9,90,501),(2,2,1,20,'listen','listen',10,91,502);`).Error)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/children/1/review-suggestions/7" {
			fmt.Fprint(w, `{"createdAt":"2026-09-06T00:00:00Z","targets":[{"key":"20:listen","kpId":20,"questionType":"listen"}]}`)
		} else {
			fmt.Fprint(w, `{"items":[{"taskId":40,"generatedRevisionId":700,"publishedRevisionId":701,"targetMap":[{"targetKey":"20:listen","questionVersionId":501,"kind":"original"}]}],"hasMore":false}`)
		}
	}))
	defer upstream.Close()
	r := NewRouter(Deps{Knowledge: knowledge.New(db), Reviews: reviewclient.New(upstream.URL, true)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge/review-suggestions/7/outcomes", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	var result struct {
		Items []reviewOutcome `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Len(t, result.Items, 1)
	o := result.Items[0]
	require.Equal(t, 1, o.ObservedAttempts)
	require.Equal(t, 1, o.OriginalAttempts)
	require.Equal(t, int64(1), o.RevisionChangedPlans)
	require.Equal(t, int64(1), o.OtherLaterAttempts)
	require.Equal(t, []int64{1}, o.AttemptIDs)
}
