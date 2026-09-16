package quiz_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/logic-server/internal/quiz"
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
INSERT INTO subjects VALUES (1,'logic'),(2,'math');
INSERT INTO modules VALUES
 (1,1,'pattern','找规律',1),
 (2,1,'classify','分类',2),
 (3,1,'order','排序',3),
 (4,1,'shape_reason','图形推理',4),
 (5,1,'diff','找不同',5),
 (7,1,'compare','比较',6),
 (6,2,'add10','加法',1);
INSERT INTO knowledge_points VALUES
 (10,1,'红蓝交替','{}',1),
 (11,1,'大小交替','{}',2),
 (12,1,'水果两个一换','{}',3),
 (13,1,'一二一二','{}',4),
 (20,2,'哪个不是水果','{}',1),
 (21,2,'哪个不会飞','{}',2),
 (22,2,'哪个不是动物','{}',3),
 (23,2,'哪个不能吃','{}',4),
 (30,3,'从小到大','{}',1),
 (31,3,'从大到小','{}',2),
 (32,3,'一天顺序','{}',3),
 (33,3,'生长顺序','{}',4),
 (40,4,'圆圆方方','{}',1),
 (41,4,'三角递增','{}',2),
 (42,4,'颜色形状','{}',3),
 (43,4,'大小圆','{}',4),
 (50,5,'三个水果一个梨','{}',1),
 (51,5,'三只动物一只狗','{}',2),
 (52,5,'三个笑一个哭','{}',3),
 (53,5,'三个蓝一个红','{}',4),
 (60,7,'哪个更高','{}',1),
 (61,7,'哪个更大','{}',2),
 (62,7,'哪个更快','{}',3),
 (63,7,'哪个更重','{}',4),
 (99,6,'1+1','{}',1);
INSERT INTO questions VALUES
 (101,10,'pick1','下一个是哪个？','[{"emoji":"⬛"},{"emoji":"🟡"},{"emoji":"🔴"},{"emoji":"🟢"}]','{"index":2}','{"kind":"seq","items":["🔴","🔵","🔴","🔵"]}'),
 (102,10,'pick2','下一个是哪个？','[{"emoji":"🟡"},{"emoji":"🔴"},{"emoji":"⬛"},{"emoji":"🟢"}]','{"index":1}','{"kind":"seq","items":["🔴","🔵","🔴","🔵"]}'),
 (103,11,'pick1','下一个是哪个？','[{"emoji":"🔵"},{"emoji":"🔺"},{"emoji":"⬛"},{"emoji":"⬜"}]','{"index":2}','{"kind":"seq","items":["⬛","⬜","⬛","⬜"]}'),
 (104,12,'pick1','下一个是哪个？','[{"emoji":"🍎"},{"emoji":"🍌"},{"emoji":"🍊"},{"emoji":"🍇"}]','{"index":1}','{"kind":"seq","items":["🍎","🍎","🍌","🍌"]}'),
 (105,13,'pick1','下一个是哪个？','[{"label":"4️⃣"},{"label":"1️⃣"},{"label":"3️⃣"},{"label":"0️⃣"}]','{"index":1}','{"kind":"seq","items":["1️⃣","2️⃣","1️⃣","2️⃣"]}'),
 (201,20,'pick1','哪个不是水果？','[{"emoji":"🍎"},{"emoji":"🍊"},{"emoji":"🍌"},{"emoji":"🚗"}]','{"index":3}','{}'),
 (202,21,'pick1','哪个不会飞？','[{"emoji":"🐦"},{"emoji":"✈️"},{"emoji":"🐕"},{"emoji":"🦋"}]','{"index":2}','{}'),
 (203,22,'pick1','哪个不是动物？','[{"emoji":"🐰"},{"emoji":"🐶"},{"emoji":"🐱"},{"emoji":"🌳"}]','{"index":3}','{}'),
 (204,23,'pick1','哪个不能吃？','[{"emoji":"🍞"},{"emoji":"🍎"},{"emoji":"👟"},{"emoji":"🧀"}]','{"index":2}','{}'),
 (301,30,'pick1','从小到大怎么排？','[{"label":"2 → 1 → 3"},{"label":"1 → 2 → 3"},{"label":"3 → 2 → 1"},{"label":"1 → 3 → 2"}]','{"index":1}','{"kind":"seq","items":["1️⃣","3️⃣","2️⃣"]}'),
 (302,31,'pick1','从大到小怎么排？','[{"label":"5 → 3 → 1"},{"label":"1 → 3 → 5"},{"label":"3 → 5 → 1"},{"label":"5 → 1 → 3"}]','{"index":0}','{"kind":"seq","items":["5️⃣","1️⃣","3️⃣"]}'),
 (303,32,'pick1','一天的正确顺序是？','[{"label":"早上 → 中午 → 晚上"},{"label":"晚上 → 早上 → 中午"},{"label":"中午 → 晚上 → 早上"},{"label":"早上 → 晚上 → 中午"}]','{"index":0}','{"kind":"seq","items":["🌅","☀️","🌙"]}'),
 (304,33,'pick1','小树长大的顺序是？','[{"label":"种子 → 小苗 → 大树"},{"label":"大树 → 小苗 → 种子"},{"label":"小苗 → 种子 → 大树"},{"label":"种子 → 大树 → 小苗"}]','{"index":0}','{"kind":"seq","items":["🌱","🌿","🌳"]}'),
 (401,40,'pick1','下一个是哪个？','[{"emoji":"🔵"},{"emoji":"🔺"},{"emoji":"⭐"},{"emoji":"⬛"}]','{"index":0}','{"kind":"seq","items":["🔵","⬛","🔵","⬛"]}'),
 (402,41,'pick1','三角形越来越多，下一个？','[{"label":"🔺🔺🔺🔺"},{"label":"🔺"},{"label":"⬛"},{"label":"🔵"}]','{"index":0}','{"kind":"seq","items":["🔺","🔺🔺","🔺🔺🔺"]}'),
 (403,42,'pick1','下一个是哪个？','[{"emoji":"🔴"},{"emoji":"🔵"},{"emoji":"⬛"},{"emoji":"⭐"}]','{"index":0}','{"kind":"seq","items":["🔴","🔺","🔴","🔺"]}'),
 (404,43,'pick1','下一个是哪个？','[{"emoji":"⚪"},{"emoji":"⚫"},{"emoji":"⬛"},{"emoji":"🔺"}]','{"index":0}','{"kind":"seq","items":["⚪","⚫","⚪","⚫"]}'),
 (501,50,'pick1','哪个和其他不一样？','[{"emoji":"🍌"},{"emoji":"🍐"},{"emoji":"🍎"},{"emoji":"🍊"}]','{"index":1}','{}'),
 (502,51,'pick1','哪个和其他不一样？','[{"emoji":"🐰"},{"emoji":"🐻"},{"emoji":"🐶"},{"emoji":"🐱"}]','{"index":2}','{}'),
 (503,52,'pick1','哪个表情不一样？','[{"emoji":"😊"},{"emoji":"😢"},{"emoji":"😄"},{"emoji":"😁"}]','{"index":1}','{}'),
 (504,53,'pick1','哪个颜色不一样？','[{"emoji":"🔴"},{"emoji":"🔵"},{"emoji":"🔹"},{"emoji":"🟦"}]','{"index":0}','{}'),
 (601,60,'pick1','哪个更高？','[{"emoji":"🐭"},{"emoji":"🦒"},{"emoji":"🐜"},{"emoji":"🐣"}]','{"index":1}','{}'),
 (602,61,'pick1','哪个更大？','[{"emoji":"🐱"},{"emoji":"🐰"},{"emoji":"🐘"},{"emoji":"🐦"}]','{"index":2}','{}'),
 (603,62,'pick1','哪个更快？','[{"emoji":"🐢"},{"emoji":"🐌"},{"emoji":"🚀"},{"emoji":"🚶"}]','{"index":2}','{}'),
 (604,63,'pick1','哪个更重？','[{"emoji":"🪶"},{"emoji":"🎈"},{"emoji":"🐘"},{"emoji":"🍃"}]','{"index":2}','{}'),
 (999,99,'pick1','错科','[{"label":"1"},{"label":"2"},{"label":"3"},{"label":"4"}]','{"index":0}','{}');
