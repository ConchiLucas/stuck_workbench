package phrasetask

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
	g, err := db.OpenSQLite("file:phrase-task-create?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, Migrate(g))
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
		 (3,1,'How are you?','{"kind":"phrase","zh":"你好吗？","wrong":["再见。","早上好。","谢谢。"],"scene":"想问问朋友好不好"}',3),
		 (4,1,'I''m fine.','{"kind":"phrase","zh":"我很好。","wrong":["我饿了。","我累了。","我不舒服。"],"scene":"别人问你好不好","replyTo":"How are you?"}',4);
		INSERT INTO phrase_item_speech VALUES (1,'Good morning.','a',X'00'),(2,'Good afternoon.','b',X'00'),(3,'How are you?','c',X'00'),(4,'I''m fine.','d',X'00');
	`).Error)
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/speech.mp3") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3 phrase " + r.URL.Path))
	}))
	t.Cleanup(media.Close)
	task, err := New(g, media.URL).Create(context.Background(), CreateInput{Title: "短句真实题包", Types: []string{"listen_zh", "listen_en", "scene", "reply"}, Count: 4})
	require.NoError(t, err)
	require.Len(t, task.Items, 4)
	seen := map[string]bool{}
	for _, item := range task.Items {
		require.NotEmpty(t, item.SourceContentHash)
		require.Equal(t, "knowledge_points", item.SourceTable)
		require.NotEmpty(t, item.Example.AnswerID)
		require.GreaterOrEqual(t, len(item.Example.Options), 4)
		seen[item.Kind] = true
		if item.Kind == "scene" {
			require.Empty(t, item.Example.SpeechURL)
			require.NotEmpty(t, item.Example.Prompt)
			continue
		}
		require.Contains(t, item.Example.SpeechURL, "/api/v1/phrase/task-media/")
		require.NotEmpty(t, item.MediaSHA256)
	}
	require.Equal(t, map[string]bool{"listen_zh": true, "listen_en": true, "scene": true, "reply": true}, seen)
	reload, err := New(g, media.URL).Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, task.Items[0].Example.SpeechURL, reload.Items[0].Example.SpeechURL)
}

func TestCreateFailsWhenSpeechMissing(t *testing.T) {
	g, err := db.OpenSQLite("file:phrase-task-missing?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, Migrate(g))
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER);
		CREATE TABLE phrase_item_speech(kp_id INTEGER PRIMARY KEY, text TEXT, sha256 TEXT, data BLOB);
		INSERT INTO subjects VALUES (1,'phrase');
		INSERT INTO modules VALUES (1,1,'greet','问候',1);
		INSERT INTO knowledge_points VALUES (1,1,'Good morning.','{"kind":"phrase","zh":"早上好。","wrong":["下午好。","晚上好。","晚安。"],"scene":"早上见到老师"}',1);
	`).Error)
	_, err = New(g, "http://127.0.0.1:9").Create(context.Background(), CreateInput{Types: []string{"listen_zh"}, Count: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "整句读音")
}
