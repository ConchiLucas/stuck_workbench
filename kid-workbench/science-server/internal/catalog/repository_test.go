package catalog_test

import (
	"context"
	"testing"

	"github.com/conchi/study-science/internal/catalog"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepositoryReturnsOnlyPublishedScience(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, difficulty INTEGER, order_no INTEGER);
CREATE TABLE science_assets(kp_id INTEGER PRIMARY KEY, summary TEXT, explanation TEXT, fun_fact TEXT, review_status TEXT, content_version INTEGER, sense_image_url TEXT, glyph_image_url TEXT, speech_audio_url TEXT, glyph_object_key TEXT, sense_object_key TEXT, speech_object_key TEXT);
CREATE TABLE questions(id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT);
INSERT INTO subjects VALUES(1,'science'),(2,'literacy');
INSERT INTO modules VALUES(1,1,'animal','动物',1),(2,2,'g1','识字',1);
INSERT INTO knowledge_points VALUES(1,1,'冬眠',1,1),(2,1,'迁徙',1,2),(3,2,'人',1,1);
INSERT INTO science_assets VALUES
  (1,'怎么过冬','节省能量','心跳变慢','published',2,'yes','yes','yes','science/glyphs/1-v2.png','science/senses/1-v2.png','science/speech/1-v2.mp3'),
  (2,'飞去南方','','','draft',1,'','','','','',''),
  (3,'人','','','published',1,'','','','','','');
INSERT INTO questions VALUES(1,1,'recognize');
`).Error)
	repo := catalog.NewRepository(gdb)

	modules, err := repo.ListModules(context.Background())
	require.NoError(t, err)
	require.Len(t, modules, 1)
	require.Equal(t, 1, modules[0].Total)

	items, err := repo.ListItems(context.Background(), "animal")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "冬眠", items[0].Title)
	require.Equal(t, "怎么过冬", items[0].Summary)
	require.True(t, items[0].HasPractice)

	_, err = repo.GetItem(context.Background(), 2)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
