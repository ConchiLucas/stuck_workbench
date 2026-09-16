package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/phrasetask"
	"github.com/stretchr/testify/require"
)

func TestPhraseMissingService(t *testing.T) {
	r := NewRouter(Deps{})
	for _, path := range []string{"/api/v1/phrase/question-tasks", "/api/v1/phrase/question-tasks/1", "/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 503, w.Code, path)
	}
}

func TestPhraseHTTPGenerationAndMedia(t *testing.T) {
	g, err := db.OpenSQLite("file:phrase-http-generation?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, phrasetask.Migrate(g))
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER);
		CREATE TABLE phrase_item_speech(kp_id INTEGER PRIMARY KEY, text TEXT, sha256 TEXT, data BLOB);
		INSERT INTO subjects VALUES (1,'phrase');
		INSERT INTO modules VALUES (1,1,'greet','问候',1);
		INSERT INTO knowledge_points VALUES
		 (1,1,'Good morning.','{"kind":"phrase","zh":"早上好。","wrong":["下午好。","晚上好。","晚安。"],"scene":"早上见到老师"}',1),
		 (2,1,'Good afternoon.','{"kind":"phrase","zh":"下午好。","wrong":["早上好。","晚上好。","晚安。"],"scene":"下午见到同学"}',2),
		 (3,1,'Hello!','{"kind":"phrase","zh":"你好！","wrong":["再见。","谢谢。","对不起。"],"scene":"第一次见面打招呼"}',3),
		 (4,1,'Good night.','{"kind":"phrase","zh":"晚安。","wrong":["早上好。","下午好。","你好。"],"scene":"睡觉前跟妈妈说"}',4);
		INSERT INTO phrase_item_speech VALUES (1,'Good morning.','a',X'00'),(2,'Good afternoon.','b',X'00'),(3,'Hello!','c',X'00'),(4,'Good night.','d',X'00');
	`).Error)
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3 frozen phrase"))
	}))
	t.Cleanup(media.Close)
	r := NewRouter(Deps{Phrase: phrasetask.New(g, media.URL)})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "/api/v1/phrase/question-tasks", `{"title":"短句题包","types":["listen_zh","listen_en"],"count":2}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	var saved phrasetask.Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	require.Len(t, saved.Items, 2)
	reload := request("GET", fmt.Sprintf("/api/v1/phrase/question-tasks/%d", saved.ID), "")
	require.Equal(t, 200, reload.Code)
	speech := saved.Items[0].Example.SpeechURL
	audio := request("GET", speech, "")
	require.Equal(t, 200, audio.Code)
	require.Contains(t, audio.Header().Get("Cache-Control"), "immutable")
}
