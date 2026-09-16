package db_test

import (
	"github.com/conchi/study-workbench/internal/db"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func TestQuestionLinksMigrationPreservesLegacyItem(t *testing.T) {
	g, err := db.OpenMemory()
	require.NoError(t, err)
	for _, sql := range []string{
		`CREATE TABLE study_plans(id INTEGER PRIMARY KEY,child_id INTEGER)`,
		`CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY)`, `CREATE TABLE questions(id INTEGER PRIMARY KEY)`,
		`CREATE TABLE plan_items(id INTEGER PRIMARY KEY,plan_id INTEGER,seq INTEGER,kp_id INTEGER,question_id INTEGER NOT NULL,bucket TEXT,status TEXT,tries INTEGER,cost_ms INTEGER,answered_at DATETIME,picks TEXT,option_order TEXT,question_stem TEXT,question_options TEXT,question_answer TEXT,question_visual TEXT,question_speech TEXT,explanation TEXT,content_snapshot_version INTEGER,question_snapshot TEXT)`,
		`INSERT INTO study_plans VALUES(1,1)`, `INSERT INTO knowledge_points VALUES(8)`, `INSERT INTO questions VALUES(9)`,
		`INSERT INTO plan_items VALUES(7,1,1,8,9,'new','correct',2,1000,NULL,'1,0','0,1','原始题干','[{"label":"甲"},{"label":"乙"}]','{"index":0}','{}','{}','原解释',1,'{"type":"choice"}')`,
	} {
		require.NoError(t, g.Exec(sql).Error)
	}
	body, err := os.ReadFile("migrations/sqlite/013_question_task_links.sql")
	require.NoError(t, err)
	for _, sql := range strings.Split(string(body), ";") {
		if strings.TrimSpace(sql) != "" {
			require.NoError(t, g.Exec(sql).Error)
		}
	}
	var row struct {
		ID, QuestionID                       int64
		QuestionStem, QuestionOptions, Picks string
	}
	require.NoError(t, g.Table("plan_items").Where("id=7").Take(&row).Error)
	require.EqualValues(t, 9, row.QuestionID)
	require.Equal(t, "原始题干", row.QuestionStem)
	require.Equal(t, "1,0", row.Picks)
	require.NoError(t, g.Exec(`INSERT INTO plan_items(plan_id,seq,kp_id,question_id,question_version_id,bucket) VALUES(1,2,8,NULL,99,'new')`).Error)
	require.Error(t, g.Exec(`INSERT INTO plan_items(plan_id,seq,kp_id,question_id,bucket) VALUES(1,3,8,NULL,'new')`).Error)
}
