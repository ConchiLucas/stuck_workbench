package diagnosis_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-diagnosis-admin/internal/diagnosis"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
)

func TestBuildHeadlineNamesSkillGapAndShakyCount(t *testing.T) {
	got := diagnosis.BuildHeadline(diagnosis.HeadlineFacts{
		Shaky:             1,
		LiteracyWriteWeak: true,
		EnglishListenWeak: true,
	})
	require.Contains(t, got, "识字能认、手写偏弱")
	require.Contains(t, got, "英语听音不稳")
	require.Contains(t, got, "1 个知识点需巩固")
}

func TestBuildHeadlineWhenHealthy(t *testing.T) {
	require.Equal(t, "目前没有突出薄弱点，可以按计划推进。", diagnosis.BuildHeadline(diagnosis.HeadlineFacts{}))
}

func TestOverviewReadsChildAndUrgentWeakPoints(t *testing.T) {
	svc := diagnosis.NewService(testdb.Open(t))
	out, err := svc.Overview(1)
	require.NoError(t, err)
	require.Equal(t, "卢沁一", out.Child.Name)
	require.Contains(t, out.Headline, "识字能认、手写偏弱")
	require.NotEmpty(t, out.Urgent)
	require.Equal(t, "Hello!", out.Urgent[0].Title)
	require.Equal(t, "shaky", out.Urgent[0].Status)
	var literacy diagnosis.SubjectHealth
	for _, s := range out.Subjects {
		if s.Code == "literacy" {
			literacy = s
		}
	}
	require.Equal(t, "watch", literacy.Health)
}

func TestOverviewOmitsGameSubject(t *testing.T) {
	svc := diagnosis.NewService(testdb.Open(t))
	out, err := svc.Overview(1)
	require.NoError(t, err)
	for _, s := range out.Subjects {
		require.NotEqual(t, "game", s.Code)
	}
	for _, u := range out.Urgent {
		require.NotEqual(t, "game", u.SubjectCode)
	}
}

func TestSubjectDiagnosisRejectsGame(t *testing.T) {
	svc := diagnosis.NewService(testdb.Open(t))
	_, err := svc.Subject(1, "game")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestSubjectDiagnosisReportsSkillGap(t *testing.T) {
	svc := diagnosis.NewService(testdb.Open(t))
	out, err := svc.Subject(1, "literacy")
	require.NoError(t, err)
	require.Equal(t, "识字", out.Name)
	require.NotEmpty(t, out.SkillGaps)
	require.Equal(t, "write_char", out.SkillGaps[0].WeakSkill)
	require.Equal(t, "glyph_sense", out.SkillGaps[0].StrongSkill)
}

func TestErrorPatternsIncludeImbalanceRepeatedWrongAndSlow(t *testing.T) {
	svc := diagnosis.NewService(testdb.Open(t))
	out, err := svc.ErrorPatterns(1)
	require.NoError(t, err)
	kinds := map[string]bool{}
	for _, p := range out {
		kinds[p.Kind] = true
	}
	require.True(t, kinds["skill_imbalance"])
	require.True(t, kinds["observed_wrong"])
	require.False(t, kinds["repeated_wrong"])
	require.True(t, kinds["slow_wrong"])
}

func TestKpArchiveIncludesHistoryAndSkills(t *testing.T) {
	svc := diagnosis.NewService(testdb.Open(t))
	out, err := svc.KpArchive(1, 10)
	require.NoError(t, err)
	require.Equal(t, "一", out.Title)
	require.Equal(t, "literacy", out.SubjectCode)
	require.Len(t, out.History, 1)
	require.NotEmpty(t, out.Skills)
}

func TestArchiveReadsVersionReceiptSkillAndWrongPick(t *testing.T) {
	gdb := testdb.Open(t)
	require.NoError(t, gdb.Exec(`CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,attempt_id INTEGER,child_id INTEGER,client_id TEXT,plan_id INTEGER,plan_item_id INTEGER,question_version_id INTEGER,kp_id INTEGER,skill_code TEXT,question_type TEXT,selected_option_id TEXT,display_index INTEGER,original_index INTEGER,is_correct BOOLEAN,cost_ms INTEGER,response_json TEXT,created_at DATETIME)`).Error)
	var kpID int64
	require.NoError(t, gdb.Table("knowledge_points kp").Select("kp.id").Joins("JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id").Where("s.code = ?", "literacy").Limit(1).Scan(&kpID).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at) VALUES(999,1,?,NULL,0,100,'quiz','version-test',CURRENT_TIMESTAMP)`, kpID).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO question_attempt_receipts(id,attempt_id,child_id,kp_id,skill_code,question_type,selected_option_id,display_index,original_index,is_correct,response_json,created_at) VALUES(999,999,1,?,'sense_char','sense_char','kp:wrong',2,1,0,'{"answerIndex":0}',CURRENT_TIMESTAMP)`, kpID).Error)
	archive, err := diagnosis.NewService(gdb).KpArchive(1, kpID)
	require.NoError(t, err)
	found := false
	for _, h := range archive.History {
		if h.CostMs == 100 {
			require.Equal(t, "sense_char", h.SkillCode)
			found = true
		}
	}
	require.True(t, found)
	found = false
	for _, w := range archive.RecentWrong {
		if w.Picks == "2" {
			found = true
		}
	}
	require.True(t, found)
}

