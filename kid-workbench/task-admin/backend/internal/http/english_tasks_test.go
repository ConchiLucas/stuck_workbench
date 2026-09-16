package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/englishtask"
	"github.com/stretchr/testify/require"
)

func TestEnglishMissingService(t *testing.T) {
	r := NewRouter(Deps{})
	for _, path := range []string{"/api/v1/english/question-tasks", "/api/v1/english/question-tasks/1", "/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 503, w.Code, path)
	}
}

func TestEnglishHTTPGenerationReloadAndMedia(t *testing.T) {
	g, err := db.OpenSQLite("file:english-http-generation?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, englishtask.Migrate(g))
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT);
		CREATE TABLE english_assets(kp_id INTEGER PRIMARY KEY, word_text TEXT, sense_image_url TEXT, speech_audio_url TEXT);
		INSERT INTO subjects VALUES (1,'english');
		INSERT INTO modules VALUES (1,1,'animals','动物');
		INSERT INTO knowledge_points VALUES (1,1,'apple','{"meaningZh":"苹果"}'),(2,1,'dog','{"meaningZh":"小狗"}'),(3,1,'cat','{"meaningZh":"小猫"}'),(4,1,'bird','{"meaningZh":"小鸟"}');
		INSERT INTO english_assets VALUES (1,'apple','/s','/a'),(2,'dog','/s','/a'),(3,'cat','/s','/a'),(4,'bird','/s','/a');
		CREATE TABLE english_sentences(id INTEGER PRIMARY KEY, code TEXT UNIQUE, text TEXT, tokens_json TEXT, target_kp_id INTEGER, speech_audio_url TEXT, content_hash TEXT);
		CREATE TABLE english_passages(id INTEGER PRIMARY KEY, code TEXT UNIQUE, passage TEXT, prompt TEXT, answer_kp_id INTEGER, option_kp_ids_json TEXT, content_hash TEXT);
		INSERT INTO english_sentences VALUES (11,'this-is-an-apple','This is an apple','["This","is","an","apple"]',1,'/sent','hash-apple');
		INSERT INTO english_passages VALUES (21,'lucy-apple','Lucy has a red apple. She puts it on the table.','Lucy 把什么放在桌上？',1,'[1,2,3,4]','pass-apple');
	`).Error)
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46}
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".mp3") {
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("ID3 frozen english"))
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpeg)
	}))
	t.Cleanup(media.Close)
	r := NewRouter(Deps{English: englishtask.New(g, media.URL)})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "/api/v1/english/question-tasks", `{"title":"真实素材题包","types":["audio-choice","image-text"],"count":2}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	var saved englishtask.Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	require.Len(t, saved.Items, 2)
	for _, item := range saved.Items {
		require.NotEmpty(t, item.SourceContentHash)
		require.Zero(t, item.SourceRevision)
	}
	reload := request("GET", fmt.Sprintf("/api/v1/english/question-tasks/%d", saved.ID), "")
	require.Equal(t, 200, reload.Code)
	var loaded englishtask.Task
	require.NoError(t, json.Unmarshal(reload.Body.Bytes(), &loaded))
	require.Equal(t, saved.Items, loaded.Items)
	speech := saved.Items[0].Example.SpeechURL
	if speech == "" {
		speech = saved.Items[0].Example.Options[0].Picture
	}
	audio := request("GET", speech, "")
	require.Equal(t, 200, audio.Code)
	require.Contains(t, audio.Header().Get("Cache-Control"), "immutable")
}
