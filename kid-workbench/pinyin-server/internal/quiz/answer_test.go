package quiz_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/conchi/pinyin-server/internal/quiz"
	"github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/conchi/study-learning/pinyincontract"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func formalService(t *testing.T) (*quiz.Service, *gorm.DB) {
	t.Helper()
	db := setupQuizDB(t)
	require.NoError(t, db.AutoMigrate(&model.Child{}, &model.KnowledgePoint{}, &model.Attempt{}, &model.MasteryState{}, &model.MasterySkill{}, &model.DailyStat{}, &model.FlowerLedger{}, &model.PinyinQuizInstance{}, &model.PinyinAnswerReceipt{}, &model.PinyinSyllableLink{}, &model.PinyinMasteryMilestone{}))
	for _, sql := range []string{
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY,code TEXT,name TEXT,order_no INTEGER)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER,code TEXT,name TEXT,order_no INTEGER)`,
		`CREATE UNIQUE INDEX module_code ON modules(subject_id,code)`,
		`CREATE UNIQUE INDEX kp_code ON knowledge_points(module_id,code)`,
		`CREATE UNIQUE INDEX attempts_idem ON attempts(child_id,client_id)`,
		`INSERT INTO subjects VALUES(1,'pinyin','拼音',1)`,
		`INSERT INTO modules VALUES(1,1,'shengmu','声母',1)`,
		`INSERT INTO children(id,name,flowers) VALUES(1,'测试',0),(2,'其他',0)`,
		`INSERT INTO knowledge_points(id,module_id,code,title,difficulty,order_no) VALUES(1,1,'b','b',1,1),(2,1,'p','p',1,2),(3,1,'m','m',1,3),(4,1,'f','f',1,4)`,
		`ALTER TABLE pinyin_syllable_assets ADD COLUMN difficulty INTEGER DEFAULT 1`,
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	conn, err := db.DB()
	require.NoError(t, err)
	conn.SetMaxOpenConns(1)
	return quiz.NewService(db), db
}
func TestFormalGenerationAndFrozenAnswer(t *testing.T) {
	service, db := formalService(t)
	ctx := context.Background()
	for _, kind := range []string{"listen", "inword", "shape", "blend"} {
		q, err := service.GenerateForChild(ctx, 1, kind, nil)
		require.NoError(t, err)
		data, err := json.Marshal(q)
		require.NoError(t, err)
		require.NotContains(t, string(data), "answerIndex")
		require.NotContains(t, string(data), "answerOptionId")
		require.Len(t, q.Options, 4)
		for _, option := range q.Options {
			require.NotEqual(t, strconv.FormatInt(q.TargetID, 10), option.ID)
		}
		if kind == "blend" {
			require.Empty(t, q.Visual.Syllable)
		}
		require.NotZero(t, q.KpID)
		if kind == "blend" {
			require.NotEqual(t, q.TargetID, q.KpID)
		}
		var n int64
		require.NoError(t, db.Model(&model.Attempt{}).Count(&n).Error)
		require.Zero(t, n)
	}
	q, err := service.GenerateForChild(ctx, 1, "shape", nil)
	require.NoError(t, err)
	var stored model.PinyinQuizInstance
	require.NoError(t, db.First(&stored, "id = ?", q.InstanceID).Error)
	require.NoError(t, db.Exec(`UPDATE pinyin_assets SET solo_text='修改', letter='修改'`).Error)
	input := pinyincontract.AnswerRequest{ClientID: "frozen", OptionID: stored.AnswerOptionID, CostMs: 1200}
	result, err := service.Answer(ctx, 1, q.InstanceID, input)
	require.NoError(t, err)
	require.True(t, result.Correct)
	require.Equal(t, "shape", result.Skill.Code)
	restored, err := service.GetInstance(ctx, 1, q.InstanceID)
	require.NoError(t, err)
	require.Equal(t, q, restored.GeneratedQuestion)
	require.Equal(t, &result, restored.AcceptedResult)
	require.NoError(t, db.Model(&model.PinyinQuizInstance{}).Where("id = ?", q.InstanceID).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	retry, err := service.Answer(ctx, 1, q.InstanceID, input)
	require.NoError(t, err)
	require.Equal(t, result, retry)
	input.OptionID = "different"
	_, err = service.Answer(ctx, 1, q.InstanceID, input)
	require.ErrorIs(t, err, quiz.ErrConflict)
	input.ClientID = "different"
	input.OptionID = stored.AnswerOptionID
	_, err = service.Answer(ctx, 1, q.InstanceID, input)
	require.ErrorIs(t, err, quiz.ErrConflict)
}
func TestFormalInvalidExpiredAndCrossChildNeverWrites(t *testing.T) {
	service, db := formalService(t)
	ctx := context.Background()
	q, err := service.GenerateForChild(ctx, 1, "listen", nil)
	require.NoError(t, err)
	input := pinyincontract.AnswerRequest{ClientID: "one", OptionID: "unknown"}
	_, err = service.Answer(ctx, 1, q.InstanceID, pinyincontract.AnswerRequest{ClientID: strings.Repeat("a", 65), OptionID: q.Options[0].ID})
	require.ErrorIs(t, err, quiz.ErrInvalidRequest)
	_, err = service.Answer(ctx, 1, q.InstanceID, input)
	require.ErrorIs(t, err, quiz.ErrInvalidRequest)
	input.OptionID = q.Options[0].ID
	input.CostMs = -1
	_, err = service.Answer(ctx, 1, q.InstanceID, input)
	require.ErrorIs(t, err, quiz.ErrInvalidRequest)
	input.CostMs = 3600001
	_, err = service.Answer(ctx, 1, q.InstanceID, input)
	require.ErrorIs(t, err, quiz.ErrInvalidRequest)
	input.CostMs = 0
	_, err = service.Answer(ctx, 2, q.InstanceID, input)
	require.ErrorIs(t, err, quiz.ErrNotFound)
	_, err = service.GetInstance(ctx, 2, q.InstanceID)
	require.ErrorIs(t, err, quiz.ErrNotFound)
	_, err = service.GenerateForChild(ctx, 999, "listen", nil)
	require.ErrorIs(t, err, quiz.ErrNotFound)
	require.NoError(t, db.Model(&model.PinyinQuizInstance{}).Where("id = ?", q.InstanceID).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	_, err = service.Answer(ctx, 1, q.InstanceID, input)
	require.ErrorIs(t, err, quiz.ErrExpired)
	_, err = service.GetInstance(ctx, 1, q.InstanceID)
	require.ErrorIs(t, err, quiz.ErrExpired)
	var n int64
	require.NoError(t, db.Model(&model.Attempt{}).Count(&n).Error)
	require.Zero(t, n)
}
func TestFormalConcurrentRetriesAndReceiptRollback(t *testing.T) {
	service, db := formalService(t)
	ctx := context.Background()
	q, err := service.GenerateForChild(ctx, 1, "listen", nil)
	require.NoError(t, err)
	input := pinyincontract.AnswerRequest{ClientID: "same", OptionID: q.Options[0].ID}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Answer(ctx, 1, q.InstanceID, input); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var n int64
	require.NoError(t, db.Model(&model.Attempt{}).Count(&n).Error)
	require.EqualValues(t, 1, n)
	next, err := service.GenerateForChild(ctx, 1, "shape", nil)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_receipt BEFORE INSERT ON pinyin_answer_receipts BEGIN SELECT RAISE(ABORT,'receipt failed'); END`).Error)
	_, err = service.Answer(ctx, 1, next.InstanceID, pinyincontract.AnswerRequest{ClientID: "rollback", OptionID: next.Options[0].ID})
	require.Error(t, err)
	require.NoError(t, db.Model(&model.Attempt{}).Count(&n).Error)
	require.EqualValues(t, 1, n)
	require.NoError(t, db.Model(&model.MasterySkill{}).Where("skill_code = 'shape'").Count(&n).Error)
	require.Zero(t, n)
	var attempts int
	require.NoError(t, db.Table("daily_stats").Select("SUM(attempts)").Scan(&attempts).Error)
	require.Equal(t, 1, attempts)
	require.NoError(t, db.Model(&model.FlowerLedger{}).Count(&n).Error)
	require.Zero(t, n)
}

