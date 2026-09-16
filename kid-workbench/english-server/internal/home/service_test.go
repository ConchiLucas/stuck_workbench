package home_test

import (
	"context"
	"github.com/conchi/english-server/internal/home"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestHomeScopesPlansAndMasteryToEnglish(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:english_home?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range []string{
		`CREATE TABLE children (id INTEGER PRIMARY KEY,name TEXT,grade TEXT,avatar_url TEXT,flowers INTEGER)`,
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY,code TEXT)`, `CREATE TABLE modules (id INTEGER PRIMARY KEY,subject_id INTEGER,code TEXT,name TEXT,order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY,module_id INTEGER)`, `CREATE TABLE mastery_states (child_id INTEGER,kp_id INTEGER,status TEXT,due_at DATETIME)`,
		`CREATE TABLE study_plans (id INTEGER PRIMARY KEY,child_id INTEGER,plan_date TEXT,seq_no INTEGER,subject_code TEXT,status TEXT,target_count INTEGER,done_count INTEGER)`,
		`INSERT INTO children VALUES (1,'安安','大班','',7)`, `INSERT INTO subjects VALUES (1,'english'),(2,'pinyin')`,
		`INSERT INTO modules VALUES (10,1,'animals','动物',1),(20,2,'initials','声母',1)`, `INSERT INTO knowledge_points VALUES (100,10),(200,20)`,
		`INSERT INTO mastery_states VALUES (1,100,'review_due',CURRENT_TIMESTAMP),(1,200,'review_due',CURRENT_TIMESTAMP)`,
		`INSERT INTO study_plans VALUES (1,1,'2026-09-01',1,'english','todo',1,0),(2,1,'2026-09-01',2,'pinyin','todo',1,0)`,
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	result, err := home.NewService(db).Get(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.CurrentPlan.ID)
	require.Equal(t, 1, result.DueCount)
	require.Equal(t, 7, result.Child.Flowers)
	require.Len(t, result.Modules, 1)
	require.Equal(t, "animals", result.Modules[0].Code)
}
