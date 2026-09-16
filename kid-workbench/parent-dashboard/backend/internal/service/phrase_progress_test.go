package service_test

import (
	"testing"
	"time"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/phrasecontent"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPhraseHistoryUsesLinkedPlanItemNotLatest(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='phrase' AND kp.title='Good morning.'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "listen_zh", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	first := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","selected":"2","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"早上好。"},{"id":"2","label":"下午好。"},{"id":"3","label":"晚上好。"},{"id":"4","label":"晚安。"}],"answerId":"1"}}`
	second := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","selected":"1","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"4","label":"晚安。"},{"id":"1","label":"早上好。"},{"id":"3","label":"晚上好。"},{"id":"2","label":"下午好。"}],"answerId":"1"}}`
	updated := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","selected":"1","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.mp3","options":[{"id":"4","label":"晚安。"},{"id":"1","label":"早上好。"},{"id":"3","label":"晚上好。"},{"id":"2","label":"下午好。"}],"answerId":"1"}}`
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	p1 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-10", SeqNo: 91, SubjectCode: "phrase", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	p2 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-11", SeqNo: 91, SubjectCode: "phrase", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now.Add(24 * time.Hour)}
	require.NoError(t, d.Create(&p1).Error)
	require.NoError(t, d.Create(&p2).Error)
	item1 := model.PlanItem{PlanID: p1.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: "2", QuestionSnapshot: first}
	item2 := model.PlanItem{PlanID: p2.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 1, CostMs: 800, Picks: "1", QuestionSnapshot: second}
	require.NoError(t, d.Create(&item1).Error)
	require.NoError(t, d.Create(&item2).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "ph-link-1", PlanItemID: &item1.ID, CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "ph-link-2", PlanItemID: &item2.ID, CreatedAt: now.Add(24 * time.Hour)}).Error)
	require.NoError(t, d.Model(&item2).Update("question_snapshot", updated).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "ph-unlinked", CreatedAt: now.Add(48 * time.Hour)}).Error)

	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 3)
	wrong := detail.History[0]
	require.NotNil(t, wrong.PhraseReview)
	require.NotNil(t, wrong.PhraseReview.Example)
	require.Equal(t, "2", wrong.PhraseReview.Selected)
	require.Equal(t, "下午好。", wrong.PhraseReview.Example.Options[1].Label)
	require.Equal(t, []string{"1", "2", "3", "4"}, phraseOptionIDs(wrong.PhraseReview.Example.Options))
	later := detail.History[1]
	require.Equal(t, "1", later.PhraseReview.Selected)
	require.Equal(t, []string{"4", "1", "3", "2"}, phraseOptionIDs(later.PhraseReview.Example.Options))
	require.Contains(t, later.PhraseReview.Example.SpeechURL, "fff")
	unlinked := detail.History[2]
	require.NotNil(t, unlinked.PhraseReview)
	require.Contains(t, unlinked.PhraseReview.UnavailableReason, "无法还原")
	require.Nil(t, unlinked.PhraseReview.Example)
}

func TestPhraseHistorySamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='phrase' AND kp.title='Good morning.'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "listen_zh", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	snap := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/items/20/speech.mp3","options":[{"id":"20","label":"早上好。"},{"id":"21","label":"下午好。"},{"id":"22","label":"晚上好。"},{"id":"23","label":"晚安。"}],"answerId":"20"}}`
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-14", SeqNo: 92, SubjectCode: "phrase", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 2, CostMs: 1600, Picks: `[{"clientId":"a","optionIndex":1},{"clientId":"b","optionIndex":0}]`, QuestionSnapshot: snap}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "ph-same-1", PlanItemID: &item.ID, Selected: "21", CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "ph-same-2", PlanItemID: &item.ID, Selected: "20", CreatedAt: now.Add(time.Minute)}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 2)
	var wrong, right *service.HistoryItem
	for i := range detail.History {
		if detail.History[i].PhraseReview == nil {
			continue
		}
		if detail.History[i].PhraseReview.Selected == "21" {
			wrong = &detail.History[i]
		}
		if detail.History[i].PhraseReview.Selected == "20" && detail.History[i].IsCorrect {
			right = &detail.History[i]
		}
	}
	require.NotNil(t, wrong)
	require.NotNil(t, right)
	require.Equal(t, "21", wrong.PhraseReview.Selected)
	require.Equal(t, "20", right.PhraseReview.Selected)
	require.Equal(t, wrong.PhraseReview.Example.SpeechURL, right.PhraseReview.Example.SpeechURL)
}

func TestPhraseHistoryRejectsCodeTypeOnlySnapshot(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='phrase' AND kp.title='Good morning.'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "listen_zh", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	now := time.Date(2026, 9, 14, 11, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-14", SeqNo: 93, SubjectCode: "phrase", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: "2", QuestionSnapshot: `{"code":"listen_zh","type":"choice"}`}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "ph-code-type", PlanItemID: &item.ID, Selected: "2", CreatedAt: now}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	var found *service.HistoryItem
	for i := range detail.History {
		if detail.History[i].PhraseReview != nil && detail.History[i].PhraseReview.UnavailableReason != "" {
			found = &detail.History[i]
		}
	}
	require.NotNil(t, found)
	require.Nil(t, found.PhraseReview.Example)
	require.Contains(t, found.PhraseReview.UnavailableReason, "无法还原")
}

func phraseOptionIDs(opts []phrasecontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}

func TestPhraseMatrixShowsReplyWithoutGrantingMastery(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='phrase' AND kp.title='I''m fine.'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	require.NoError(t, d.Create(&model.MasterySkill{
		ChildID: 1, KpID: kp, SkillCode: mastery.SkillPhraseReply,
		Status: string(mastery.StatusMastered), Attempts: 2, Correct: 1, Ease: 2.5,
	}).Error)
	matrix, err := service.NewDashboardService(repo.New(d)).Matrix(1, "phrase")
	require.NoError(t, err)
	var point *service.MatrixPoint
	for _, mod := range matrix.Modules {
		for i := range mod.Points {
			if mod.Points[i].ID == kp {
				point = &mod.Points[i]
			}
		}
	}
	require.NotNil(t, point)
	codes := map[string]service.MatrixSkill{}
	for _, sk := range point.Skills {
		codes[sk.Code] = sk
	}
	require.Equal(t, 2, codes["reply"].Attempts)
	require.Equal(t, "not_started", point.Status)
}
