package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/model"
)

func TestMigrateCreatesTables(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))

	for _, table := range []string{
		"kid_users", "children", "parent_child", "subjects", "modules",
		"knowledge_points", "questions", "attempts", "mastery_states",
		"study_sessions", "daily_stats", "daily_tasks", "rewards", "flower_ledger",
	} {
		require.True(t, gdb.Migrator().HasTable(table), "缺少表 %s", table)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, db.Migrate(gdb))
}

func TestMigrateKeepsEnglishAndChildAccountsSeparate(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO users(id, username) VALUES (1, 'english-account')`).Error)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Exec(`INSERT INTO kid_users(id, phone, nickname, password_hash, created_at) VALUES (1, '123', 'parent', '', CURRENT_TIMESTAMP)`).Error)
	var username string
	require.NoError(t, gdb.Raw(`SELECT username FROM users WHERE id=1`).Scan(&username).Error)
	require.Equal(t, "english-account", username)
	var nickname string
	require.NoError(t, gdb.Raw(`SELECT nickname FROM kid_users WHERE id=1`).Scan(&nickname).Error)
	require.Equal(t, "parent", nickname)
	var parent model.User
	require.NoError(t, gdb.First(&parent, 1).Error)
	require.Equal(t, "parent", parent.Nickname)
	var referencedTable string
	require.NoError(t, gdb.Raw(`SELECT "table" FROM pragma_foreign_key_list('parent_child') WHERE "from"='user_id'`).Scan(&referencedTable).Error)
	require.Equal(t, "kid_users", referencedTable)
}

func TestMigrateCreatesMathPlanColumns(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))

	for _, tc := range []struct{ table, column string }{
		{"plan_items", "question_snapshot"},
		{"study_plans", "plan_kind"},
		{"study_plans", "module_code"},
		{"study_plans", "stage_code"},
	} {
		require.True(t, gdb.Migrator().HasColumn(tc.table, tc.column), "%s.%s", tc.table, tc.column)
	}
}

func TestMigrateCreatesAttemptSelected(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.True(t, gdb.Migrator().HasColumn("attempts", "selected"))
	require.True(t, gdb.Migrator().HasColumn("attempts", "plan_item_id"))
}

func TestMigrateCreatesScienceIsolationColumns(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))

	for _, tc := range []struct{ table, column string }{
		{"science_assets", "glyph_object_key"},
		{"science_assets", "sense_object_key"},
		{"science_assets", "speech_object_key"},
		{"plan_items", "content_snapshot_version"},
		{"science_attempt_receipts", "plan_id"},
		{"science_attempt_receipts", "item_id"},
		{"poem_attempt_receipts", "plan_id"},
		{"poem_attempt_receipts", "item_id"},
	} {
		require.True(t, gdb.Migrator().HasColumn(tc.table, tc.column), "%s.%s", tc.table, tc.column)
	}
}
