package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-task-admin/internal/db"
	httpapi "github.com/conchi/study-task-admin/internal/http"
	"github.com/conchi/study-task-admin/internal/qtask"
)

func setupRouter(t *testing.T) http.Handler {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, order_no INT);
CREATE TABLE questions (id INTEGER PRIMARY KEY, kp_id INT, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, speech TEXT, difficulty INT);
CREATE TABLE literacy_assets (
  kp_id INTEGER PRIMARY KEY,
  char_text TEXT NOT NULL,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT ''
);
INSERT INTO subjects(id, code) VALUES (1, 'literacy');
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'g1', '第1组', 1);
INSERT INTO knowledge_points(id, module_id, code, title, order_no) VALUES
 (1,1,'c1','一',1),(2,1,'c2','二',2),(3,1,'c3','三',3),(4,1,'c4','四',4),(5,1,'c5','五',5),
 (6,1,'c6','六',6),(7,1,'c7','七',7),(8,1,'c8','八',8),(9,1,'c9','九',9),(10,1,'c10','十',10);
INSERT INTO questions(id, kp_id, code, type, stem, options, answer, visual, speech, difficulty) VALUES
 (101,1,'glyph_sense','choice','看字图，选出义图','[{"label":"一"},{"label":"二"}]','{"index":0}','','{"text":"一"}',1),
 (102,1,'sense_char','choice','看义图，选出字','[{"label":"一"},{"label":"二"}]','{"index":0}','','{"text":"一"}',1),
 (103,2,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (104,3,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (105,4,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (106,5,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (107,6,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (108,7,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (109,8,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1),
 (110,9,'sense_char','choice','看义图，选出字','[]','{"index":0}','','{}',1),
 (111,10,'glyph_sense','choice','看字图，选出义图','[]','{"index":0}','','{}',1);
INSERT INTO literacy_assets(kp_id, char_text, glyph_image_url) VALUES
 (1,'一','http://localhost:19091/g1.png');
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return httpapi.NewRouter(httpapi.Deps{QTask: qtask.NewService(gdb)})
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	h.ServeHTTP(w, req)
	var out map[string]any
	if w.Body.Len() > 0 && w.Header().Get("Content-Type") != "" || w.Body.Len() > 2 {
		_ = json.Unmarshal(w.Body.Bytes(), &out)
	}
	return w.Code, out
}

func TestHealthz(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, http.MethodGet, "/healthz", nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "ok", body["status"])
}

func TestQuestionTaskHTTPLifecycle(t *testing.T) {
	h := setupRouter(t)

	code, modules := doJSON(t, h, http.MethodGet, "/api/v1/question-tasks/literacy-modules", nil)
	require.Equal(t, http.StatusOK, code)
	require.Nil(t, modules["error"])

	code, created := doJSON(t, h, http.MethodPost, "/api/v1/question-tasks", map[string]any{
		"subjectCode": "literacy",
		"moduleCode":  "g1",
		"title":       "测一卷",
	})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "draft", created["status"])
	require.Equal(t, "测一卷", created["title"])
	id := int64(created["id"].(float64))
	items := created["items"].([]any)
	require.Len(t, items, 10)

	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, httptest.NewRequest(http.MethodGet, "/api/v1/question-tasks?subject=literacy", nil))
	require.Equal(t, http.StatusOK, listW.Code)
	var listed []map[string]any
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listed))
	require.NotEmpty(t, listed)

	path := "/api/v1/question-tasks/" + strconv.FormatInt(id, 10)
	code, detail := doJSON(t, h, http.MethodGet, path, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "测一卷", detail["title"])

	code, _ = doJSON(t, h, http.MethodPost, path+"/publish", nil)
	require.Equal(t, http.StatusOK, code)

	code, body := doJSON(t, h, http.MethodPost, path+"/reshuffle", nil)
	require.Equal(t, http.StatusBadRequest, code)
	require.Contains(t, body["error"], "draft")

	code, _ = doJSON(t, h, http.MethodDelete, path, nil)
	require.Equal(t, http.StatusBadRequest, code)

	code, _ = doJSON(t, h, http.MethodPost, path+"/unpublish", nil)
	require.Equal(t, http.StatusOK, code)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, path, nil))
	require.Equal(t, http.StatusNoContent, w.Code)
}