func TestFormalDisabledContentRetainsIssuedSnapshotAndBlendIdentity(t *testing.T) {
	service, db := formalService(t)
	ctx := context.Background()
	q, err := service.GenerateForChild(ctx, 1, "blend", nil)
	require.NoError(t, err)
	before, err := service.GetInstance(ctx, 1, q.InstanceID)
	require.NoError(t, err)
	require.Nil(t, before.AcceptedResult)
	var stored model.PinyinQuizInstance
	require.NoError(t, db.First(&stored, "id = ?", q.InstanceID).Error)
	require.NoError(t, db.Exec(`UPDATE pinyin_syllable_assets SET enabled=FALSE, speech_text='已改文案'`).Error)
	_, err = service.GenerateForChild(ctx, 1, "blend", nil)
	require.ErrorIs(t, err, quiz.ErrNoMaterial)
	accepted, err := service.Answer(ctx, 1, q.InstanceID, pinyincontract.AnswerRequest{ClientID: "blend", OptionID: stored.AnswerOptionID})
	require.NoError(t, err)
	require.True(t, accepted.Correct)
	var rows []model.MasterySkill
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, q.KpID, rows[0].KpID)
	require.Equal(t, "blend", rows[0].SkillCode)
	restored, err := service.GetInstance(ctx, 1, q.InstanceID)
	require.NoError(t, err)
	require.Equal(t, q, restored.GeneratedQuestion)
	letter, err := service.GenerateForChild(ctx, 1, "listen", nil)
	require.NoError(t, err)
	_, err = service.Answer(ctx, 1, letter.InstanceID, pinyincontract.AnswerRequest{ClientID: "blend", OptionID: letter.Options[0].ID})
	require.ErrorIs(t, err, quiz.ErrConflict)
	require.NoError(t, db.Migrator().DropTable("pinyin_syllable_assets"))
	_, err = service.GenerateForChild(ctx, 1, "blend", nil)
	require.ErrorIs(t, err, pinyincatalog.ErrUnavailable)
	_, err = service.GenerateForChild(ctx, 1, "listen", nil)
	require.NoError(t, err)
}

