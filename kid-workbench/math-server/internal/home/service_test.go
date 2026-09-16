package home_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/home"
)

func TestHomeIsSubjectIsolated(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	past := time.Now().Add(-time.Hour)
	require.NoError(t, db.Exec(`
CREATE TABLE children(id INTEGER PRIMARY KEY, name TEXT, flowers INTEGER);
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER);
CREATE TABLE mastery_states(child_id INTEGER, kp_id INTEGER, status TEXT, due_at DATETIME);
CREATE TABLE study_plans(id INTEGER PRIMARY KEY, child_id INTEGER, subject_code TEXT, plan_kind TEXT, status TEXT, target_count INTEGER, done_count INTEGER, correct_count INTEGER, created_at DATETIME);
INSERT INTO children VALUES(1,'安安',12);
INSERT INTO subjects VALUES(1,'math'),(2,'literacy');
INSERT INTO modules VALUES(10,1,'add10','20以内加法',1),(11,1,'sub10','20以内减法',2),(12,1,'shape','认识图形',3),(20,2,'basic','识字',1);
INSERT INTO knowledge_points VALUES(100,10),(101,11),(102,12),(200,20);
INSERT INTO mastery_states VALUES(1,100,'mastered',?),(1,200,'mastered',?);
INSERT INTO study_plans VALUES(7,1,'math','daily','doing',10,2,1,CURRENT_TIMESTAMP),(8,1,'literacy','daily','doing',8,1,1,CURRENT_TIMESTAMP);
`, past, past).Error)

	result, err := home.NewService(db).Get(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Child.ID)
	require.Equal(t, 12, result.Child.Flowers)
	require.NotNil(t, result.Plan)
	require.Equal(t, int64(7), result.Plan.ID)
	require.Equal(t, 1, result.DueCount)
	require.Len(t, result.Modules, 3)
	require.Equal(t, "add10", result.Modules[0].Code)
	require.Equal(t, 1, result.Modules[0].MasteredCount)
}

func TestHomeRejectsUnknownChild(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE children(id INTEGER PRIMARY KEY, name TEXT, flowers INTEGER)`).Error)
	_, err = home.NewService(db).Get(context.Background(), 99)
	require.ErrorIs(t, err, home.ErrChildNotFound)
}
