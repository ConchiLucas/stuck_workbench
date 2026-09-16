package service_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/seed"
	"github.com/conchi/study-workbench/internal/service"
)

func newAttemptSvc(t *testing.T) (*service.AttemptService, *gorm.DB) {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	return service.NewAttemptService(repo.New(gdb), mastery.DefaultConfig()), gdb
}

func mathKpID(t *testing.T, gdb *gorm.DB) int64 {
	t.Helper()
	var id int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'math' AND m.code = 'add10'
		ORDER BY kp.order_no LIMIT 1`).Scan(&id).Error)
	require.NotZero(t, id)
	return id
}

func TestReportAttemptsUpdatesMasteryAndStats(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	kpID := mathKpID(t, gdb)
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

	in := []service.AttemptInput{
		{ClientID: "c1", KpID: kpID, IsCorrect: true, CostMs: 3000, Source: "quiz", At: now},
		{ClientID: "c2", KpID: kpID, IsCorrect: true, CostMs: 2500, Source: "quiz", At: now.Add(time.Minute)},
		{ClientID: "c3", KpID: kpID, IsCorrect: true, CostMs: 2000, Source: "quiz", At: now.Add(2 * time.Minute)},
	}
	states, err := svc.Report(1, in)
	require.NoError(t, err)
	require.Len(t, states, 1)
	require.Contains(t, []string{string(mastery.StatusMastered), string(mastery.StatusReviewDue)}, states[0].Status)

	var ms model.MasteryState
	require.NoError(t, gdb.Where("child_id = 1 AND kp_id = ?", kpID).First(&ms).Error)
	require.Equal(t, string(mastery.StatusMastered), ms.Status)

	var ds model.DailyStat
	require.NoError(t, gdb.Where("child_id = 1 AND stat_date = ?", "2026-08-20").First(&ds).Error)
	require.Equal(t, 3, ds.Attempts)
	require.Equal(t, 3, ds.Correct)
	require.Equal(t, 1, ds.NewlyMastered)
	require.Greater(t, ds.PracticeSec, 0)

	var child model.Child
	require.NoError(t, gdb.First(&child, 1).Error)
	require.Equal(t, 1, child.Flowers)
}

func TestReportIsIdempotent(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	kpID := mathKpID(t, gdb)
	now := time.Now()

	in := []service.AttemptInput{
		{ClientID: "dup", KpID: kpID, IsCorrect: true, Source: "quiz", At: now},
	}
	_, err := svc.Report(1, in)
	require.NoError(t, err)
	_, err = svc.Report(1, in)
	require.NoError(t, err)

	var attempts int64
	require.NoError(t, gdb.Model(&model.Attempt{}).Count(&attempts).Error)
	require.Equal(t, int64(1), attempts)

	var st model.MasteryState
	require.NoError(t, gdb.Where("child_id = 1 AND kp_id = ?", kpID).First(&st).Error)
	require.Equal(t, 1, st.Attempts)
}

func TestApplyOneRollsBackAllLearningWrites(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	kpID := mathKpID(t, gdb)
	rollbackErr := errors.New("rollback")

	err := repo.New(gdb).Tx(func(tx *gorm.DB) error {
		_, applied, err := svc.ApplyOne(tx, 1, service.AttemptInput{
			ClientID: "rollback-attempt", KpID: kpID, IsCorrect: true,
			CostMs: 2000, Source: mastery.SourceQuiz, At: time.Now(),
		})
		require.NoError(t, err)
		require.True(t, applied)
		return rollbackErr
	})
	require.ErrorIs(t, err, rollbackErr)

	for table, where := range map[string]string{
		"attempts":       "client_id = 'rollback-attempt'",
		"mastery_states": fmt.Sprintf("child_id = 1 AND kp_id = %d", kpID),
		"mastery_skills": fmt.Sprintf("child_id = 1 AND kp_id = %d", kpID),
		"daily_stats":    "child_id = 1",
		"flower_ledger":  "child_id = 1",
	} {
		var count int64
		require.NoError(t, gdb.Table(table).Where(where).Count(&count).Error)
		require.Zero(t, count, table)
	}
}

func TestApplyOneAggregatesPinyinSkills(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	_, err := seed.Questions(gdb)
	require.NoError(t, err)

	var rows []struct {
		KpID       int64
		QuestionID int64
		Code       string
	}
	require.NoError(t, gdb.Raw(`
		SELECT kp.id AS kp_id, q.id AS question_id, q.code
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		JOIN questions q ON q.kp_id = kp.id
		WHERE s.code = 'pinyin' AND q.code IN ('inword', 'listen')
		ORDER BY kp.id, q.code`).Scan(&rows).Error)
	require.NotEmpty(t, rows)

	byKp := map[int64][]struct {
		KpID       int64
		QuestionID int64
		Code       string
	}{}
	for _, row := range rows {
		byKp[row.KpID] = append(byKp[row.KpID], row)
	}
	var pair = byKp[rows[0].KpID]
	for _, candidates := range byKp {
		if len(candidates) == 2 {
			pair = candidates
			break
		}
	}
	require.Len(t, pair, 2)

	require.NoError(t, repo.New(gdb).Tx(func(tx *gorm.DB) error {
		for i, row := range pair {
			questionID := row.QuestionID
			_, applied, err := svc.ApplyOne(tx, 1, service.AttemptInput{
				ClientID: fmt.Sprintf("pinyin-skill-%d", i), KpID: row.KpID,
				QuestionID: &questionID, IsCorrect: true, Source: mastery.SourceQuiz,
				At: time.Now().Add(time.Duration(i) * time.Minute),
			})
			if err != nil {
				return err
			}
			require.True(t, applied)
		}
		return nil
	}))

	var skills []model.MasterySkill
	require.NoError(t, gdb.Where("child_id = ? AND kp_id = ?", 1, pair[0].KpID).
		Order("skill_code").Find(&skills).Error)
	require.Len(t, skills, 2)
	require.Equal(t, []string{mastery.SkillPinyinInWord, mastery.SkillPinyinListen},
		[]string{skills[0].SkillCode, skills[1].SkillCode})
	require.Equal(t, 2, skills[0].Attempts+skills[1].Attempts)

	var rolled model.MasteryState
	require.NoError(t, gdb.Where("child_id = ? AND kp_id = ?", 1, pair[0].KpID).
		First(&rolled).Error)
	require.NotEqual(t, string(mastery.StatusNotStarted), rolled.Status)
}

func TestMarkMasteredAndUndo(t *testing.T) {
	svc, gdb := newAttemptSvc(t)
	kpID := mathKpID(t, gdb)
	now := time.Now()

	_, err := svc.Report(1, []service.AttemptInput{
		{ClientID: "a1", KpID: kpID, IsCorrect: true, Source: "quiz", At: now},
		{ClientID: "a2", KpID: kpID, IsCorrect: false, Source: "quiz", At: now.Add(time.Minute)},
	})
	require.NoError(t, err)

	st, err := svc.MarkMastered(1, kpID)
	require.NoError(t, err)
	require.Equal(t, string(mastery.StatusMastered), st.Status)

	st, err = svc.UndoMark(1, kpID)
	require.NoError(t, err)
	require.Equal(t, string(mastery.StatusLearning), st.Status)
	require.Equal(t, 2, st.Attempts)
}

func newDashboard(t *testing.T) (*service.DashboardService, *gorm.DB) {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, seed.Demo(gdb, mastery.DefaultConfig(), 1, 60))
	return service.NewDashboardService(repo.New(gdb)), gdb
}

func catalogSize(t *testing.T, gdb *gorm.DB, subjectCode string) (subjects, modules, points int) {
	t.Helper()
	require.NoError(t, gdb.Raw(`SELECT COUNT(1) FROM subjects`).Scan(&subjects).Error)
	require.NoError(t, gdb.Raw(`
		SELECT COUNT(1) FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ?`, subjectCode).Scan(&modules).Error)
	require.NoError(t, gdb.Raw(`
		SELECT COUNT(1) FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ?`, subjectCode).Scan(&points).Error)
	return subjects, modules, points
}

func TestOverviewCountsAddUpToTotal(t *testing.T) {
	svc, gdb := newDashboard(t)
	ov, err := svc.Overview(1)
	require.NoError(t, err)

	var catalogTotal int
	require.NoError(t, gdb.Raw(`SELECT COUNT(1) FROM knowledge_points`).Scan(&catalogTotal).Error)
	require.Equal(t, catalogTotal, ov.TotalKp)
	sum := ov.Counts.Mastered + ov.Counts.Learning + ov.Counts.Shaky +
		ov.Counts.ReviewDue + ov.Counts.NotStarted
	require.Equal(t, ov.TotalKp, sum)
	require.Greater(t, ov.Counts.Mastered, 0)
	require.Equal(t, "卢沁一", ov.Child.Name)
}

func TestSubjectsSummary(t *testing.T) {
	svc, gdb := newDashboard(t)
	list, err := svc.Subjects(1)
	require.NoError(t, err)
	subjectCount, _, mathCount := catalogSize(t, gdb, "math")
	_, _, englishCount := catalogSize(t, gdb, "english")
	require.Len(t, list, subjectCount-1)

	byCode := map[string]service.SubjectSummary{}
	for _, s := range list {
		byCode[s.Code] = s
		require.Equal(t, s.Total, s.Counts.Mastered+s.Counts.Learning+
			s.Counts.Shaky+s.Counts.ReviewDue+s.Counts.NotStarted)
	}
	require.NotContains(t, byCode, "game")
	require.Equal(t, mathCount, byCode["math"].Total)
	require.Equal(t, englishCount, byCode["english"].Total)
	require.GreaterOrEqual(t, len(byCode["english"].QuestionTypes), 5)
	codes := map[string]string{}
	for _, tpe := range byCode["english"].QuestionTypes {
		codes[tpe.Code] = tpe.Name
		require.Equal(t, englishCount, tpe.Total)
	}
	require.Equal(t, "听音选词", codes["listen"])
	require.Equal(t, "看图选词", codes["picture"])
	require.Equal(t, "组句子", codes["build"])
	require.Equal(t, "写单词", codes["type"])
	require.Equal(t, "读一读", codes["read"])
	require.NotContains(t, codes, "单词认读")
}

func TestSubjectsQuestionTypesStayZeroWithoutSkills(t *testing.T) {
	svc, gdb := newDashboard(t)
	require.NoError(t, gdb.Create(&model.Child{Name: "英语空账本（测试）", Grade: "验收演示"}).Error)
	var id int64
	require.NoError(t, gdb.Raw(`SELECT id FROM children WHERE name = ?`, "英语空账本（测试）").Scan(&id).Error)
	list, err := svc.Subjects(id)
	require.NoError(t, err)
	var english service.SubjectSummary
	for _, s := range list {
		if s.Code == "english" {
			english = s
		}
	}
	require.GreaterOrEqual(t, len(english.QuestionTypes), 5)
	for _, tpe := range english.QuestionTypes {
		require.Equal(t, 0, tpe.Mastered)
		require.Equal(t, 0, tpe.Attempted)
		require.Greater(t, tpe.Total, 0)
	}
	ov, err := svc.Overview(id)
	require.NoError(t, err)
	require.Equal(t, 0, ov.Counts.Mastered)
	require.Equal(t, 0, ov.Counts.Learning)
}

func TestMatrixGroupsByModule(t *testing.T) {
	svc, gdb := newDashboard(t)
	m, err := svc.Matrix(1, "math")
	require.NoError(t, err)
	_, moduleCount, pointCount := catalogSize(t, gdb, "math")

	require.Equal(t, "算术", m.Subject.Name)
	require.Equal(t, pointCount, m.Subject.Total)
	require.Len(t, m.Modules, moduleCount)

	total := 0
	for _, mod := range m.Modules {
		require.NotEmpty(t, mod.Name)
		require.Len(t, mod.Points, mod.Total)
		total += len(mod.Points)
		for _, p := range mod.Points {
			require.NotEmpty(t, p.Title)
			require.Contains(t,
				[]string{"not_started", "learning", "shaky", "mastered", "review_due"}, p.Status)
		}
	}
	require.Equal(t, pointCount, total)
}

func TestAttentionRanksWorstFirst(t *testing.T) {
	svc, _ := newDashboard(t)
	list, err := svc.Attention(1, 10)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	require.LessOrEqual(t, len(list), 10)

	for i := range list {
		require.Contains(t, []string{"shaky", "review_due"}, list[i].Status)
		require.NotEmpty(t, list[i].SubjectName)
	}
}

func TestKpDetailIncludesHistory(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp_id FROM attempts WHERE child_id = 1
		GROUP BY kp_id ORDER BY COUNT(1) DESC LIMIT 1`).Scan(&kpID).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	require.Equal(t, kpID, d.KpID)
	require.NotEmpty(t, d.Title)
	require.NotEmpty(t, d.History)
	require.Equal(t, len(d.History), d.Attempts)
}

