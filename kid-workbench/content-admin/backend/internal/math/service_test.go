package math_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/math"
	"github.com/conchi/study-content-admin/internal/storage"
)

func setupMathDB(t *testing.T) *math.Service {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'math', '算术', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'add10', '20以内加法', 0),
 (2, 1, 'sub10', '20以内减法', 1),
 (3, 1, 'shape', '认识图形', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (1, 1, '1p2', '1+2', '{"kind":"add","a":1,"b":2}', 1, 0),
 (2, 2, '5m3', '5-3', '{"kind":"sub","a":5,"b":3}', 2, 0),
 (3, 3, 's1', '圆形', '{}', 1, 0),
 (99, 1, 'gone', '旧题', '{"kind":"add","a":1,"b":1}', 1, 9);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return math.NewService(gdb, nil, nil, nil, nil)
}

func TestSyncAndListGroups(t *testing.T) {
	svc := setupMathDB(t)
	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 4, res.Total)
	require.Equal(t, 4, res.Upserted)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 4, list.Total)
	require.Len(t, list.Groups, 3)
	require.Equal(t, "add10", list.Groups[0].ModuleCode)
	require.Equal(t, "1+2", list.Groups[0].Items[0].Title)
	require.Equal(t, "add", list.Groups[0].Items[0].Kind)
	require.Equal(t, "sub", list.Groups[1].Items[0].Kind)
	require.Equal(t, "圆形", list.Groups[2].Items[0].Title)
}

func TestSyncPrunesRemovedKPs(t *testing.T) {
	svc := setupMathDB(t)
	_, err := svc.Sync()
	require.NoError(t, err)

	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`DELETE FROM knowledge_points WHERE id = 99`).Error)

	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 3, res.Total)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 3, list.Total)
	for _, g := range list.Groups {
		for _, it := range g.Items {
			require.NotEqual(t, int64(99), it.KpID)
		}
	}
}

func TestSyncSetsSpeechText(t *testing.T) {
	svc := setupMathDB(t)
	_, err := svc.Sync()
	require.NoError(t, err)
	list, err := svc.List("groups")
	require.NoError(t, err)
	byTitle := map[string]string{}
	for _, g := range list.Groups {
		for _, it := range g.Items {
			byTitle[it.Title] = it.SpeechText
		}
	}
	require.Equal(t, "一加二", byTitle["1+2"])
	require.Equal(t, "五减三", byTitle["5-3"])
	require.Equal(t, "圆形", byTitle["圆形"])
}

func TestBatchGenerateSpeechRequiresModule(t *testing.T) {
	svc := setupMathDB(t)
	_, err := svc.BatchGenerateSpeech(context.Background(), "")
	require.Error(t, err)
}

type failoverSpeech struct {
	calls  []string
	failID string
}

func (s *failoverSpeech) Synthesize(_ context.Context, provider configclient.AIProvider, text string) ([]byte, error) {
	s.calls = append(s.calls, provider.ID)
	if provider.ID == s.failID {
		return nil, fmt.Errorf("TTS 上游返回 503: No available Grok accounts")
	}
	return []byte("mp3-" + text), nil
}

type twoTTSVoices struct{}

func (twoTTSVoices) LoadVoiceModels(context.Context) (configclient.AIConfiguration, error) {
	return configclient.AIConfiguration{
		ActiveProviderID: "grok-tts",
		Providers: []configclient.AIProvider{
			{ID: "grok-tts", Type: "grok-tts", Enabled: true, BaseURL: "http://grok", Capabilities: []string{"AUDIO_TTS"}},
			{ID: "xiaomi-mimo-tts", Type: "mimo-tts", Enabled: true, BaseURL: "http://mimo", Capabilities: []string{"AUDIO_TTS"}},
		},
	}, nil
}

func TestRegenerateSpeechFallsBackWhenPreferredTTSFails(t *testing.T) {
	svc := setupMathDB(t)
	_, err := svc.Sync()
	require.NoError(t, err)
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	store := &questionStore{}
	speech := &failoverSpeech{failID: "grok-tts"}
	s := math.NewService(gdb, store, nil, twoTTSVoices{}, speech)
	dto, err := s.RegenerateSpeech(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, []string{"grok-tts", "xiaomi-mimo-tts"}, speech.calls)
	require.Contains(t, dto.SpeechAudioURL, "/speech.mp3")
	require.NotEmpty(t, store.LastData)
}

type questionStore struct {
	LastKey  string
	LastData []byte
	files    map[string][]byte
}

func (s *questionStore) PutBytes(_ context.Context, key string, data []byte, _ string) (string, error) {
	copied := append([]byte(nil), data...)
	if s.files == nil {
		s.files = map[string][]byte{}
	}
	s.files[key] = copied
	s.LastKey = key
	s.LastData = copied
	return key, nil
}