func TestFormalAllSkillsUseServerJudgment(t *testing.T) {
	service, db := formalService(t)
	ctx := context.Background()
	for index, kind := range []string{"listen", "inword", "shape", "blend"} {
		q, err := service.GenerateForChild(ctx, 1, kind, nil)
		require.NoError(t, err)
		var stored model.PinyinQuizInstance
		require.NoError(t, db.First(&stored, "id = ?", q.InstanceID).Error)
		selected := stored.AnswerOptionID
		if index%2 == 1 {
			for _, option := range q.Options {
				if option.ID != selected {
					selected = option.ID
					break
				}
			}
		}
		result, err := service.Answer(ctx, 1, q.InstanceID, pinyincontract.AnswerRequest{ClientID: kind, OptionID: selected, CostMs: 100})
		require.NoError(t, err)
		require.Equal(t, index%2 == 0, result.Correct)
		require.Equal(t, kind, result.Skill.Code)
		require.Equal(t, q.KpID, result.Knowledge.KpID)
	}
	var totals struct{ Attempts, Correct int }
	require.NoError(t, db.Table("daily_stats").Select("SUM(attempts) AS attempts,SUM(correct) AS correct").Scan(&totals).Error)
	require.Equal(t, 4, totals.Attempts)
	require.Equal(t, 2, totals.Correct)
}

func TestFormalReceiptFailureRollsBackFirstMasteryReward(t *testing.T) {
	service, db := formalService(t)
	ctx := context.Background()
	q, err := service.GenerateForChild(ctx, 1, "shape", []int64{2, 3, 4})
	require.NoError(t, err)
	for _, code := range []string{"listen", "inword"} {
		require.NoError(t, db.Create(&model.MasterySkill{ChildID: 1, KpID: q.KpID, SkillCode: code, Status: "mastered", Attempts: 2, Correct: 2, Streak: 2, Ease: 2.5}).Error)
	}
	require.NoError(t, db.Create(&model.MasterySkill{ChildID: 1, KpID: q.KpID, SkillCode: "shape", Status: "learning", Attempts: 2, Correct: 2, Streak: 2, Ease: 2.5}).Error)
	var stored model.PinyinQuizInstance
	require.NoError(t, db.First(&stored, "id = ?", q.InstanceID).Error)
	input := pinyincontract.AnswerRequest{ClientID: "mastery", OptionID: stored.AnswerOptionID}
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_mastery_receipt BEFORE INSERT ON pinyin_answer_receipts BEGIN SELECT RAISE(ABORT,'failed receipt'); END`).Error)
	_, err = service.Answer(ctx, 1, q.InstanceID, input)
	require.Error(t, err)
	for _, table := range []string{"attempts", "daily_stats", "mastery_states", "pinyin_mastery_milestones", "flower_ledger", "pinyin_answer_receipts"} {
		var count int64
		require.NoError(t, db.Table(table).Count(&count).Error)
		require.Zero(t, count, table)
	}
	var child model.Child
	require.NoError(t, db.First(&child, 1).Error)
	require.Zero(t, child.Flowers)
	var skill model.MasterySkill
	require.NoError(t, db.Where("child_id=1 AND kp_id=? AND skill_code='shape'", q.KpID).First(&skill).Error)
	require.Equal(t, 2, skill.Attempts)
	require.NoError(t, db.Exec(`DROP TRIGGER fail_mastery_receipt`).Error)
	accepted, err := service.Answer(ctx, 1, q.InstanceID, input)
	require.NoError(t, err)
	require.True(t, accepted.Knowledge.NewlyMastered)
	require.Equal(t, "mastered", accepted.Knowledge.Status)
	retry, err := service.Answer(ctx, 1, q.InstanceID, input)
	require.NoError(t, err)
	require.Equal(t, accepted, retry)
	var count int64
	require.NoError(t, db.Table("flower_ledger").Count(&count).Error)
	require.EqualValues(t, 1, count)
}
