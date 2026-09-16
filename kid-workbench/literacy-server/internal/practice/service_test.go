package practice_test

import (
	"context"
	"testing"

	"github.com/conchi/study-learning/mastery"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/literacy-server/internal/plan"
	"github.com/conchi/literacy-server/internal/practice"
)

func TestAnswerSupportsTwoTriesAndIdempotency(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:practice?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	for _, statement := range practiceSchema() {
		require.NoError(t, database.Exec(statement).Error)
	}
	plans := plan.NewService(database)
	detail, err := plans.Create(context.Background(), 1, 2)
	require.NoError(t, err)
	item := detail.Items[0]
	// Publishing a new question version must not change an already-created plan.
	require.NoError(t, database.Table("questions").Where("id = ?", item.Question.ID).
		Updates(map[string]any{"answer": `{"index":1}`, "options": `[{"label":"x"},{"label":"b"}]`}).Error)
	order, err := plan.ParseOrder(item.OptionOrder)
	require.NoError(t, err)
	correctDisplay := 0
	for display, original := range order {
		if original == 0 {
			correctDisplay = display
		}
	}
	wrongDisplay := (correctDisplay + 1) % len(order)

	service := practice.NewService(database, mastery.Config{BaseMasterStreak: 2, MinAccuracy: .8, ShakyMinAttempts: 3, ShakyAccuracy: .6, EaseMin: 1.3, EaseMax: 2.8, EaseUp: .1, EaseDown: .2, MaxIntervalDays: 60})
	first, err := service.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-1", OptionIndex: wrongDisplay, CostMs: 1200})
	require.NoError(t, err)
	require.False(t, first.Correct)
	require.True(t, first.CanRetry)
	var startedPlan plan.StudyPlan
	require.NoError(t, database.First(&startedPlan, detail.Plan.ID).Error)
	require.Equal(t, "doing", startedPlan.Status)
	require.NotNil(t, startedPlan.StartedAt)

	second, err := service.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-2", OptionIndex: correctDisplay, CostMs: 800})
	require.NoError(t, err)
	require.True(t, second.Correct)
	require.False(t, second.CanRetry)
	require.Equal(t, "correct", second.Status)

	_, err = service.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-2", OptionIndex: correctDisplay, CostMs: 800})
	require.NoError(t, err)
	_, err = service.Answer(context.Background(), 1, detail.Plan.ID, detail.Items[1].ID, practice.AnswerInput{ClientID: "try-2", OptionIndex: 0, CostMs: 800})
	require.ErrorIs(t, err, practice.ErrIdempotencyConflict)
	var attempts, dailyAttempts int64
	require.NoError(t, database.Table("attempts").Count(&attempts).Error)
	require.NoError(t, database.Table("daily_stats").Select("attempts").Where("child_id = 1").Scan(&dailyAttempts).Error)
	require.Equal(t, int64(2), attempts)
	require.Equal(t, int64(2), dailyAttempts)
}

