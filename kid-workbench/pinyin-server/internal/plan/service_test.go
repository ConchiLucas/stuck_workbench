package plan_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/pinyin-server/internal/plan"
)

func TestCreatePlanIsStableAndPinyinScoped(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:plan?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE children (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, order_no INTEGER)`,
		`CREATE TABLE questions (id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, speech TEXT, difficulty INTEGER)`,
		`CREATE TABLE study_plans (id INTEGER PRIMARY KEY AUTOINCREMENT, child_id INTEGER, plan_date TEXT, seq_no INTEGER, subject_code TEXT, status TEXT, target_count INTEGER, done_count INTEGER DEFAULT 0, correct_count INTEGER DEFAULT 0, stars INTEGER DEFAULT 0, duration_sec INTEGER DEFAULT 0, created_at DATETIME, started_at DATETIME, completed_at DATETIME)`,
		`CREATE TABLE plan_items (id INTEGER PRIMARY KEY AUTOINCREMENT, plan_id INTEGER, seq INTEGER, kp_id INTEGER, question_id INTEGER, bucket TEXT, status TEXT, tries INTEGER DEFAULT 0, cost_ms INTEGER DEFAULT 0, picks TEXT DEFAULT '', option_order TEXT DEFAULT '', question_stem TEXT DEFAULT '', question_options TEXT DEFAULT '', question_answer TEXT DEFAULT '', question_visual TEXT DEFAULT '', question_speech TEXT DEFAULT '', question_snapshot TEXT DEFAULT '{}', explanation TEXT DEFAULT '', content_snapshot_version INTEGER DEFAULT 0, answered_at DATETIME)`,
		`INSERT INTO children VALUES (1, '安安')`,
		`INSERT INTO subjects VALUES (1, 'pinyin'), (2, 'english')`,
		`INSERT INTO modules VALUES (10, 1, 'initials', 1), (20, 2, 'words', 1)`,
		`INSERT INTO knowledge_points VALUES (100, 10, 'b', 1), (200, 20, 'book', 1)`,
		`INSERT INTO questions VALUES (1000, 100, 'inword', 'choice', '请选择', '[{"label":"b"},{"label":"p"},{"label":"m"},{"label":"f"}]', '{"index":0}', '{}', '{}', 1), (1001, 100, 'listen', 'choice', '听一听', '[{"label":"b"},{"label":"p"},{"label":"m"},{"label":"f"}]', '{"index":0}', '{}', '{}', 1), (2000, 200, 'listen', 'choice', 'wrong subject', '[]', '{"index":0}', '{}', '{}', 1)`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}

	service := plan.NewService(database)
	created, err := service.Create(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, "pinyin", created.Plan.SubjectCode)
	require.Len(t, created.Items, 2)
	require.NoError(t, database.Table("questions").Where("id = ?", created.Items[0].Question.ID).
		Updates(map[string]any{"stem": "后台已修改", "options": `[{"label":"x"}]`}).Error)

	reloaded, err := service.Get(context.Background(), 1, created.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, created.Items[0].Question.ID, reloaded.Items[0].Question.ID)
	require.Equal(t, created.Items[0].Question.Stem, reloaded.Items[0].Question.Stem)
	require.Equal(t, created.Items[0].Question.Options, reloaded.Items[0].Question.Options)
	require.NotEmpty(t, reloaded.Items[0].OptionOrder)
}
