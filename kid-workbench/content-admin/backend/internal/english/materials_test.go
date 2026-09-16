package english

import (
	"context"
	"strings"
	"testing"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/stretchr/testify/require"
)

func TestCuratedSentencesAndPassagesAreLegal(t *testing.T) {
	illegal := []string{"This is a nine", "This is a white", "This is a mouth", "red nine", "red white", "red mouth", "red nose"}
	for _, spec := range curatedSentences {
		require.Equal(t, spec.Text, strings.Join(spec.Tokens, " "))
		require.Contains(t, spec.Text, spec.Target)
		for _, bad := range illegal {
			require.NotContains(t, spec.Text, bad)
		}
	}
	for _, spec := range curatedPassages {
		require.Contains(t, strings.ToLower(spec.Passage), strings.ToLower(spec.Answer))
		require.GreaterOrEqual(t, len(spec.Options), 4)
		for _, bad := range illegal {
			require.NotContains(t, spec.Passage, bad)
		}
	}
}

func TestEnsureQuestionMaterialsDoesNotTemplateIllegalWords(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Exec(`
		CREATE TABLE IF NOT EXISTS subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE IF NOT EXISTS modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT);
		CREATE TABLE IF NOT EXISTS knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT);
		INSERT INTO subjects(id, code) VALUES (1,'english');
		INSERT INTO modules(id, subject_id, code, name) VALUES (1,1,'animals','动物');
	`).Error)
	words := []struct {
		id   int64
		text string
	}{
		{1, "apple"}, {2, "dog"}, {3, "cat"}, {4, "bird"}, {5, "banana"}, {6, "pencil"},
		{7, "ball"}, {8, "bag"}, {9, "book"}, {10, "cake"}, {11, "milk"}, {12, "sun"},
		{13, "fish"}, {14, "hat"}, {15, "nine"}, {16, "white"}, {17, "mouth"}, {18, "nose"},
	}
	for _, word := range words {
		require.NoError(t, gdb.Exec(`INSERT INTO knowledge_points(id, module_id, title, payload) VALUES (?,?,?,?)`, word.id, 1, word.text, `{"meaningZh":"`+word.text+`"}`).Error)
		require.NoError(t, gdb.Exec(`INSERT INTO english_assets(kp_id, word_text, module_code, module_name, needs_sense_image, sense_image_url, speech_audio_url) VALUES (?,?,?,?,1,?,?)`, word.id, word.text, "animals", "动物", "/s", "/a").Error)
	}
	require.NoError(t, EnsureQuestionMaterials(gdb))
	require.NoError(t, EnsureQuestionMaterials(gdb))
	var count int64
	require.NoError(t, gdb.Model(&Sentence{}).Count(&count).Error)
	require.Equal(t, int64(len(curatedSentences)), count)
	var texts []string
	require.NoError(t, gdb.Model(&Sentence{}).Pluck("text", &texts).Error)
	require.NotEmpty(t, texts)
	joined := strings.Join(texts, "\n")
	require.Contains(t, joined, "This is an apple")
	require.NotContains(t, joined, "This is a nine")
	require.NotContains(t, joined, "This is a white")
	require.NotContains(t, joined, "This is a mouth")
	var passages []string
	require.NoError(t, gdb.Model(&Passage{}).Pluck("passage", &passages).Error)
	require.NotEmpty(t, passages)
	require.GreaterOrEqual(t, len(passages), 3)
	blob := strings.Join(passages, "\n")
	require.NotContains(t, blob, "red mouth")
	require.NotContains(t, blob, "red nose")
	require.NotContains(t, blob, "red nine")
	svc := NewService(gdb, nil, nil, nil, nil, nil, nil)
	_, err = svc.SentenceSpeechMP3(context.Background(), "this-is-an-apple")
	require.Error(t, err)
	require.Contains(t, err.Error(), "尚未准备")
}