func practiceSchema() []string {
	return []string{
		`CREATE TABLE children (id INTEGER PRIMARY KEY, name TEXT, grade TEXT DEFAULT '', avatar_url TEXT DEFAULT '', flowers INTEGER DEFAULT 0, created_at DATETIME)`,
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, code TEXT, title TEXT, difficulty INTEGER, order_no INTEGER)`,
		`CREATE TABLE questions (id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, speech TEXT, difficulty INTEGER)`,
		`CREATE TABLE study_plans (id INTEGER PRIMARY KEY AUTOINCREMENT, child_id INTEGER, plan_date TEXT, seq_no INTEGER, subject_code TEXT, status TEXT, target_count INTEGER, done_count INTEGER DEFAULT 0, correct_count INTEGER DEFAULT 0, stars INTEGER DEFAULT 0, duration_sec INTEGER DEFAULT 0, created_at DATETIME, started_at DATETIME, completed_at DATETIME)`,
		`CREATE TABLE plan_items (id INTEGER PRIMARY KEY AUTOINCREMENT, plan_id INTEGER, seq INTEGER, kp_id INTEGER, question_id INTEGER, bucket TEXT, status TEXT, tries INTEGER DEFAULT 0, cost_ms INTEGER DEFAULT 0, picks TEXT DEFAULT '', option_order TEXT DEFAULT '', question_stem TEXT DEFAULT '', question_options TEXT DEFAULT '', question_answer TEXT DEFAULT '', question_visual TEXT DEFAULT '', question_speech TEXT DEFAULT '', question_snapshot TEXT DEFAULT '{}', explanation TEXT DEFAULT '', content_snapshot_version INTEGER DEFAULT 0, answered_at DATETIME)`,
		`CREATE TABLE attempts (id INTEGER PRIMARY KEY AUTOINCREMENT, child_id INTEGER, kp_id INTEGER, question_id INTEGER, is_correct BOOLEAN, cost_ms INTEGER, source TEXT, client_id TEXT, created_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_attempts_idem ON attempts(child_id, client_id)`,
		`CREATE TABLE mastery_states (child_id INTEGER, kp_id INTEGER, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, best_streak INTEGER, ease REAL, interval_days INTEGER, due_at DATETIME, first_seen_at DATETIME, mastered_at DATETIME, updated_at DATETIME, PRIMARY KEY(child_id,kp_id))`,
		`CREATE TABLE mastery_skills (child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, best_streak INTEGER, ease REAL, interval_days INTEGER, due_at DATETIME, first_seen_at DATETIME, mastered_at DATETIME, updated_at DATETIME, PRIMARY KEY(child_id,kp_id,skill_code))`,
		`CREATE TABLE daily_stats (child_id INTEGER, stat_date TEXT, practice_sec INTEGER DEFAULT 0, attempts INTEGER DEFAULT 0, correct INTEGER DEFAULT 0, newly_mastered INTEGER DEFAULT 0, review_done INTEGER DEFAULT 0, checked_in BOOLEAN DEFAULT FALSE, PRIMARY KEY(child_id,stat_date))`,
		`CREATE TABLE flower_ledger (id INTEGER PRIMARY KEY AUTOINCREMENT, child_id INTEGER, delta INTEGER, reason TEXT, ref_type TEXT, ref_id INTEGER, created_at DATETIME)`,
		`INSERT INTO children(id,name,created_at) VALUES (1, '安安', CURRENT_TIMESTAMP)`,
		`INSERT INTO subjects VALUES (1, 'literacy')`,
		`INSERT INTO modules VALUES (10, 1, 'g1', 1)`,
		`INSERT INTO knowledge_points VALUES (100, 10, 'l1-yi', '一', 1, 1)`,
		`INSERT INTO questions VALUES (1000, 100, 'glyph_sense', 'choice', '看字选义', '[{"label":"一"},{"label":"二"},{"label":"三"},{"label":"十"}]', '{"index":0}', '{}', '{}', 1), (1001, 100, 'write_char', 'write', '写一写这个字', '[]', '{"index":0}', '{}', '{}', 1)`,
	}
}

func TestLegacyWriteCharRequiresVersionedTask(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:practice-write?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	for _, statement := range practiceSchema() {
		require.NoError(t, database.Exec(statement).Error)
	}
	plans := plan.NewService(database)
	detail, err := plans.Create(context.Background(), 1, 2)
	require.NoError(t, err)
	var writeItem plan.Item
	for _, item := range detail.Items {
		if item.Question.Code == "write_char" {
			writeItem = item
		}
	}
	require.Equal(t, "write_char", writeItem.Question.Code)

	service := practice.NewService(database, mastery.Config{BaseMasterStreak: 2, MinAccuracy: .8, ShakyMinAttempts: 3, ShakyAccuracy: .6, EaseMin: 1.3, EaseMax: 2.8, EaseUp: .1, EaseDown: .2, MaxIntervalDays: 60})
	_, err = service.Answer(context.Background(), 1, detail.Plan.ID, writeItem.ID, practice.AnswerInput{ClientID: "write-1", OptionIndex: 0, CostMs: 4000})
	require.ErrorIs(t, err, practice.ErrLegacyWritingRequiresTask)
	var n int64
	require.NoError(t, database.Table("attempts").Count(&n).Error)
	require.Zero(t, n)
}
