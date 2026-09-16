package plan_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/plan"
)

func newPlanDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
CREATE TABLE children(id INTEGER PRIMARY KEY, name TEXT, flowers INTEGER DEFAULT 0);
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, difficulty INTEGER, order_no INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, media_url TEXT);
CREATE TABLE mastery_states(child_id INTEGER, kp_id INTEGER, status TEXT, due_at DATETIME);
CREATE TABLE mastery_skills(child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, due_at DATETIME);
CREATE TABLE attempts(id INTEGER PRIMARY KEY, child_id INTEGER, kp_id INTEGER, is_correct INTEGER, created_at DATETIME);
CREATE TABLE study_plans(
 id INTEGER PRIMARY KEY AUTOINCREMENT, child_id INTEGER, plan_date TEXT, seq_no INTEGER,
 subject_code TEXT, plan_kind TEXT, module_code TEXT, stage_code TEXT, status TEXT,
 target_count INTEGER, done_count INTEGER DEFAULT 0, correct_count INTEGER DEFAULT 0,
 stars INTEGER DEFAULT 0, duration_sec INTEGER DEFAULT 0, created_at DATETIME,
 started_at DATETIME, completed_at DATETIME, UNIQUE(child_id, plan_date, seq_no));
CREATE TABLE plan_items(
 id INTEGER PRIMARY KEY AUTOINCREMENT, plan_id INTEGER, seq INTEGER, kp_id INTEGER, question_id INTEGER,
 bucket TEXT, status TEXT, tries INTEGER DEFAULT 0, cost_ms INTEGER DEFAULT 0, picks TEXT DEFAULT '',
 option_order TEXT DEFAULT '', question_stem TEXT DEFAULT '', question_options TEXT DEFAULT '',
 question_answer TEXT DEFAULT '', question_visual TEXT DEFAULT '', question_speech TEXT DEFAULT '',
 question_snapshot TEXT DEFAULT '{}', explanation TEXT DEFAULT '', content_snapshot_version INTEGER DEFAULT 0,
 answered_at DATETIME, UNIQUE(plan_id, seq));
INSERT INTO children(id,name) VALUES(1,'安安');
INSERT INTO subjects VALUES(1,'math'),(2,'literacy');
INSERT INTO modules VALUES(10,1,'add10','20以内加法',1),(11,1,'sub10','20以内减法',2),(12,1,'shape','认识图形',3),(20,2,'add10','识字同名组',1);
`).Error)
	for i := 1; i <= 4; i++ {
		insertArithmeticQuestion(t, db, int64(100+i), 10, "add10", i, 1, int64(1000+i), "calc")
	}
	for i := 1; i <= 4; i++ {
		insertArithmeticQuestion(t, db, int64(200+i), 11, "sub10", 5+i, i, int64(2000+i), "story")
	}
	insertShapeQuestion(t, db, 301, 3001, "find", "圆形", `{}`)
	insertShapeQuestion(t, db, 302, 3002, "name", "正方形", `{"kind":"shape","text":"square"}`)
	return db
}

func insertArithmeticQuestion(t *testing.T, db *gorm.DB, kpID int64, moduleID int, moduleCode string, a, b int, questionID int64, code string) {
	t.Helper()
	kind, emoji, operator := "add", "🍎", "+"
	if moduleCode == "sub10" {
		kind, emoji, operator = "sub", "🍓", "-"
	}
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points VALUES(?,?,?,?,?,?)`, kpID, moduleID, fmt.Sprintf("%d%s%d", a, operator, b), fmt.Sprintf(`{"kind":"%s","a":%d,"b":%d}`, kind, a, b), 1, kpID).Error)
	visual := fmt.Sprintf(`{"kind":"%s","a":%d,"b":%d,"emoji":"%s"}`, kind, a, b, emoji)
	require.NoError(t, db.Exec(`INSERT INTO questions VALUES(?,?,?,?,?,?,?,?)`, questionID, kpID, code, "题目", `[{"label":"1"},{"label":"2"},{"label":"3"},{"label":"4"}]`, `{"index":1}`, visual, fmt.Sprintf("math/questions/%d.mp3", questionID)).Error)
}

