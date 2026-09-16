package progress_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/literacy-server/internal/progress"
)

func TestServiceReturnsOnlyLiteracyWithOrderedSkills(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:progress?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE children (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, order_no INTEGER)`,
		`CREATE TABLE mastery_skills (child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, due_at DATETIME)`,
		`INSERT INTO children VALUES (1, '安安')`,
		`INSERT INTO subjects VALUES (1, 'literacy'), (2, 'pinyin')`,
		`INSERT INTO modules VALUES (10, 1, 'g1', 1), (20, 2, 'shengmu', 1)`,
		`INSERT INTO knowledge_points VALUES (100, 10, '一', 1), (200, 20, 'b', 1)`,
		`INSERT INTO mastery_skills VALUES (1, 100, 'glyph_sense', 'mastered', 2, 2, 2, CURRENT_TIMESTAMP), (1, 100, 'sense_char', 'mastered', 2, 2, 2, NULL), (1, 100, 'write_char', 'mastered', 2, 2, 2, NULL), (1, 200, 'listen', 'mastered', 2, 2, 2, NULL)`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}

	items, err := progress.NewService(database).Get(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(100), items[0].KpID)
	require.Equal(t, "一", items[0].Character)
	require.Equal(t, "review_due", items[0].Status)
	require.Equal(t, []string{"glyph_sense", "sense_char", "write_char"}, []string{
		items[0].Skills[0].Code, items[0].Skills[1].Code, items[0].Skills[2].Code,
	})
}
