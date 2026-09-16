package quiz_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-science/internal/quiz"
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
CREATE TABLE science_assets(kp_id INTEGER PRIMARY KEY, review_status TEXT);
INSERT INTO subjects VALUES (1,'science'),(2,'literacy');
INSERT INTO modules VALUES (1,1,'animal','动物',1),(2,2,'hanzi','汉字',1);
INSERT INTO knowledge_points VALUES
 (10,1,'鸭子','{}',1),
 (11,1,'太阳','{}',2),
 (12,1,'冰','{}',3),
 (13,1,'金鱼','{}',4),
 (99,2,'山','{}',1);
INSERT INTO questions VALUES
 (101,10,'recognize','哪种动物的脚掌适合在水里游泳？','[{"label":"猫"},{"label":"鸭子"},{"label":"兔子"}]','{"index":1}','{"kind":"emoji","emoji":"🦆"}'),
 (102,11,'recognize','植物长大主要靠哪一种光？','[{"label":"灯光"},{"label":"太阳"},{"label":"萤火虫"}]','{"index":1}','{"kind":"emoji","emoji":"🌻"}'),
 (103,12,'recognize','冰变成水，是因为什么？','[{"label":"变热了"},{"label":"变冷了"},{"label":"变重了"}]','{"index":0}','{"kind":"emoji","emoji":"🧊"}'),
 (104,13,'recognize','哪种动物用鳃在水里呼吸？','[{"label":"猫"},{"label":"金鱼"},{"label":"兔子"}]','{"index":1}','{"kind":"emoji","emoji":"🐠"}'),
 (999,99,'recognize','错科','[{"label":"1"},{"label":"2"}]','{"index":0}','{}');
INSERT INTO science_assets VALUES (10,'draft');
`).Error)
	return quiz.NewService(db)
}

func TestGenerateChoiceUsesRecognizeWithoutPublishedAssets(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "choice", nil)
	require.NoError(t, err)
	require.Equal(t, "choice", question.Type)
	require.GreaterOrEqual(t, len(question.Options), 2)
	require.Contains(t, []int64{10, 11, 12, 13}, question.TargetID)
	require.NotEqual(t, int64(99), question.TargetID)
	require.NotEmpty(t, question.Stem)
	require.GreaterOrEqual(t, question.AnswerIndex, 0)
	require.Less(t, question.AnswerIndex, len(question.Options))
}

func TestGenerateHonorsExcludeTargetIds(t *testing.T) {
	service := setup(t)
	first, err := service.Generate(context.Background(), "choice", nil)
	require.NoError(t, err)
	second, err := service.Generate(context.Background(), "choice", []int64{first.TargetID})
	require.NoError(t, err)
	require.NotEqual(t, first.TargetID, second.TargetID)
}

func TestGenerateChoiceUsesStableAnswerID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:id-answer?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT);
INSERT INTO subjects VALUES (1,'science');
INSERT INTO modules VALUES (1,1,'observe');
INSERT INTO knowledge_points VALUES (10,1,'鸭子','{}',1);
INSERT INTO questions VALUES (101,10,'choice','哪种动物的脚掌适合在水里游泳？','[{"id":"cat","label":"猫"},{"id":"duck","label":"鸭子"},{"id":"rabbit","label":"兔子"}]','{"id":"duck"}','{"kind":"icon","key":"duck"}');
`).Error)
	question, err := quiz.NewService(db).Generate(context.Background(), "choice", nil)
	require.NoError(t, err)
	require.Equal(t, 1, question.AnswerIndex)
	require.Equal(t, "duck", question.Visual.Text)
}

func TestGenerateRejectsUnknownType(t *testing.T) {
	_, err := setup(t).Generate(context.Background(), "match", nil)
	require.ErrorIs(t, err, quiz.ErrInvalidType)
}

func TestGenerateRejectsWhenAllTargetsExcluded(t *testing.T) {
	_, err := setup(t).Generate(context.Background(), "choice", []int64{10, 11, 12, 13})
	require.ErrorIs(t, err, quiz.ErrNoMaterial)
}
