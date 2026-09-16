package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/study-task-admin/internal/chengyutask"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
)

func TestChengyuMissingService(t *testing.T) {
	r := NewRouter(Deps{})
	for _, path := range []string{"/api/v1/chengyu/question-tasks", "/api/v1/chengyu/question-tasks/1", "/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 503, w.Code, path)
	}
}

func TestChengyuHTTPGenerationAndMedia(t *testing.T) {
	g, err := db.OpenSQLite("file:chengyu-http-generation?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, chengyutask.Migrate(g))
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER);
		CREATE TABLE chengyu_item_speech(kp_id INTEGER NOT NULL, kind TEXT NOT NULL, text TEXT, sha256 TEXT, data BLOB, PRIMARY KEY(kp_id, kind));
		INSERT INTO subjects VALUES (1,'chengyu');
		INSERT INTO modules VALUES (1,1,'daily','日常成语',1);
		INSERT INTO knowledge_points VALUES
		 (1,1,'一心一意','{"kind":"chengyu","pinyin":"yì xīn yì yì","meaning":"集中精神，做事专心","example":"做作业要一心一意。","wrong":["心思不专一","慢慢来","随便玩玩"]}',1),
		 (2,1,'二话不说','{"kind":"chengyu","pinyin":"èr huà bù shuō","meaning":"不说别的话，立刻行动","example":"听到口令，他二话不说就跑了起来。","wrong":["说很多话","慢慢商量","再想一想"]}',2),
		 (3,1,'三心二意','{"kind":"chengyu","pinyin":"sān xīn èr yì","meaning":"心思不专一，不坚定","example":"学习不能三心二意。","wrong":["一心一意","非常专心","坚持到底"]}',3),
		 (4,1,'五颜六色','{"kind":"chengyu","pinyin":"wǔ yán liù sè","meaning":"形容色彩繁多","example":"花园里开着五颜六色的花。","wrong":["只有一种颜色","黑漆漆的","灰蒙蒙的"]}',4);
		INSERT INTO chengyu_item_speech VALUES (1,'chengyu','一心一意','a',X'00'),(2,'chengyu','二话不说','b',X'00'),(3,'chengyu','三心二意','c',X'00'),(4,'chengyu','五颜六色','d',X'00');
	`).Error)
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3 frozen chengyu"))
	}))
	t.Cleanup(media.Close)
	r := NewRouter(Deps{Chengyu: chengyutask.New(g, media.URL)})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "/api/v1/chengyu/question-tasks", `{"title":"成语题包","types":["meaning","pick"],"count":2}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	var saved chengyutask.Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	require.Len(t, saved.Items, 2)
	reload := request("GET", fmt.Sprintf("/api/v1/chengyu/question-tasks/%d", saved.ID), "")
	require.Equal(t, 200, reload.Code)
	speech := saved.Items[0].Example.SpeechURL
	if speech != "" {
		audio := request("GET", speech, "")
		require.Equal(t, 200, audio.Code)
		require.Contains(t, audio.Header().Get("Cache-Control"), "immutable")
	}
}
