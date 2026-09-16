package plan_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/phrase-server/internal/plan"
	"github.com/conchi/phrase-server/internal/testdb"
	"github.com/conchi/study-learning/phrasecontent"
)

func TestCreatePlanFiltersByPhraseQuestionCode(t *testing.T) {
	db := testdb.Open(t, "phrase-plan")
	s := plan.NewService(db)
	created, err := s.Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "scene", Count: 8})
	require.NoError(t, err)
	require.Equal(t, "phrase", created.Plan.SubjectCode)
	require.Len(t, created.Items, 1)
	require.Equal(t, "scene", created.Items[0].Question.Code)
	require.Equal(t, "早上见到老师", created.Items[0].Scene)
	require.NoError(t, db.Table("questions").Where("id=?", created.Items[0].Question.ID).Update("stem", "changed").Error)
	again, err := s.Get(context.Background(), 1, created.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, created.Items[0].Question.Stem, again.Items[0].Question.Stem)
}

func TestCreateRejectsUnknownType(t *testing.T) {
	_, err := plan.NewService(testdb.Open(t, "phrase-plan-invalid")).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "listen"})
	require.ErrorIs(t, err, plan.ErrInvalidType)
}

func TestCreateFreezesSpeechAndRollsBackWhenMediaFails(t *testing.T) {
	db := testdb.Open(t, "phrase-plan-freeze")
	audio := []byte("ID3phrase-original")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/speech.mp3") {
			http.NotFound(w, r)
			return
		}
		w.Write(audio)
	}))
	t.Cleanup(server.Close)
	created, err := plan.NewService(db, server.URL).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "listen_zh", Count: 1})
	require.NoError(t, err)
	require.Len(t, created.Items, 1)
	var snap phrasecontent.HistorySnapshot
	require.NoError(t, json.Unmarshal([]byte(mustSnapshot(t, db, created.Items[0].ID)), &snap))
	require.Equal(t, 1, snap.Schema)
	require.True(t, phrasecontent.HasFrozenMedia(snap.Example))
	require.Contains(t, string(created.Items[0].Question.Speech), snap.Example.SpeechURL)
	data, err := plan.NewService(db).FrozenSpeech(context.Background(), strings.TrimPrefix(snap.Example.SpeechURL, phrasecontent.MediaPrefix))
	require.NoError(t, err)
	require.Equal(t, audio, data)

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(failing.Close)
	var before int64
	require.NoError(t, db.Table("study_plans").Count(&before).Error)
	_, err = plan.NewService(db, failing.URL).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "listen_en", Count: 1})
	require.Error(t, err)
	var after int64
	require.NoError(t, db.Table("study_plans").Count(&after).Error)
	require.Equal(t, before, after)
}

func mustSnapshot(t *testing.T, db *gorm.DB, id int64) string {
	t.Helper()
	var raw string
	require.NoError(t, db.Raw("SELECT question_snapshot FROM plan_items WHERE id=?", id).Scan(&raw).Error)
	return raw
}
