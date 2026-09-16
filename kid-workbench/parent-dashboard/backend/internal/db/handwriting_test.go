package db_test

import (
	"github.com/conchi/study-workbench/internal/db"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWritingReceiptMigration(t *testing.T) {
	d, e := db.OpenMemory()
	require.NoError(t, e)
	require.NoError(t, db.Migrate(d))
	require.True(t, d.Migrator().HasTable("literacy_writing_template_cache"))
	for _, col := range []string{"response_kind", "answer_payload_json", "evaluation_json", "evaluator_version"} {
		require.True(t, d.Migrator().HasColumn("question_attempt_receipts", col), col)
	}
	require.NoError(t, d.Exec(`INSERT INTO question_attempt_receipts(attempt_id,child_id,client_id,plan_id,plan_item_id,question_version_id,kp_id,skill_code,question_type,is_correct,cost_ms,response_json,response_kind) VALUES(1,1,'writing',1,1,1,1,'write_char','write_char',0,0,'{}','handwriting')`).Error)
}