func TestWritingFailureIsNotAnOptionPick(t *testing.T) {
	d := testdb.Open(t)
	require.NoError(t, d.Exec(`CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,attempt_id INTEGER,child_id INTEGER,kp_id INTEGER,skill_code TEXT,question_type TEXT,selected_option_id TEXT,display_index INTEGER,original_index INTEGER,is_correct BOOLEAN,response_json TEXT,created_at DATETIME,response_kind TEXT,answer_payload_json TEXT,evaluation_json TEXT,question_version_id INTEGER)`).Error)
	require.NoError(t, d.Exec(`INSERT INTO question_attempt_receipts VALUES(1,1,1,10,'write_char','write_char',NULL,NULL,NULL,0,'{"answerIndex":-1}',CURRENT_TIMESTAMP,'handwriting','{"kind":"handwriting","strokes":[]}','{"outcome":"not_passed","assistance":"none"}',1)`).Error)
	svc := diagnosis.NewService(d)
	archive, err := svc.KpArchive(1, 10)
	require.NoError(t, err)
	found := false
	for _, w := range archive.RecentWrong {
		if w.ResponseKind == "handwriting" {
			found = true
			require.Empty(t, w.Picks)
			require.Contains(t, string(w.EvaluationJSON), "not_passed")
		}
	}
	require.True(t, found)
	patterns, err := svc.ErrorPatterns(1)
	require.NoError(t, err)
	found = false
	for _, p := range patterns {
		if p.Kind == "writing_not_passed" {
			found = true
			require.NotContains(t, p.Detail, "第")
		}
	}
	require.True(t, found)
}

func TestArchiveCountsAssistedFactsWithoutMasteryEvidence(t *testing.T) {
	d := testdb.Open(t)
	require.NoError(t, d.Exec("DELETE FROM attempts WHERE kp_id=10 AND child_id=1").Error)
	require.NoError(t, d.Exec("DELETE FROM mastery_states WHERE kp_id=10 AND child_id=1").Error)
	require.NoError(t, d.Exec("DELETE FROM mastery_skills WHERE kp_id=10 AND child_id=1").Error)
	require.NoError(t, d.Exec(`CREATE TABLE question_attempt_receipts(attempt_id INTEGER,child_id INTEGER,kp_id INTEGER,is_correct BOOLEAN,created_at DATETIME,id INTEGER,skill_code TEXT)`).Error)
	require.NoError(t, d.Exec(`INSERT INTO attempts(id,child_id,kp_id,is_correct,cost_ms,source,client_id,created_at) VALUES(999,1,10,1,500,'quiz','hinted',CURRENT_TIMESTAMP)`).Error)
	require.NoError(t, d.Exec(`INSERT INTO question_attempt_receipts VALUES(999,1,10,1,CURRENT_TIMESTAMP,999,'write_char')`).Error)
	a, e := diagnosis.NewService(d).KpArchive(1, 10)
	require.NoError(t, e)
	require.Equal(t, 1, a.Attempts)
	require.Equal(t, float64(1), a.Accuracy)
	for _, s := range a.Skills {
		if s.Code == "write_char" {
			require.Equal(t, 1, s.Attempts)
			require.Equal(t, "not_started", s.Status)
		}
	}
}
