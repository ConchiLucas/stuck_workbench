package catalog_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/english-server/internal/catalog"
)

func TestRepositoryOnlyReturnsEnglish(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:english_catalog?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, code TEXT, title TEXT, payload TEXT, difficulty INTEGER, order_no INTEGER)`,
		`CREATE TABLE english_assets (kp_id INTEGER PRIMARY KEY, glyph_image_url TEXT, sense_image_url TEXT, speech_audio_url TEXT)`,
		`INSERT INTO subjects VALUES (1, 'english', '英语', 1), (2, 'pinyin', '拼音', 2)`,
		`INSERT INTO modules VALUES (10, 1, 'animals', '动物', 1), (20, 2, 'initials', '声母', 1)`,
		`INSERT INTO knowledge_points VALUES (100, 10, 'e1001', 'cat', '{"meaningZh":"猫","phonetic":"/kæt/"}', 1, 1), (200, 20, 'sm001', 'b', '{}', 1, 1)`,
		`INSERT INTO english_assets VALUES (100, '/glyph', '/sense', '/speech'), (200, '/bad', '/bad', '/bad')`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}

	repository := catalog.NewRepository(database)
	modules, err := repository.ListModules(context.Background())
	require.NoError(t, err)
	require.Equal(t, []catalog.Module{{Code: "animals", Name: "动物", OrderNo: 1, WordCount: 1}}, modules)

	words, err := repository.ListWords(context.Background(), "animals")
	require.NoError(t, err)
	require.Len(t, words, 1)
	require.Equal(t, "cat", words[0].Word)
	require.Equal(t, "猫", words[0].MeaningZh)
	require.Equal(t, "/kæt/", words[0].Phonetic)
	require.True(t, words[0].HasGlyph && words[0].HasSense && words[0].HasSpeech)

	_, err = repository.GetWord(context.Background(), 200)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
