package progress_test

import (
	"context"
	"github.com/conchi/english-server/internal/progress"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestServiceReturnsOnlyEnglishWithOrderedSkills(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:english_progress?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range []string{
		`CREATE TABLE children (id INTEGER PRIMARY KEY, name TEXT)`, `CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER)`,
		`CREATE TABLE mastery_skills (child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, due_at DATETIME)`,
		`INSERT INTO children VALUES (1,'安安')`, `INSERT INTO subjects VALUES (1,'english'),(2,'pinyin')`,
		`INSERT INTO modules VALUES (10,1,'animals',1),(20,2,'initials',1)`,
		`INSERT INTO knowledge_points VALUES (100,10,'cat','{"meaningZh":"猫"}',1),(200,20,'b','{}',1)`,
		`INSERT INTO mastery_skills VALUES (1,100,'listen','mastered',2,2,2,NULL),(1,200,'listen','mastered',2,2,2,NULL)`,
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	rows, err := progress.NewService(db).Get(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "cat", rows[0].Word)
	require.Equal(t, "猫", rows[0].MeaningZh)
	require.Equal(t, []string{"listen", "picture"}, []string{rows[0].Skills[0].Code, rows[0].Skills[1].Code})
	require.Equal(t, "learning", rows[0].Status)
	require.Equal(t, "not_started", rows[0].Skills[1].Status)
}
