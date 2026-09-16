package englishtask

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
)

func englishFixture(t *testing.T) (*Service, *httptest.Server) {
	t.Helper()
	g, err := db.OpenSQLite(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	require.NoError(t, Migrate(g))
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT);
		CREATE TABLE english_assets(kp_id INTEGER PRIMARY KEY, word_text TEXT, sense_image_url TEXT, speech_audio_url TEXT, module_code TEXT, module_name TEXT, updated_at TEXT);
		CREATE TABLE english_sentences(id INTEGER PRIMARY KEY, code TEXT UNIQUE, text TEXT, tokens_json TEXT, target_kp_id INTEGER, speech_audio_url TEXT, content_hash TEXT, updated_at TEXT);
		CREATE TABLE english_passages(id INTEGER PRIMARY KEY, code TEXT UNIQUE, passage TEXT, prompt TEXT, answer_kp_id INTEGER, option_kp_ids_json TEXT, content_hash TEXT, updated_at TEXT);
		INSERT INTO subjects VALUES (1,'english');
		INSERT INTO modules VALUES (1,1,'animals','动物');
		INSERT INTO knowledge_points VALUES
		 (1,1,'apple','{"meaningZh":"苹果"}'),
		 (2,1,'dog','{"meaningZh":"小狗"}'),
		 (3,1,'cat','{"meaningZh":"小猫"}'),
		 (4,1,'bird','{"meaningZh":"小鸟"}'),
		 (5,1,'nine','{"meaningZh":"九"}'),
		 (6,1,'white','{"meaningZh":"白色"}'),
		 (7,1,'mouth','{"meaningZh":"嘴巴"}');
		INSERT INTO english_assets VALUES
		 (1,'apple','/s','/a','animals','动物','2026-09-13'),
		 (2,'dog','/s','/a','animals','动物','2026-09-13'),
		 (3,'cat','/s','/a','animals','动物','2026-09-13'),
		 (4,'bird','/s','/a','animals','动物','2026-09-13'),
		 (5,'nine','/s','/a','animals','动物','2026-09-13'),
		 (6,'white','/s','/a','animals','动物','2026-09-13'),
		 (7,'mouth','/s','/a','animals','动物','2026-09-13');
		INSERT INTO english_sentences VALUES
		 (11,'this-is-an-apple','This is an apple','["This","is","an","apple"]',1,'/sent','hash-apple','2026-09-13'),
		 (12,'this-is-a-cat','This is a cat','["This","is","a","cat"]',3,'/sent','hash-cat','2026-09-13'),
		 (13,'this-is-a-dog','This is a dog','["This","is","a","dog"]',2,'/sent','hash-dog','2026-09-13'),
		 (14,'we-see-a-bird','We see a bird','["We","see","a","bird"]',4,'/sent','hash-bird','2026-09-13');
		INSERT INTO english_passages VALUES
		 (21,'lucy-apple','Lucy has a red apple. She puts it on the table.','Lucy 把什么放在桌上？',1,'[1,2,3,4]','pass-apple','2026-09-13'),
		 (22,'tom-dog','Tom has a small dog. He plays with it in the park.','Tom 有什么？',2,'[3,2,4,1]','pass-dog','2026-09-13'),
		 (23,'sam-bird','Sam sees a blue bird. It sits on the tree.','Sam 看见了什么？',4,'[4,1,2,3]','pass-bird','2026-09-13');
	`).Error)
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46}
	wordAudio := []byte("ID3english-word-audio")
	sentenceAudio := []byte("ID3english-sentence-audio-xxxx")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/sentences/") {
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write(sentenceAudio)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".mp3") {
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write(wordAudio)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpeg)
	}))
	t.Cleanup(server.Close)
	return New(g, server.URL), server
}

func TestCreateFiveKindsFreezeRealMedia(t *testing.T) {
	s, _ := englishFixture(t)
	types := []string{"audio-choice", "image-text", "card-builder", "input-gap", "reading-qa"}
	task, err := s.Create(context.Background(), CreateInput{Title: "英语五型", Types: types, Count: 5})
	require.NoError(t, err)
	require.Len(t, task.Items, 5)
	for i, item := range task.Items {
		require.Equal(t, types[i], item.Kind)
		require.NotEmpty(t, item.MediaSHA256)
		require.NotEmpty(t, item.SourceContentHash)
		require.Zero(t, item.SourceRevision)
		require.Equal(t, types[i], item.Example.Kind)
		if item.Kind == "card-builder" {
			require.Equal(t, "english_sentences", item.SourceTable)
			require.Contains(t, item.Example.Answer, " ")
			require.NotContains(t, item.Example.Answer, "This is a nine")
			require.NotContains(t, item.Example.Answer, "This is a white")
			require.Len(t, item.Example.Bank, 4)
			require.Empty(t, item.Example.Options)
			require.Contains(t, item.Example.SpeechURL, "/api/v1/english/task-media/")
		}
		if item.Kind == "input-gap" {
			require.NotEmpty(t, item.Example.Answer)
			require.Contains(t, item.Example.Cue, "/api/v1/english/task-media/")
		}
		if item.Kind == "audio-choice" {
			require.Contains(t, item.Example.SpeechURL, "/api/v1/english/task-media/")
			require.Len(t, item.Example.Options, 4)
		}
		if item.Kind == "reading-qa" {
			require.Equal(t, "english_passages", item.SourceTable)
			require.NotContains(t, item.Example.Passage, "red nine")
			require.NotContains(t, item.Example.Passage, "red mouth")
			require.NotContains(t, item.Example.Passage, "red white")
			require.Contains(t, item.Example.Passage, ".")
		}
	}
	saved, err := s.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, task.Items, saved.Items)
}

func TestCardBuilderSpeechIsNotWordAudio(t *testing.T) {
	s, _ := englishFixture(t)
	task, err := s.Create(context.Background(), CreateInput{Types: []string{"card-builder", "audio-choice"}, Count: 2})
	require.NoError(t, err)
	var sentenceHash, wordHash string
	for _, item := range task.Items {
		if item.Kind == "card-builder" {
			for source, hash := range item.MediaSHA256 {
				require.Contains(t, source, "/sentences/")
				sentenceHash = hash
			}
			require.Equal(t, item.Example.Speech, item.Example.Answer)
			require.NotContains(t, item.Example.SpeechURL, "/words/")
		}
		if item.Kind == "audio-choice" {
			for source, hash := range item.MediaSHA256 {
				if strings.Contains(source, "speech") || strings.HasSuffix(source, ".mp3") {
					wordHash = hash
				}
			}
		}
	}
	require.NotEmpty(t, sentenceHash)
	require.NotEmpty(t, wordHash)
	require.NotEqual(t, sentenceHash, wordHash)
}

func TestGeneratorNeverTemplatesIllegalSentences(t *testing.T) {
	s, _ := englishFixture(t)
	task, err := s.Create(context.Background(), CreateInput{Types: []string{"card-builder", "reading-qa"}, Count: 6})
	require.NoError(t, err)
	require.Len(t, task.Items, 6)
	for _, item := range task.Items {
		require.NotContains(t, item.Example.Answer, "This is a nine")
		require.NotContains(t, item.Example.Answer, "This is a white")
		require.NotContains(t, item.Example.Answer, "This is a mouth")
		require.NotContains(t, item.Example.Passage, "red nine")
		require.NotContains(t, item.Example.Passage, "red white")
		require.NotContains(t, item.Example.Passage, "red mouth")
		if item.Kind == "card-builder" {
			require.Equal(t, "english_sentences", item.SourceTable)
		}
		if item.Kind == "reading-qa" {
			require.Equal(t, "english_passages", item.SourceTable)
		}
	}
}

func TestMissingMediaDoesNotSave(t *testing.T) {
	s, _ := englishFixture(t)
	bad := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(bad.Close)
	s.contentURL = bad.URL
	_, err := s.Create(context.Background(), CreateInput{Types: []string{"audio-choice"}, Count: 1})
	require.Error(t, err)
	rows, err := s.List()
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestInvalidSpecDoesNotSave(t *testing.T) {
	s, _ := englishFixture(t)
	for _, in := range []CreateInput{{Count: 1}, {Count: 1, Types: []string{"listen"}}, {Count: 2, Types: []string{"audio-choice", "audio-choice"}}} {
		_, err := s.Create(context.Background(), in)
		require.Error(t, err)
	}
}

func TestCreateFailsWithoutPreparedSentenceSpeech(t *testing.T) {
	s, _ := englishFixture(t)
	require.NoError(t, s.db.Exec(`UPDATE english_sentences SET speech_audio_url=''`).Error)
	_, err := s.Create(context.Background(), CreateInput{Types: []string{"card-builder"}, Count: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "组句子")
}

func TestGeneratorSourceHasNoWordTemplates(t *testing.T) {
	raw, err := os.ReadFile("generate.go")
	require.NoError(t, err)
	src := string(raw)
	require.NotContains(t, src, "This is a/an")
	require.NotContains(t, src, "Lucy has a red")
	require.NotContains(t, src, "func article(")
	require.Contains(t, src, "/api/v1/english/sentences/")
}
