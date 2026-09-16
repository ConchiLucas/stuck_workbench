package httpapi

import (
	"encoding/json"
	"fmt"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/pinyintask"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPinyinMissingService(t *testing.T) {
	r := NewRouter(Deps{})
	for _, path := range []string{"/api/v1/pinyin/question-tasks", "/api/v1/pinyin/question-tasks/1", "/api/v1/pinyin/task-media/123.mp3"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 503, w.Code, path)
	}
}

func TestPinyinHTTPGenerationReloadAndMediaWithoutLearningTables(t *testing.T) {
	g, e := db.OpenSQLite("file:pinyin-http-generation?mode=memory&cache=shared")
	require.NoError(t, e)
	require.NoError(t, pinyintask.Migrate(g))
	require.NoError(t, g.Exec(`CREATE TABLE pinyin_assets(kp_id INTEGER PRIMARY KEY,letter TEXT,module_code TEXT,solo_text TEXT,word_text TEXT)`).Error)
	require.NoError(t, g.Exec(`INSERT INTO pinyin_assets VALUES(1,'b','initial','b','爸'),(2,'p','initial','p','坡'),(3,'m','initial','m','妈'),(4,'f','initial','f','佛')`).Error)
	failed := false
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failed {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte("ID3 frozen audio"))
	}))
	defer media.Close()
	svc := pinyintask.New(g, media.URL)
	r := NewRouter(Deps{Pinyin: svc})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "/api/v1/pinyin/question-tasks", `{"title":"真实素材题包","types":["listen","inword","shape"],"count":3}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	var saved pinyintask.Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	require.Len(t, saved.Items, 3)
	require.NoError(t, g.Exec(`UPDATE pinyin_assets SET word_text='changed'`).Error)
	reload := request("GET", fmt.Sprintf("/api/v1/pinyin/question-tasks/%d", saved.ID), "")
	require.Equal(t, 200, reload.Code)
	var loaded pinyintask.Task
	require.NoError(t, json.Unmarshal(reload.Body.Bytes(), &loaded))
	require.Equal(t, saved.Items, loaded.Items)
	audio := request("GET", saved.Items[0].SpeechURL, "")
	require.Equal(t, 200, audio.Code)
	require.Equal(t, "ID3 frozen audio", audio.Body.String())
	require.Contains(t, audio.Header().Get("Cache-Control"), "immutable")
	failed = true
	bad := request("POST", "/api/v1/pinyin/question-tasks", `{"types":["listen"],"count":2}`)
	require.Equal(t, 400, bad.Code)
	tasks, e := svc.List()
	require.NoError(t, e)
	require.Len(t, tasks, 1)
	var tables []string
	require.NoError(t, g.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables).Error)
	require.ElementsMatch(t, []string{"pinyin_assets", "pinyin_question_tasks", "pinyin_question_task_media"}, tables)
}
