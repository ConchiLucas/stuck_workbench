package pinyintask

import (
	"context"
	"fmt"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	g, e := db.OpenSQLite(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, e)
	require.NoError(t, Migrate(g))
	require.NoError(t, g.AutoMigrate(&letter{}, &syllable{}))
	for i, v := range []string{"b", "p", "m", "f"} {
		require.NoError(t, g.Create(&letter{KpID: int64(i + 1), Letter: v, ModuleCode: "initial", SoloText: v, WordText: "爸"}).Error)
		require.NoError(t, g.Create(&syllable{ID: int64(i + 1), InitialText: v, FinalText: "ā", SyllableText: v + "ā", SpeechURL: fmt.Sprintf("/api/v1/pinyin/syllables/%d/speech.mp3", i+1), Enabled: true}).Error)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte("ID3 real test fixture"))
	}))
	t.Cleanup(server.Close)
	return New(g, server.URL)
}
func TestFrozenFourTypesReload(t *testing.T) {
	s := fixture(t)
	task, e := s.Create(context.Background(), CreateInput{Types: []string{"listen", "inword", "shape", "blend"}, Count: 8})
	require.NoError(t, e)
	require.Len(t, task.Items, 8)
	for i, q := range task.Items {
		require.Equal(t, []string{"listen", "inword", "shape", "blend"}[i%4], q.Type)
		require.Len(t, q.Options, 4)
		seen := map[string]bool{}
		found := false
		for _, o := range q.Options {
			require.False(t, seen[o.ID])
			seen[o.ID] = true
			found = found || o.ID == q.AnswerOptionID
		}
		require.True(t, found)
		require.NotEmpty(t, q.MediaSHA256)
	}
	require.NoError(t, s.db.Model(&letter{}).Where("kp_id > 0").Update("letter", "changed").Error)
	saved, e := s.Get(task.ID)
	require.NoError(t, e)
	require.Equal(t, task.Items, saved.Items)
	list, e := s.List()
	require.NoError(t, e)
	require.Len(t, list, 1)
	require.False(t, s.db.Migrator().HasTable("question_attempts"))
}
func TestMissingMediaDoesNotSave(t *testing.T) {
	s := fixture(t)
	bad := httptest.NewServer(http.NotFoundHandler())
	defer bad.Close()
	s.contentURL = bad.URL
	_, e := s.Create(context.Background(), CreateInput{Types: []string{"blend"}, Count: 1})
	require.ErrorContains(t, e, "音频")
	rows, e := s.List()
	require.NoError(t, e)
	require.Empty(t, rows)
}
func TestInvalidSpecDoesNotSave(t *testing.T) {
	s := fixture(t)
	for _, in := range []CreateInput{{Count: 1}, {Count: 1, Types: []string{"invalid"}}, {Count: 1, Types: []string{"listen", "shape"}}, {Count: 2, Types: []string{"listen", "listen"}}} {
		_, e := s.Create(context.Background(), in)
		require.Error(t, e)
	}
}

func TestVersionedAudioFrozenAndUnsafeURLsRejected(t *testing.T) {
	s := fixture(t)
	good := "/api/v1/pinyin/syllables/1/speech.mp3?v=0123456789abcdef"
	require.NoError(t, s.db.Model(&syllable{}).Where("id = 1").Update("speech_url", good).Error)
	task, e := s.Create(context.Background(), CreateInput{Types: []string{"blend"}, Count: 1})
	require.NoError(t, e)
	for _, o := range task.Items[0].Options {
		require.Contains(t, o.SpeechURL, "/api/v1/pinyin/task-media/")
	}
	hash := task.Items[0].MediaSHA256[good]
	require.Len(t, hash, 64)
	media, e := s.Audio(hash)
	require.NoError(t, e)
	require.Equal(t, "ID3 real test fixture", string(media))
	for _, url := range []string{"http://example.com/speech.mp3", "/api/v1/pinyin/syllables/1/speech.mp3?v=../bad", "/api/v1/pinyin/items/1/speech/solo.mp3?redirect=evil"} {
		_, e := s.fetchAudio(context.Background(), url)
		require.Error(t, e)
	}
}
func TestNonAudioResponseAndRedirectRejected(t *testing.T) {
	s := fixture(t)
	for _, handler := range []http.HandlerFunc{func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("<html>not media</html>")) }, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://example.com", 302) }} {
		server := httptest.NewServer(handler)
		s.contentURL = server.URL
		_, e := s.Create(context.Background(), CreateInput{Types: []string{"listen"}, Count: 1})
		require.Error(t, e)
		server.Close()
		list, e := s.List()
		require.NoError(t, e)
		require.Empty(t, list)
	}
}
