package pinyin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/pinyin"
)

func setupPinyinDB(t *testing.T) *pinyin.Service {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'pinyin', '拼音', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'shengmu', '声母', 1),
 (2, 1, 'yunmu', '韵母', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (101, 1, 'b', 'b', '{}', 1, 1),
 (102, 1, 'p', 'p', '{}', 1, 2),
 (201, 2, 'a', 'a', '{}', 1, 1),
 (202, 2, 'eng', 'eng', '{}', 1, 2);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return pinyin.NewService(gdb, nil, nil, nil, nil)
}

func TestSyncAndListGroups(t *testing.T) {
	svc := setupPinyinDB(t)
	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 4, res.Total)
	require.Equal(t, 4, res.Upserted)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 4, list.Total)
	require.Len(t, list.Groups, 2)
	require.Equal(t, "shengmu", list.Groups[0].ModuleCode)
	require.Equal(t, "yunmu", list.Groups[1].ModuleCode)

	byLetter := map[string]pinyin.ItemDTO{}
	for _, g := range list.Groups {
		for _, it := range g.Items {
			byLetter[it.Letter] = it
		}
	}
	require.Equal(t, "波", byLetter["b"].SoloText)
	require.Equal(t, "包", byLetter["b"].WordText)
	require.Equal(t, []string{"包", "八", "不", "白", "北"}, byLetter["b"].WordExamples)
	require.Equal(t, "", byLetter["eng"].SoloText)
	require.Equal(t, "灯", byLetter["eng"].WordText)
	require.Equal(t, []string{"灯", "风", "正", "生", "朋"}, byLetter["eng"].WordExamples)
}

func TestBatchGenerateSpeechRequiresModule(t *testing.T) {
	svc := setupPinyinDB(t)
	_, err := svc.BatchGenerateSpeech(context.Background(), "")
	require.Error(t, err)
}

func TestBatchGenerateGlyphsRequiresModule(t *testing.T) {
	svc := setupPinyinDB(t)
	_, err := svc.BatchGenerateGlyphs(context.Background(), "")
	require.Error(t, err)
}

type memStore struct {
	files map[string][]byte
	keys  []string
}

func (s *memStore) PutBytes(_ context.Context, key string, data []byte, _ string) (string, error) {
	if s.files == nil {
		s.files = map[string][]byte{}
	}
	s.files[key] = append([]byte(nil), data...)
	s.keys = append(s.keys, key)
	return key, nil
}

func (s *memStore) GetBytes(_ context.Context, key string) ([]byte, error) {
	data, ok := s.files[key]
	if !ok {
		return nil, context.Canceled
	}
	return append([]byte(nil), data...), nil
}

type xiaomiVoices struct{}

func (xiaomiVoices) LoadVoiceModels(context.Context) (configclient.AIConfiguration, error) {
	return configclient.AIConfiguration{
		ActiveProviderID: "grok-tts",
		Providers: []configclient.AIProvider{
			{ID: "grok-tts", Type: "grok-tts", Enabled: true, BaseURL: "http://grok", APIKey: "k", Capabilities: []string{"AUDIO_TTS"}},
			{ID: "xiaomi-mimo-tts", Type: "mimo-tts", Enabled: true, BaseURL: "http://mimo", APIKey: "k", Capabilities: []string{"AUDIO_TTS"}},
		},
	}, nil
}

type recordingSpeech struct {
	providerIDs []string
	texts       []string
}

func (s *recordingSpeech) Synthesize(_ context.Context, provider configclient.AIProvider, text string) ([]byte, error) {
	s.providerIDs = append(s.providerIDs, provider.ID)
	s.texts = append(s.texts, text)
	return []byte("mp3-" + text), nil
}

func setupPinyinSpeechService(t *testing.T) (*pinyin.Service, *memStore, *recordingSpeech) {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'pinyin', '拼音', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'shengmu', '声母', 1);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (101, 1, 'b', 'b', '{}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	store := &memStore{}
	speech := &recordingSpeech{}
	svc := pinyin.NewService(gdb, store, xiaomiVoices{}, speech, nil)
	_, err = svc.Sync()
	require.NoError(t, err)
	return svc, store, speech
}

func setupPinyinYunmuService(t *testing.T) (*pinyin.Service, *memStore, *recordingSpeech) {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'pinyin', '拼音', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (2, 1, 'yunmu', '韵母', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (201, 2, 'a', 'a', '{}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	store := &memStore{}
	speech := &recordingSpeech{}
	svc := pinyin.NewService(gdb, store, xiaomiVoices{}, speech, nil)
	_, err = svc.Sync()
	require.NoError(t, err)
	return svc, store, speech
}

func TestSpeechMP3GeneratesXiaomiExampleAudioAndStoresMapping(t *testing.T) {
	svc, store, speech := setupPinyinSpeechService(t)
	mp3, err := svc.SpeechMP3(context.Background(), 101, "word-1")
	require.NoError(t, err)
	require.Equal(t, []byte("mp3-八"), mp3)
	require.Equal(t, []string{"xiaomi-mimo-tts"}, speech.providerIDs)
	require.Equal(t, []string{"八"}, speech.texts)
	require.Contains(t, store.keys, "pinyin/speech/101/word-1.mp3")

	list, err := svc.List("table")
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	item := list.Items[0]
	require.Contains(t, item.WordExampleSpeechURLs["八"], "/api/v1/pinyin/items/101/speech/word-1.mp3")
}
