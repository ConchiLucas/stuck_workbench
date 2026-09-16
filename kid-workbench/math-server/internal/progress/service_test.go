package progress_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/progress"
)

func TestMathProgressUsesModuleSkillsAndExcludesOtherSubjects(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`
CREATE TABLE children(id INTEGER PRIMARY KEY, name TEXT);
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER);
CREATE TABLE mastery_skills(child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, due_at DATETIME);
INSERT INTO children VALUES(1,'安安');
INSERT INTO subjects VALUES(1,'math'),(2,'pinyin');
INSERT INTO modules VALUES(10,1,'add10','20以内加法',1),(11,1,'shape','认识图形',2),(20,2,'add10','拼音同名组',1);
INSERT INTO knowledge_points VALUES
 (100,10,'2+3','{"kind":"add","a":2,"b":3}',1),
 (101,11,'圆形','{}',1),
 (200,20,'b','{}',1);
INSERT INTO mastery_skills VALUES
 (1,100,'calc','mastered',2,2,2,NULL),
 (1,101,'find','mastered',2,2,2,NULL),
 (1,101,'name','mastered',2,2,2,NULL),
 (1,200,'calc','mastered',2,2,2,NULL);
`).Error)

	result, err := progress.NewService(database).Get(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Modules, 2)
	add := result.Modules[0].Stages[0].Items[0]
	require.Equal(t, "learning", add.Status)
	require.Equal(t, []string{"calc", "story"}, []string{add.Skills[0].Code, add.Skills[1].Code})
	shape := result.Modules[1].Stages[0].Items[0]
	require.Equal(t, "mastered", shape.Status)
	require.Equal(t, []string{"find", "name"}, []string{shape.Skills[0].Code, shape.Skills[1].Code})
}

func TestMathProgressRejectsUnknownChild(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`CREATE TABLE children(id INTEGER PRIMARY KEY)`).Error)
	_, err = progress.NewService(database).Get(context.Background(), 99)
	require.ErrorIs(t, err, progress.ErrChildNotFound)
}
