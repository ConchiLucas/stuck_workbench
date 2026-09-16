package home_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/pinyin-server/internal/home"
)

func TestSummaryIsPinyinScoped(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:home?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE children (id INTEGER PRIMARY KEY, name TEXT, grade TEXT, avatar_url TEXT, flowers INTEGER)`,
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, order_no INTEGER)`,
		`CREATE TABLE pinyin_syllable_links(kp_id INTEGER,enabled BOOLEAN)`,
		`CREATE TABLE mastery_skills(child_id INTEGER,kp_id INTEGER,skill_code TEXT,status TEXT,attempts INTEGER,correct INTEGER,streak INTEGER,due_at DATETIME)`,
		`CREATE TABLE mastery_states (child_id INTEGER, kp_id INTEGER, status TEXT, due_at DATETIME)`,
		`CREATE TABLE study_plans (id INTEGER PRIMARY KEY, child_id INTEGER, plan_date TEXT, seq_no INTEGER, subject_code TEXT, status TEXT, target_count INTEGER, done_count INTEGER, correct_count INTEGER, stars INTEGER, duration_sec INTEGER, created_at DATETIME, started_at DATETIME, completed_at DATETIME)`,
		`INSERT INTO children VALUES (1, '安安', '一年级', '', 3)`,
		`INSERT INTO subjects VALUES (1, 'pinyin'), (2, 'literacy')`,
		`INSERT INTO modules VALUES (10, 1, 'initials', '声母', 1), (20, 2, 'basic', '汉字', 1)`,
		`INSERT INTO knowledge_points VALUES (100, 10, 'b', 1), (101, 10, 'p', 2), (200, 20, '一', 1)`,
		`INSERT INTO mastery_states VALUES (1, 100, 'mastered', CURRENT_TIMESTAMP), (1, 101, 'mastered', NULL), (1, 200, 'mastered', CURRENT_TIMESTAMP)`,
		`INSERT INTO mastery_skills VALUES(1,100,'listen','mastered',2,2,2,CURRENT_TIMESTAMP),(1,100,'inword','mastered',2,2,2,NULL),(1,100,'shape','mastered',2,2,2,NULL),(1,101,'listen','mastered',2,2,2,NULL),(1,101,'inword','mastered',2,2,2,NULL)`,
		`INSERT INTO study_plans VALUES (50, 1, '2026-09-01', 1, 'pinyin', 'doing', 2, 1, 1, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, NULL), (60, 1, '2026-09-01', 2, 'literacy', 'doing', 1, 0, 0, 0, 0, CURRENT_TIMESTAMP, NULL, NULL)`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}

	summary, err := home.NewService(database).Get(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, summary.CurrentPlan)
	require.Equal(t, int64(50), summary.CurrentPlan.ID)
	require.Equal(t, 1, summary.DueCount)
	require.Len(t, summary.Modules, 1)
	require.Equal(t, 1, summary.Modules[0].Mastered)
}
