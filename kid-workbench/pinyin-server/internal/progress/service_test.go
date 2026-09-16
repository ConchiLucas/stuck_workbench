package progress_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/pinyin-server/internal/progress"
)

func TestServiceReturnsOnlyPinyinWithOrderedSkills(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:progress?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE children (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, order_no INTEGER)`,
		`CREATE TABLE mastery_skills (child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, due_at DATETIME)`,
		`CREATE TABLE pinyin_syllable_links(kp_id INTEGER,enabled BOOLEAN)`,
		`INSERT INTO pinyin_syllable_links VALUES(300,TRUE),(301,FALSE)`,
		`INSERT INTO children VALUES (1, '安安')`,
		`INSERT INTO subjects VALUES (1, 'pinyin'), (2, 'literacy')`,
		`INSERT INTO modules VALUES (10, 1, 'initials', '声母', 1), (20, 2, 'basic', '基础', 1), (30,1,'syllables','音节拼读',3)`,
		`INSERT INTO knowledge_points VALUES (100, 10, 'b', 1), (200, 20, '一', 1),(300,30,'bā',1),(301,30,'bá',2)`,
		`INSERT INTO mastery_skills VALUES (1, 100, 'listen', 'mastered', 2, 2, 2, CURRENT_TIMESTAMP), (1, 100, 'inword', 'mastered', 2, 2, 2, NULL), (1, 200, 'glyph_sense', 'mastered', 2, 2, 2, NULL)`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}

	items, err := progress.NewService(database).Get(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, int64(100), items[0].KpID)
	require.Equal(t, "learning", items[0].Status)
	require.Equal(t, []string{"listen", "inword", "shape"}, []string{items[0].Skills[0].Code, items[0].Skills[1].Code, items[0].Skills[2].Code})
	require.Len(t, items[1].Skills, 1)
	require.Equal(t, "blend", items[1].Skills[0].Code)
	require.Equal(t, "not_started", items[1].Status)
}
