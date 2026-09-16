package learning_test

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
)

func newLearningDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(
		&model.Child{}, &model.KnowledgePoint{}, &model.Question{}, &model.Attempt{},
		&model.MasteryState{}, &model.MasterySkill{}, &model.DailyStat{}, &model.FlowerLedger{},
	))
	require.NoError(t, gdb.Exec(`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER NOT NULL, code TEXT NOT NULL DEFAULT '')`).Error)
	require.NoError(t, gdb.Exec(`CREATE UNIQUE INDEX attempts_child_client ON attempts(child_id, client_id)`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO subjects(id, code) VALUES(1, 'pinyin')`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO modules(id, subject_id, code) VALUES(1, 1, 'initials')`).Error)
	require.NoError(t, gdb.Create(&model.Child{ID: 1, Name: "Test"}).Error)
	require.NoError(t, gdb.Create(&model.KnowledgePoint{
		ID: 10, ModuleID: 1, Code: "sm001", Title: "b", Difficulty: 1,
	}).Error)
	require.NoError(t, gdb.Create(&model.Question{
		ID: 20, KpID: 10, Code: mastery.SkillPinyinInWord,
	}).Error)
	return gdb
}

func newMathLearningDB(t *testing.T, moduleCode string) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(
		&model.Child{}, &model.KnowledgePoint{}, &model.Question{}, &model.Attempt{},
		&model.MasteryState{}, &model.MasterySkill{}, &model.DailyStat{}, &model.FlowerLedger{},
	))
	require.NoError(t, gdb.Exec(`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER NOT NULL, code TEXT NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`CREATE UNIQUE INDEX attempts_child_client ON attempts(child_id, client_id)`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO subjects(id, code) VALUES(1, 'math')`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO modules(id, subject_id, code) VALUES(1, 1, ?)`, moduleCode).Error)
	require.NoError(t, gdb.Create(&model.Child{ID: 1, Name: "Test"}).Error)
	require.NoError(t, gdb.Create(&model.KnowledgePoint{
		ID: 10, ModuleID: 1, Code: moduleCode + "-1", Title: "Math", Difficulty: 1,
	}).Error)

	codes := []string{mastery.SkillMathCalc, mastery.SkillMathStory}
	if moduleCode == "shape" {
		codes = []string{mastery.SkillMathFind, mastery.SkillMathName}
	}
	for i, code := range codes {
		require.NoError(t, gdb.Create(&model.Question{ID: int64(20 + i), KpID: 10, Code: code}).Error)
	}
	return gdb
}

func applyCorrect(t *testing.T, gdb *gorm.DB, svc *learning.Service, questionID int64, clientID string, at time.Time) {
	t.Helper()
	require.NoError(t, gdb.Transaction(func(tx *gorm.DB) error {
		_, applied, err := svc.ApplyOne(tx, 1, learning.AttemptInput{
			ClientID: clientID, KpID: 10, QuestionID: &questionID,
			IsCorrect: true, CostMs: 1000, Source: mastery.SourceQuiz, At: at,
		})
		require.True(t, applied)
		return err
	}))
}

func requireSkillStatus(t *testing.T, gdb *gorm.DB, code, want string) {
	t.Helper()
	var row model.MasterySkill
	err := gdb.Where("child_id = ? AND kp_id = ? AND skill_code = ?", 1, 10, code).First(&row).Error
	if want == string(mastery.StatusNotStarted) {
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		return
	}
	require.NoError(t, err)
	require.Equal(t, want, row.Status)
}

func requireKpStatus(t *testing.T, gdb *gorm.DB, want string) {
	t.Helper()
	var row model.MasteryState
	require.NoError(t, gdb.Where("child_id = ? AND kp_id = ?", 1, 10).First(&row).Error)
	require.Equal(t, want, row.Status)
}

func TestMathKnowledgePointRequiresBothModuleSkills(t *testing.T) {
	gdb := newMathLearningDB(t, "add10")
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	svc := learning.NewService(cfg)
	// These assertions concern newly learned skills. Aggregate status compares due dates with the real clock.
	at := time.Now().UTC()

	applyCorrect(t, gdb, svc, 20, "calc-1", at)
	requireSkillStatus(t, gdb, mastery.SkillMathCalc, string(mastery.StatusMastered))
	requireSkillStatus(t, gdb, mastery.SkillMathStory, string(mastery.StatusNotStarted))
	requireKpStatus(t, gdb, string(mastery.StatusLearning))

	applyCorrect(t, gdb, svc, 21, "story-1", at.Add(time.Minute))
	requireKpStatus(t, gdb, string(mastery.StatusMastered))
}

func TestMathAttemptCanUseFrozenPlanSkillCode(t *testing.T) {
	gdb := newMathLearningDB(t, "add10")
	require.NoError(t, gdb.Model(&model.Question{}).Where("id = ?", 20).Update("code", mastery.SkillMathStory).Error)
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	svc := learning.NewService(cfg)
	questionID := int64(20)
	require.NoError(t, gdb.Transaction(func(tx *gorm.DB) error {
		_, _, err := svc.ApplyOne(tx, 1, learning.AttemptInput{
			ClientID: "snapshot-calc", KpID: 10, QuestionID: &questionID, SkillCode: mastery.SkillMathCalc,
			IsCorrect: true, CostMs: 1000, Source: mastery.SourceQuiz,
		})
		return err
	}))
	requireSkillStatus(t, gdb, mastery.SkillMathCalc, string(mastery.StatusMastered))
	requireSkillStatus(t, gdb, mastery.SkillMathStory, string(mastery.StatusNotStarted))
}

func TestMathShapeRequiresFindAndName(t *testing.T) {
	gdb := newMathLearningDB(t, "shape")
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	svc := learning.NewService(cfg)
	// These assertions concern newly learned skills. Aggregate status compares due dates with the real clock.
	at := time.Now().UTC()

	applyCorrect(t, gdb, svc, 20, "find-1", at)
	requireKpStatus(t, gdb, string(mastery.StatusLearning))
	applyCorrect(t, gdb, svc, 21, "name-1", at.Add(time.Minute))
	requireKpStatus(t, gdb, string(mastery.StatusMastered))
}

func TestEnglishKnowledgePointRequiresAllFiveSkills(t *testing.T) {
	gdb := newLearningDB(t)
	require.NoError(t, gdb.Exec(`UPDATE subjects SET code = 'english' WHERE id = 1`).Error)
	require.NoError(t, gdb.Exec(`UPDATE modules SET code = 'animals' WHERE id = 1`).Error)
	require.NoError(t, gdb.Exec(`UPDATE questions SET code = 'listen' WHERE id = 20`).Error)
	for i, code := range []string{"picture", "build", "type", "read"} {
		require.NoError(t, gdb.Create(&model.Question{ID: int64(21 + i), KpID: 10, Code: code}).Error)
	}

	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	svc := learning.NewService(cfg)
	at := time.Now().UTC()

	applyCorrect(t, gdb, svc, 20, "english-listen-1", at)
	requireSkillStatus(t, gdb, mastery.SkillEnglishListen, string(mastery.StatusMastered))
	requireSkillStatus(t, gdb, mastery.SkillEnglishPicture, string(mastery.StatusNotStarted))
	requireKpStatus(t, gdb, string(mastery.StatusLearning))

	applyCorrect(t, gdb, svc, 21, "english-picture-1", at.Add(time.Minute))
	requireKpStatus(t, gdb, string(mastery.StatusLearning))
	applyCorrect(t, gdb, svc, 22, "english-build-1", at.Add(2*time.Minute))
	applyCorrect(t, gdb, svc, 23, "english-type-1", at.Add(3*time.Minute))
	applyCorrect(t, gdb, svc, 24, "english-read-1", at.Add(4*time.Minute))
	requireKpStatus(t, gdb, string(mastery.StatusMastered))
}

func TestApplyOneIsIdempotent(t *testing.T) {
	gdb := newLearningDB(t)
	svc := learning.NewService(mastery.DefaultConfig())
	questionID := int64(20)
	in := learning.AttemptInput{
		ClientID: "pinyin-plan-1-item-1-try-1", KpID: 10, QuestionID: &questionID,
		IsCorrect: true, CostMs: 2000, Source: mastery.SourceQuiz,
		At: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
	}

	for i, expectedApplied := range []bool{true, false} {
		err := gdb.Transaction(func(tx *gorm.DB) error {
			_, applied, err := svc.ApplyOne(tx, 1, in)
			require.Equal(t, expectedApplied, applied, "call %d", i+1)
			return err
		})
		require.NoError(t, err)
	}

	var attempts int64
	require.NoError(t, gdb.Model(&model.Attempt{}).Count(&attempts).Error)
	require.Equal(t, int64(1), attempts)

	var daily model.DailyStat
	require.NoError(t, gdb.Where("child_id = ?", 1).First(&daily).Error)
	require.Equal(t, 1, daily.Attempts)
	require.Equal(t, 1, daily.Correct)
}

func TestPhraseReplyAttemptIsRecordedWithoutGrantingMastery(t *testing.T) {
	gdb := newLearningDB(t)
	require.NoError(t, gdb.Exec(`UPDATE subjects SET code = 'phrase' WHERE id = 1`).Error)
	require.NoError(t, gdb.Exec(`UPDATE modules SET code = 'greet' WHERE id = 1`).Error)
	require.NoError(t, gdb.Exec(`UPDATE questions SET code = 'reply' WHERE id = 20`).Error)
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	svc := learning.NewService(cfg)
	qid := int64(20)
	item := int64(88)
	require.NoError(t, gdb.Transaction(func(tx *gorm.DB) error {
		_, applied, err := svc.ApplyOne(tx, 1, learning.AttemptInput{
			ClientID: "phrase-reply-right", KpID: 10, QuestionID: &qid, PlanItemID: &item,
			SkillCode: "reply", Selected: "101", IsCorrect: true, CostMs: 700,
			Source: mastery.SourceQuiz, At: time.Now().UTC(),
		})
		require.True(t, applied)
		return err
	}))
	requireSkillStatus(t, gdb, mastery.SkillPhraseReply, string(mastery.StatusMastered))
	requireSkillStatus(t, gdb, mastery.SkillPhraseListenZh, string(mastery.StatusNotStarted))
	requireKpStatus(t, gdb, string(mastery.StatusNotStarted))
	var rows []model.Attempt
	require.NoError(t, gdb.Where("plan_item_id = ?", item).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "101", rows[0].Selected)
	require.True(t, rows[0].IsCorrect)
}

func TestApplyOneStoresSelectedPerAttempt(t *testing.T) {
	gdb := newLearningDB(t)
	svc := learning.NewService(mastery.DefaultConfig())
	qid := int64(20)
	item := int64(44)
	require.NoError(t, gdb.Transaction(func(tx *gorm.DB) error {
		_, applied, err := svc.ApplyOne(tx, 1, learning.AttemptInput{
			ClientID: "spell-wrong", KpID: 10, QuestionID: &qid, PlanItemID: &item,
			Selected: "aple", IsCorrect: false, CostMs: 800, Source: mastery.SourceQuiz,
			At: time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC),
		})
		require.True(t, applied)
		return err
	}))
	require.NoError(t, gdb.Transaction(func(tx *gorm.DB) error {
		_, applied, err := svc.ApplyOne(tx, 1, learning.AttemptInput{
			ClientID: "spell-right", KpID: 10, QuestionID: &qid, PlanItemID: &item,
			Selected: "apple", IsCorrect: true, CostMs: 700, Source: mastery.SourceQuiz,
			At: time.Date(2026, 9, 14, 10, 1, 0, 0, time.UTC),
		})
		require.True(t, applied)
		return err
	}))
	var rows []model.Attempt
	require.NoError(t, gdb.Where("plan_item_id = ?", item).Order("id").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, "aple", rows[0].Selected)
	require.False(t, rows[0].IsCorrect)
	require.Equal(t, "apple", rows[1].Selected)
	require.True(t, rows[1].IsCorrect)
}