func insertShapeQuestion(t *testing.T, db *gorm.DB, kpID, questionID int64, code, title, visual string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points VALUES(?,?,?,?,?,?)`, kpID, 12, title, `{}`, 1, kpID).Error)
	options := `[{"label":"圆形"},{"label":"正方形"},{"label":"三角形"},{"label":"五角星"}]`
	if code == "find" {
		options = `[{"shape":"circle"},{"shape":"square"},{"shape":"triangle"},{"shape":"star"}]`
	}
	require.NoError(t, db.Exec(`INSERT INTO questions VALUES(?,?,?,?,?,?,?,?)`, questionID, kpID, code, "题目", options, `{"index":0}`, visual, fmt.Sprintf("math/questions/%d.mp3", questionID)).Error)
}

func TestCreateDailyPlanIsStableAndUsesModuleQuotas(t *testing.T) {
	db := newPlanDB(t)
	svc := plan.NewService(db)
	first, err := svc.Create(context.Background(), 1, plan.CreateInput{Kind: "daily"})
	require.NoError(t, err)
	second, err := svc.Create(context.Background(), 1, plan.CreateInput{Kind: "daily"})
	require.NoError(t, err)
	require.Equal(t, first.Plan.ID, second.Plan.ID)
	require.Len(t, first.Items, 10)

	counts := map[string]int{}
	for _, item := range first.Items {
		if item.Question.Code == "find" || item.Question.Code == "name" {
			counts["shape"]++
		} else if item.Question.Visual.Kind == "equation" && item.Question.Visual.Operator == "+" || item.Question.Visual.Kind == "add" {
			counts["add10"]++
		} else {
			counts["sub10"]++
		}
	}
	require.Equal(t, map[string]int{"add10": 4, "sub10": 4, "shape": 2}, counts)
}

func TestPlanReloadUsesImmutableSnapshot(t *testing.T) {
	db := newPlanDB(t)
	svc := plan.NewService(db)
	detail, err := svc.Create(context.Background(), 1, plan.CreateInput{Kind: "module", ModuleCode: "add10", StageCode: "within5"})
	require.NoError(t, err)
	require.NotEmpty(t, detail.Items)
	wantStem := detail.Items[0].Question.Stem
	require.NoError(t, db.Exec(`UPDATE questions SET stem='被后台修改', media_url='math/questions/999.mp3'`).Error)

	reloaded, err := svc.Get(context.Background(), 1, detail.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, wantStem, reloaded.Items[0].Question.Stem)
	require.Contains(t, reloaded.Items[0].Question.AudioURL, fmt.Sprintf("/plans/%d/items/", detail.Plan.ID))
}

func TestModuleAndReviewPlansStayInScope(t *testing.T) {
	db := newPlanDB(t)
	svc := plan.NewService(db)
	module, err := svc.Create(context.Background(), 1, plan.CreateInput{Kind: "module", ModuleCode: "sub10", StageCode: "within10"})
	require.NoError(t, err)
	require.Len(t, module.Items, 4)
	for _, item := range module.Items {
		require.Equal(t, "sub", item.Question.Visual.Kind)
	}

	now := time.Now().Add(-time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO mastery_states(child_id,kp_id,status,due_at) VALUES(1,101,'mastered',?)`, now).Error)
	review, err := svc.Create(context.Background(), 1, plan.CreateInput{Kind: "review"})
	require.NoError(t, err)
	require.Len(t, review.Items, 1)
	require.Equal(t, int64(101), review.Items[0].KpID)
}

func TestReviewPlanIgnoresOldWrongAttempts(t *testing.T) {
	db := newPlanDB(t)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,is_correct,created_at) VALUES(1,1,101,FALSE,?)`, time.Now().AddDate(0, 0, -8)).Error)
	_, err := plan.NewService(db).Create(context.Background(), 1, plan.CreateInput{Kind: "review"})
	require.ErrorIs(t, err, plan.ErrNoQuestions)
}
