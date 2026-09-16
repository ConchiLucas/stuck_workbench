package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-workbench/internal/db"
)

func TestMigrateCreatesTables(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))

	for _, table := range []string{
		"users", "children", "parent_child", "subjects", "modules",
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
