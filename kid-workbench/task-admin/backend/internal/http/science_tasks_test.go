package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/study-learning/sciencecontent"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/sciencetask"
	"github.com/stretchr/testify/require"
)

func TestScienceMissingService(t *testing.T) {
	r := NewRouter(Deps{})
	for _, path := range []string{"/api/v1/science/question-tasks", "/api/v1/science/question-tasks/1", "/api/v1/science/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 503, w.Code, path)
	}
}

func TestScienceHTTPGenerationAndMedia(t *testing.T) {
	g, err := db.OpenSQLite("file:science-http-generation?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY AUTOINCREMENT, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY AUTOINCREMENT, module_id INTEGER, code TEXT, title TEXT, payload TEXT, difficulty INTEGER, order_no INTEGER);
		CREATE TABLE questions(id INTEGER PRIMARY KEY AUTOINCREMENT, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, difficulty INTEGER);
		CREATE TABLE science_assets(
			kp_id INTEGER PRIMARY KEY, title TEXT, module_code TEXT, module_name TEXT, module_order INTEGER, kp_order INTEGER,
			needs_sense_image INTEGER, needs_sense_image_override INTEGER, glyph_image_url TEXT, sense_image_url TEXT, speech_audio_url TEXT,
			glyph_object_key TEXT, sense_object_key TEXT, speech_object_key TEXT, summary TEXT, explanation TEXT, fun_fact TEXT,
			review_status TEXT, reviewed_at DATETIME, content_version INTEGER, synced_at DATETIME, updated_at DATETIME
		);
		INSERT INTO subjects VALUES (1,'science');
	`).Error)
	require.NoError(t, sciencecontent.EnsureMaterials(g))
	png := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3, 4})
	}))
	t.Cleanup(png.Close)
	require.NoError(t, sciencetask.Migrate(g))
	r := NewRouter(Deps{Science: sciencetask.New(g, png.URL)})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "/api/v1/science/question-tasks", `{"title":"科普题包","types":["choice","match"],"count":2}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	var saved sciencetask.Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	require.Len(t, saved.Items, 2)
	reload := request("GET", fmt.Sprintf("/api/v1/science/question-tasks/%d", saved.ID), "")
	require.Equal(t, 200, reload.Code)
}