func TestKpDetailLiteracyIncludesSkills(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy'
		ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)
	require.NotZero(t, kpID)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	require.Equal(t, "literacy", d.SubjectCode)
	require.Len(t, d.Skills, 3)
	require.Equal(t, mastery.LiteracySkills, []string{d.Skills[0].Code, d.Skills[1].Code, d.Skills[2].Code})
	for _, sk := range d.Skills {
		require.Contains(t, []string{"not_started", "learning", "shaky", "mastered", "review_due"}, sk.Status)
	}
}

func TestKpDetailLiteracyNoSkillRowsNotStarted(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy'
		ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)
	require.NotZero(t, kpID)
	require.NoError(t, gdb.Where("child_id = ? AND kp_id = ?", 1, kpID).Delete(&model.MasterySkill{}).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	require.Equal(t, "not_started", d.Status)
	require.Len(t, d.Skills, 3)
	for _, sk := range d.Skills {
		require.Equal(t, "not_started", sk.Status)
	}
}

func TestKpDetailHistorySkillCodeFromQuestion(t *testing.T) {
	svc, gdb := newDashboard(t)

	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy'
		ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)
	require.NotZero(t, kpID)

	q := model.Question{
		KpID: kpID, Code: mastery.SkillGlyphSense, Type: "choice",
		Stem: "test", Options: "[]", Answer: "0",
	}
	require.NoError(t, gdb.Create(&q).Error)

	qID := q.ID
	require.NoError(t, gdb.Create(&model.Attempt{
		ChildID: 1, KpID: kpID, QuestionID: &qID,
		IsCorrect: true, CostMs: 1200, Source: "quiz",
		ClientID: fmt.Sprintf("test-skill-%d", qID),
	}).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)

	var found bool
	for _, h := range d.History {
		if h.SkillCode == mastery.SkillGlyphSense {
			found = true
			require.True(t, h.IsCorrect)
		}
	}
	require.True(t, found, "expected history row with skill_code=glyph_sense")
}

