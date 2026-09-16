package quiz_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/quiz"
)

func setup(t *testing.T) *quiz.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT);
INSERT INTO subjects VALUES (1,'math'),(2,'literacy');
INSERT INTO modules VALUES (1,1,'add10','20以内加法',1),(2,1,'sub10','20以内减法',2),(3,1,'shape','认识图形',3),(4,2,'hanzi','汉字',1);
INSERT INTO knowledge_points VALUES
 (10,1,'1+1','{"kind":"add","a":1,"b":1,"emoji":"🍎"}',1),
 (11,1,'1+2','{"kind":"add","a":1,"b":2,"emoji":"🍎"}',2),
 (12,1,'3+5','{"kind":"add","a":3,"b":5,"emoji":"🍎"}',3),
 (13,1,'4+4','{"kind":"add","a":4,"b":4,"emoji":"🍎"}',4),
 (20,2,'9-4','{"kind":"sub","a":9,"b":4,"emoji":"🍓"}',1),
 (21,2,'8-3','{"kind":"sub","a":8,"b":3,"emoji":"🍓"}',2),
 (30,3,'圆形','{}',1),
 (31,3,'三角形','{}',2),
 (32,3,'正方形','{}',3),
 (33,3,'长方形','{}',4),
 (99,4,'山','{}',1);
INSERT INTO questions VALUES
 (101,10,'calc','1 + 1 = ?','[{"label":"1"},{"label":"2"},{"label":"3"},{"label":"4"}]','{"index":1}','{"kind":"add","a":1,"b":1,"emoji":"🍎"}'),
 (102,11,'calc','1 + 2 = ?','[{"label":"2"},{"label":"3"},{"label":"4"},{"label":"5"}]','{"index":1}','{"kind":"add","a":1,"b":2,"emoji":"🍎"}'),
 (103,12,'calc','3 + 5 = ?','[{"label":"7"},{"label":"8"},{"label":"9"},{"label":"6"}]','{"index":1}','{"kind":"add","a":3,"b":5,"emoji":"🍎"}'),
 (104,13,'calc','4 + 4 = ?','[{"label":"6"},{"label":"7"},{"label":"8"},{"label":"9"}]','{"index":2}','{"kind":"add","a":4,"b":4,"emoji":"🍎"}'),
 (201,10,'story','一共有几个？','[{"label":"1"},{"label":"2"},{"label":"3"},{"label":"4"}]','{"index":1}','{"kind":"add","a":1,"b":1,"emoji":"🍎"}'),
 (202,11,'story','一共有几个？','[{"label":"2"},{"label":"3"},{"label":"4"},{"label":"5"}]','{"index":1}','{"kind":"add","a":1,"b":2,"emoji":"🍎"}'),
 (301,30,'name','这是什么形状？','[{"label":"圆形"},{"label":"正方形"},{"label":"菱形"},{"label":"三角形"}]','{"index":0}','{"kind":"shape","text":"circle"}'),
 (302,31,'name','这是什么形状？','[{"label":"圆形"},{"label":"三角形"},{"label":"正方形"},{"label":"长方形"}]','{"index":1}','{"kind":"shape","text":"triangle"}'),
 (999,99,'calc','错科','[{"label":"1"},{"label":"2"},{"label":"3"},{"label":"4"}]','{"index":0}','{}');
`).Error)
	return quiz.NewService(db)
}

func TestGenerateEquationUsesStoredCalcQuestions(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "equation", nil)
	require.NoError(t, err)
	require.Equal(t, "equation", question.Type)
	require.Len(t, question.Options, 4)
	require.GreaterOrEqual(t, question.AnswerIndex, 0)
	require.Less(t, question.AnswerIndex, 4)
	require.NotEmpty(t, question.Options[question.AnswerIndex].Label)
	require.Contains(t, []int64{10, 11, 12, 13}, question.TargetID)
	require.NotEqual(t, int64(99), question.TargetID)
	require.Contains(t, question.Stem, "=")
}

func TestGenerateStoryUsesStoredStoryQuestions(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "story", nil)
	require.NoError(t, err)
	require.Equal(t, "story", question.Type)
	require.Equal(t, "一共有几个？", question.Stem)
	require.Contains(t, []int64{10, 11}, question.TargetID)
}

func TestGenerateShapeUsesStoredNameQuestions(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "shape", nil)
	require.NoError(t, err)
	require.Equal(t, "shape", question.Type)
	require.Equal(t, "这是什么形状？", question.Stem)
	require.Equal(t, "shape", question.Visual.Kind)
	require.Contains(t, []int64{30, 31}, question.TargetID)
}

func TestGenerateMissingAndJudgeFromKnowledgePoints(t *testing.T) {
	service := setup(t)
	missing, err := service.Generate(context.Background(), "missing", nil)
	require.NoError(t, err)
	require.Equal(t, "missing", missing.Type)
	require.Contains(t, missing.Stem, "□")
	require.Len(t, missing.Options, 4)
	require.Contains(t, []int64{10, 11, 12, 13, 20, 21}, missing.TargetID)

	judge, err := service.Generate(context.Background(), "judge", nil)
	require.NoError(t, err)
	require.Equal(t, "judge", judge.Type)
	require.Len(t, judge.Options, 2)
	require.Equal(t, "对", judge.Options[0].Label)
	require.Equal(t, "错", judge.Options[1].Label)
}

func TestGenerateHonorsExcludeTargetIds(t *testing.T) {
	service := setup(t)
	first, err := service.Generate(context.Background(), "equation", nil)
	require.NoError(t, err)
	second, err := service.Generate(context.Background(), "equation", []int64{first.TargetID})
	require.NoError(t, err)
	require.NotEqual(t, first.TargetID, second.TargetID)
}

func TestGenerateRejectsUnknownType(t *testing.T) {
	_, err := setup(t).Generate(context.Background(), "blend", nil)
	require.ErrorIs(t, err, quiz.ErrInvalidType)
}
