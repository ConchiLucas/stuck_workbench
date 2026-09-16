package practice_test

import (
	"context"
	"testing"

	"github.com/conchi/study-learning/mastery"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/practice"
)

func practiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE children(id INTEGER PRIMARY KEY, name TEXT, grade TEXT DEFAULT '', avatar_url TEXT DEFAULT '', flowers INTEGER DEFAULT 0, created_at DATETIME)`,
		`CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, code TEXT, title TEXT, difficulty INTEGER, order_no INTEGER)`,
		`CREATE TABLE questions(id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, media_url TEXT)`,
		`CREATE TABLE study_plans(id INTEGER PRIMARY KEY, child_id INTEGER, plan_date TEXT, seq_no INTEGER, subject_code TEXT, plan_kind TEXT, module_code TEXT, stage_code TEXT, status TEXT, target_count INTEGER, done_count INTEGER DEFAULT 0, correct_count INTEGER DEFAULT 0, stars INTEGER DEFAULT 0, duration_sec INTEGER DEFAULT 0, created_at DATETIME, started_at DATETIME, completed_at DATETIME)`,
		`CREATE TABLE plan_items(id INTEGER PRIMARY KEY, plan_id INTEGER, seq INTEGER, kp_id INTEGER, question_id INTEGER, bucket TEXT, status TEXT, tries INTEGER DEFAULT 0, cost_ms INTEGER DEFAULT 0, picks TEXT DEFAULT '', option_order TEXT DEFAULT '', question_stem TEXT DEFAULT '', question_options TEXT DEFAULT '', question_answer TEXT DEFAULT '', question_visual TEXT DEFAULT '', question_speech TEXT DEFAULT '', question_snapshot TEXT DEFAULT '{}', explanation TEXT DEFAULT '', content_snapshot_version INTEGER DEFAULT 1, answered_at DATETIME)`,
		`CREATE TABLE attempts(id INTEGER PRIMARY KEY AUTOINCREMENT, child_id INTEGER, kp_id INTEGER, question_id INTEGER, is_correct BOOLEAN, cost_ms INTEGER, source TEXT, client_id TEXT, created_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_attempts_idem ON attempts(child_id, client_id)`,
		`CREATE TABLE mastery_states(child_id INTEGER, kp_id INTEGER, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, best_streak INTEGER, ease REAL, interval_days INTEGER, due_at DATETIME, first_seen_at DATETIME, mastered_at DATETIME, updated_at DATETIME, PRIMARY KEY(child_id,kp_id))`,
		`CREATE TABLE mastery_skills(child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, attempts INTEGER, correct INTEGER, streak INTEGER, best_streak INTEGER, ease REAL, interval_days INTEGER, due_at DATETIME, first_seen_at DATETIME, mastered_at DATETIME, updated_at DATETIME, PRIMARY KEY(child_id,kp_id,skill_code))`,
		`CREATE TABLE daily_stats(child_id INTEGER, stat_date TEXT, practice_sec INTEGER DEFAULT 0, attempts INTEGER DEFAULT 0, correct INTEGER DEFAULT 0, newly_mastered INTEGER DEFAULT 0, review_done INTEGER DEFAULT 0, checked_in BOOLEAN DEFAULT FALSE, PRIMARY KEY(child_id,stat_date))`,
		`CREATE TABLE flower_ledger(id INTEGER PRIMARY KEY AUTOINCREMENT, child_id INTEGER, delta INTEGER, reason TEXT, ref_type TEXT, ref_id INTEGER, created_at DATETIME)`,
		`INSERT INTO children(id,name,created_at) VALUES(1,'安安',CURRENT_TIMESTAMP)`,
		`INSERT INTO subjects VALUES(1,'math')`,
		`INSERT INTO modules VALUES(10,1,'add10',1)`,
		`INSERT INTO knowledge_points VALUES(100,10,'2p5','2+5',1,1)`,
		`INSERT INTO questions VALUES(42,100,'calc','math/questions/42.mp3')`,
		`INSERT INTO study_plans VALUES(7,1,'2026-09-01',1,'math','daily','','','doing',1,0,0,0,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,NULL)`,
		`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,bucket,status,question_snapshot) VALUES(9,7,1,100,42,'new','pending','{"questionId":42,"code":"calc","stem":"2 + 5 = ?","options":[{"label":"5"},{"label":"6"},{"label":"7"},{"label":"8"}],"answerIndex":2,"visual":{"kind":"equation","a":2,"b":5,"operator":"+"},"audioObjectKey":"math/questions/42.mp3"}')`,
	} {
		require.NoError(t, db.Exec(statement).Error)
	}
	return db
}

func TestAnswerSupportsTwoTriesIdempotencyAndExactReward(t *testing.T) {
	db := practiceDB(t)
	service := practice.NewService(db, mastery.DefaultConfig())

	first, err := service.Answer(context.Background(), 1, 7, 9, practice.AnswerInput{ClientID: "try-1", OptionIndex: 0, CostMs: 1200})
	require.NoError(t, err)
	require.False(t, first.Correct)
	require.True(t, first.CanRetry)
	require.Equal(t, 1, first.Tries)

	replayed, err := service.Answer(context.Background(), 1, 7, 9, practice.AnswerInput{ClientID: "try-1", OptionIndex: 0, CostMs: 1200})
	require.NoError(t, err)
	require.Equal(t, first, replayed)

	second, err := service.Answer(context.Background(), 1, 7, 9, practice.AnswerInput{ClientID: "try-2", OptionIndex: 2, CostMs: 800})
	require.NoError(t, err)
	require.True(t, second.Correct)
	require.False(t, second.CanRetry)
	require.Equal(t, 2, second.Tries)
	require.Equal(t, "correct", second.Status)

	var attempts, dailyAttempts int64
	require.NoError(t, db.Table("attempts").Count(&attempts).Error)
	require.NoError(t, db.Table("daily_stats").Select("attempts").Where("child_id = 1").Scan(&dailyAttempts).Error)
	require.Equal(t, int64(2), attempts)
	require.Equal(t, int64(2), dailyAttempts)

	finished, err := service.Finish(context.Background(), 1, 7)
	require.NoError(t, err)
	require.Equal(t, 3, finished.Stars)
	require.Equal(t, 2, finished.DurationSec)
	_, err = service.Finish(context.Background(), 1, 7)
	require.NoError(t, err)
	var childFlowers, ledgers int64
	require.NoError(t, db.Table("children").Select("flowers").Where("id = 1").Scan(&childFlowers).Error)
	require.NoError(t, db.Table("flower_ledger").Where("reason = 'plan_done' AND ref_id = 7").Count(&ledgers).Error)
	require.Equal(t, int64(4), childFlowers)
	require.Equal(t, int64(1), ledgers)
}

func TestAnswerRollsBackLearningWhenItemUpdateFails(t *testing.T) {
	db := practiceDB(t)
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_plan_item_update BEFORE UPDATE ON plan_items BEGIN SELECT RAISE(ABORT, 'forced failure'); END`).Error)
	service := practice.NewService(db, mastery.DefaultConfig())

	_, err := service.Answer(context.Background(), 1, 7, 9, practice.AnswerInput{ClientID: "rollback", OptionIndex: 2, CostMs: 1000})
	require.Error(t, err)
	for _, table := range []string{"attempts", "mastery_states", "mastery_skills", "daily_stats"} {
		var count int64
		require.NoError(t, db.Table(table).Count(&count).Error)
		require.Zero(t, count, table)
	}
}