`).Error)
	return quiz.NewService(db)
}

func TestGeneratePatternUsesPick1AndSeq(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "pattern", nil)
	require.NoError(t, err)
	require.Equal(t, "pattern", question.Type)
	require.Equal(t, "seq", question.Visual.Kind)
	require.NotEmpty(t, question.Visual.Items)
	require.Len(t, question.Options, 4)
	require.Contains(t, []int64{10, 11, 12, 13}, question.TargetID)
	require.NotEqual(t, int64(99), question.TargetID)
	require.NotEmpty(t, question.Options[question.AnswerIndex].Display())
}

func TestGenerateClassifyHasNoSequence(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "classify", nil)
	require.NoError(t, err)
	require.Equal(t, "classify", question.Type)
	require.Empty(t, question.Visual.Items)
	require.Contains(t, []int64{20, 21, 22, 23}, question.TargetID)
}

func TestGenerateOrderUsesTextOptions(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "order", nil)
	require.NoError(t, err)
	require.Equal(t, "order", question.Type)
	require.Contains(t, question.Options[question.AnswerIndex].Label, "→")
	require.NotEmpty(t, question.Visual.Items)
}

func TestGenerateShapeReasonAndDiff(t *testing.T) {
	service := setup(t)
	shape, err := service.Generate(context.Background(), "shape_reason", nil)
	require.NoError(t, err)
	require.Equal(t, "shape_reason", shape.Type)
	require.Contains(t, []int64{40, 41, 42, 43}, shape.TargetID)

	diff, err := service.Generate(context.Background(), "diff", nil)
	require.NoError(t, err)
	require.Equal(t, "diff", diff.Type)
	require.Contains(t, []int64{50, 51, 52, 53}, diff.TargetID)

	compare, err := service.Generate(context.Background(), "compare", nil)
	require.NoError(t, err)
	require.Equal(t, "compare", compare.Type)
	require.Contains(t, []int64{60, 61, 62, 63}, compare.TargetID)
	require.Empty(t, compare.Visual.Items)
}

func TestGenerateHonorsExcludeTargetIds(t *testing.T) {
	service := setup(t)
	first, err := service.Generate(context.Background(), "pattern", nil)
	require.NoError(t, err)
	second, err := service.Generate(context.Background(), "pattern", []int64{first.TargetID})
	require.NoError(t, err)
	require.NotEqual(t, first.TargetID, second.TargetID)
}

func TestGenerateRejectsUnknownType(t *testing.T) {
	_, err := setup(t).Generate(context.Background(), "blend", nil)
	require.ErrorIs(t, err, quiz.ErrInvalidType)
}

func TestGenerateRejectsWhenAllTargetsExcluded(t *testing.T) {
	_, err := setup(t).Generate(context.Background(), "pattern", []int64{10, 11, 12, 13})
	require.ErrorIs(t, err, quiz.ErrNoMaterial)
}
