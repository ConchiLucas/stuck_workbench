package catalog_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/pinyin-server/internal/catalog"
)

func TestRepositoryOnlyReturnsPinyin(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:catalog?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, code TEXT, title TEXT, difficulty INTEGER, order_no INTEGER)`,
		`CREATE TABLE pinyin_assets (kp_id INTEGER PRIMARY KEY, letter TEXT, module_code TEXT, module_name TEXT, module_order INTEGER, kp_order INTEGER, solo_text TEXT, word_text TEXT, solo_speech_url TEXT, word_speech_url TEXT, glyph_image_url TEXT)`,
		`INSERT INTO subjects VALUES (1, 'pinyin', '拼音', 1), (2, 'literacy', '识字', 2)`,
		`INSERT INTO modules VALUES (10, 1, 'initials', '声母', 1), (20, 2, 'basic', '基础汉字', 1),(30,1,'syllables','音节拼读',3)`,
		`INSERT INTO knowledge_points VALUES (100, 10, 'pinyin-b', 'b', 1, 1), (200, 20, 'han-yi', '一', 1, 1),(300,30,'py-syllable-1','bā',1,1)`,
		`INSERT INTO pinyin_assets VALUES (100, 'b', 'initials', '声母', 1, 1, 'b', '广播', '/solo', '/word', '/glyph'), (200, '一', 'basic', '基础汉字', 1, 1, '一', '一个', '/x', '/y', '/z')`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}

	repository := catalog.NewRepository(database)
	modules, err := repository.ListModules(context.Background())
	require.NoError(t, err)
	require.Len(t, modules, 1)
	require.Equal(t, "initials", modules[0].Code)

	items, err := repository.ListItems(context.Background(), "initials")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(100), items[0].KpID)
	require.True(t, items[0].HasSoloSpeech)
	require.True(t, items[0].HasWordSpeech)
	require.True(t, items[0].HasGlyph)
	syllables, err := repository.ListItems(context.Background(), "syllables")
	require.NoError(t, err)
	require.Empty(t, syllables)
	_, err = repository.GetItem(context.Background(), 300)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
