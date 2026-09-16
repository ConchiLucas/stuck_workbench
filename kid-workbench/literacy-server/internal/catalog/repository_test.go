package catalog_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/literacy-server/internal/catalog"
)

func TestRepositoryOnlyReturnsLiteracy(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:catalog?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, code TEXT, title TEXT, difficulty INTEGER, order_no INTEGER)`,
		`CREATE TABLE literacy_assets (kp_id INTEGER PRIMARY KEY, char_text TEXT, module_code TEXT, module_name TEXT, module_order INTEGER, kp_order INTEGER, glyph_image_url TEXT, sense_image_url TEXT, speech_audio_url TEXT)`,
		`INSERT INTO subjects VALUES (1, 'literacy', '识字', 1), (2, 'pinyin', '拼音', 2)`,
		`INSERT INTO modules VALUES (10, 1, 'g1', '第1组', 1), (20, 2, 'shengmu', '声母', 1)`,
		`INSERT INTO knowledge_points VALUES (100, 10, 'l1-yi', '一', 1, 1), (200, 20, 'pinyin-b', 'b', 1, 1)`,
		`INSERT INTO literacy_assets VALUES (100, '一', 'g1', '第1组', 1, 1, '/glyph', '/sense', '/speech'), (200, 'b', 'shengmu', '声母', 1, 1, '/x', '/y', '/z')`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}

	repository := catalog.NewRepository(database)
	modules, err := repository.ListModules(context.Background())
	require.NoError(t, err)
	require.Len(t, modules, 1)
	require.Equal(t, "g1", modules[0].Code)

	items, err := repository.ListItems(context.Background(), "g1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(100), items[0].KpID)
	require.Equal(t, "一", items[0].Character)
	require.True(t, items[0].HasGlyph)
	require.True(t, items[0].HasSense)
	require.True(t, items[0].HasSpeech)
}