func (s *questionStore) GetBytes(_ context.Context, key string) ([]byte, error) {
	if s.files != nil {
		if data, ok := s.files[key]; ok {
			return append([]byte(nil), data...), nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

type questionVoices struct{}

func (questionVoices) LoadVoiceModels(context.Context) (configclient.AIConfiguration, error) {
	return configclient.AIConfiguration{
		ActiveProviderID: "voice",
		Providers: []configclient.AIProvider{{
			ID: "voice", Type: "openai-compatible", Enabled: true,
			BaseURL: "http://voice", Model: "tts", Capabilities: []string{"AUDIO_TTS"},
		}},
	}, nil
}

type questionSpeech struct{ LastText string }

func (s *questionSpeech) Synthesize(_ context.Context, _ configclient.AIProvider, text string) ([]byte, error) {
	s.LastText = text
	return []byte("mp3-" + text), nil
}

func setupMathQuestionSpeechService(t *testing.T) (*math.Service, *gorm.DB, *questionStore, *questionSpeech) {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT NOT NULL);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER NOT NULL, code TEXT NOT NULL, order_no INTEGER NOT NULL DEFAULT 0);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER NOT NULL, order_no INTEGER NOT NULL DEFAULT 0);
CREATE TABLE questions (
 id INTEGER PRIMARY KEY, kp_id INTEGER NOT NULL, code TEXT NOT NULL,
 speech TEXT NOT NULL DEFAULT '{}', media_url TEXT NOT NULL DEFAULT ''
);
INSERT INTO subjects(id, code) VALUES (1, 'math'), (2, 'literacy');
INSERT INTO modules(id, subject_id, code, order_no) VALUES (1, 1, 'add10', 1), (2, 2, 'g1', 1);
INSERT INTO knowledge_points(id, module_id, order_no) VALUES (10, 1, 1), (11, 2, 1);
INSERT INTO questions(id, kp_id, code, speech) VALUES
 (42, 10, 'calc', '{"text":"二加五等于几"}'),
 (43, 10, 'story', '{"text":"一共有几个"}'),
 (99, 11, 'glyph_sense', '{"text":"人"}');
`).Error)
	store := &questionStore{}
	speech := &questionSpeech{}
	return math.NewService(gdb, store, nil, questionVoices{}, speech), gdb, store, speech
}

func TestMathQuestionSpeechObjectKey(t *testing.T) {
	require.Equal(t, "math/questions/42.mp3", storage.MathQuestionSpeechObjectKey(42))
}

func TestGenerateQuestionSpeechUsesQuestionSpeechJSON(t *testing.T) {
	svc, gdb, store, speech := setupMathQuestionSpeechService(t)
	dto, err := svc.RegenerateQuestionSpeech(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, "二加五等于几", speech.LastText)
	require.Equal(t, "math/questions/42.mp3", store.LastKey)
	require.Equal(t, "math/questions/42.mp3", dto.MediaURL)

	var mediaURL string
	require.NoError(t, gdb.Raw(`SELECT media_url FROM questions WHERE id = 42`).Scan(&mediaURL).Error)
	require.Equal(t, dto.MediaURL, mediaURL)
}

func TestGenerateQuestionSpeechRejectsNonMathQuestion(t *testing.T) {
	svc, _, _, _ := setupMathQuestionSpeechService(t)
	_, err := svc.RegenerateQuestionSpeech(context.Background(), 99)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestQuestionSpeechStatusCountsOnlyStableQuestionKeys(t *testing.T) {
	svc, gdb, _, _ := setupMathQuestionSpeechService(t)
	require.NoError(t, gdb.Exec(`UPDATE questions SET media_url = 'math/questions/42.mp3' WHERE id = 42`).Error)
	require.NoError(t, gdb.Exec(`UPDATE questions SET media_url = 'http://localhost:19091/legacy.mp3' WHERE id = 43`).Error)

	status, err := svc.QuestionSpeechStatus("add10")
	require.NoError(t, err)
	require.Equal(t, math.QuestionSpeechStatus{ModuleCode: "add10", Total: 2, Ready: 1, Missing: 1}, status)
}

func TestBatchGenerateQuestionSpeechRegeneratesMissingAndLegacyKeys(t *testing.T) {
	svc, gdb, _, _ := setupMathQuestionSpeechService(t)
	require.NoError(t, gdb.Exec(`UPDATE questions SET media_url = 'http://localhost:19091/legacy.mp3' WHERE id = 43`).Error)

	result, err := svc.BatchGenerateQuestionSpeech(context.Background(), "add10")
	require.NoError(t, err)
	require.Equal(t, 2, result.Generated)
	require.Zero(t, result.Skipped)
	require.Zero(t, result.Failed)

	status, err := svc.QuestionSpeechStatus("add10")
	require.NoError(t, err)
	require.Equal(t, 2, status.Ready)
	require.Zero(t, status.Missing)
}
