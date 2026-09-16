package chengyutask

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
)

func TestCreateFreezesFourKindsFromRealSpeech(t *testing.T) {
	g, err := db.OpenSQLite("file:chengyu-task-create?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, Migrate(g))
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
		if !strings.HasSuffix(r.URL.Path, "/speech.mp3") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3 chengyu " + r.URL.Path))
	}))
	t.Cleanup(media.Close)
	task, err := New(g, media.URL).Create(context.Background(), CreateInput{Title: "成语真实题包", Types: []string{"meaning", "pick", "pinyin", "example"}, Count: 4})
	require.NoError(t, err)
	require.Len(t, task.Items, 4)
	seen := map[string]bool{}
	for _, item := range task.Items {
		require.NotEmpty(t, item.SourceContentHash)
		require.Equal(t, "knowledge_points", item.SourceTable)
		require.NotEmpty(t, item.Example.AnswerID)
		require.GreaterOrEqual(t, len(item.Example.Options), 4)
		seen[item.Kind] = true
		if item.Kind == "meaning" {
			require.Contains(t, item.Example.SpeechURL, "/api/v1/chengyu/task-media/")
			require.NotEmpty(t, item.MediaSHA256)
			continue
		}
		require.Empty(t, item.Example.SpeechURL)
		if item.Kind == "example" {
			require.NotNil(t, item.Example.Blank)
			require.Contains(t, item.Example.Blank.Blanked, "____")
		}
	}
	require.Equal(t, map[string]bool{"meaning": true, "pick": true, "pinyin": true, "example": true}, seen)
	reload, err := New(g, media.URL).Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, task.Items[0].Example.SpeechURL, reload.Items[0].Example.SpeechURL)
}

func TestCreateFailsWhenSpeechMissing(t *testing.T) {
	g, err := db.OpenSQLite("file:chengyu-task-missing?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, Migrate(g))
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER);
		CREATE TABLE chengyu_item_speech(kp_id INTEGER NOT NULL, kind TEXT NOT NULL, text TEXT, sha256 TEXT, data BLOB, PRIMARY KEY(kp_id, kind));
		INSERT INTO subjects VALUES (1,'chengyu');
		INSERT INTO modules VALUES (1,1,'daily','日常成语',1);
		INSERT INTO knowledge_points VALUES (1,1,'一心一意','{"kind":"chengyu","pinyin":"yì xīn yì yì","meaning":"集中精神，做事专心","example":"做作业要一心一意。","wrong":["心思不专一","慢慢来","随便玩玩"]}',1);
	`).Error)
	_, err = New(g, "http://127.0.0.1:9").Create(context.Background(), CreateInput{Types: []string{"meaning"}, Count: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "成语读音")
}