func TestKpDetailParentMarkHasEmptySkillCode(t *testing.T) {
	svc, gdb := newDashboard(t)
	var kpID int64
	require.NoError(t, gdb.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy' ORDER BY kp.id LIMIT 1`).Scan(&kpID).Error)

	require.NoError(t, gdb.Create(&model.Attempt{
		ChildID: 1, KpID: kpID, QuestionID: nil,
		IsCorrect: true, CostMs: 0, Source: "parent_mark",
		ClientID: fmt.Sprintf("parent-mark-%d", kpID),
	}).Error)

	d, err := svc.KpDetail(1, kpID)
	require.NoError(t, err)
	var sawMark bool
	for _, h := range d.History {
		if h.Source == "parent_mark" {
			sawMark = true
			require.Empty(t, h.SkillCode)
		}
	}
	require.True(t, sawMark)
}

func newStats(t *testing.T) *service.StatsService {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, seed.Demo(gdb, mastery.DefaultConfig(), 1, 60))
	return service.NewStatsService(repo.New(gdb))
}

func TestTrendFillsEmptyDays(t *testing.T) {
	svc := newStats(t)
	pts, err := svc.Trend(1, 30)
	require.NoError(t, err)
	require.Len(t, pts, 30)

	cumulative := 0
	for _, p := range pts {
		require.NotEmpty(t, p.Date)
		require.GreaterOrEqual(t, p.CumulativeMastered, cumulative)
		cumulative = p.CumulativeMastered
	}
	require.Greater(t, cumulative, 0)
}

func TestCalendarReturnsActiveDays(t *testing.T) {
	svc := newStats(t)
	days, err := svc.Calendar(1, 3)
	require.NoError(t, err)
	require.NotEmpty(t, days)
}

func TestReviewQueuePrioritisesShaky(t *testing.T) {
	svc := newStats(t)
	q, err := svc.ReviewQueue(1, 20)
	require.NoError(t, err)
	require.NotEmpty(t, q)
	require.Equal(t, "shaky", q[0].Status)
}

func TestRedeemRewardDeductsFlowers(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	require.NoError(t, gdb.Create(&model.Reward{ChildID: 1, Name: "看动画片 20 分钟", Cost: 5, Stock: 3}).Error)
	require.NoError(t, gdb.Model(&model.Child{}).Where("id = 1").Update("flowers", 6).Error)

	svc := service.NewRewardService(repo.New(gdb))
	require.NoError(t, svc.Redeem(1, 1))

	var child model.Child
	require.NoError(t, gdb.First(&child, 1).Error)
	require.Equal(t, 1, child.Flowers)

	require.Error(t, svc.Redeem(1, 1))
}
