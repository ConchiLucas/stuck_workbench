package service_test

import (
	"testing"
	"time"

	"github.com/conchi/study-learning/englishcontent"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
)

func TestEnglishHistoryUsesLinkedPlanItemNotLatest(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='english' AND LOWER(kp.title)='apple'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "listen", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	first := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"2","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"苹果","picture":"/one.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"4","label":"小鸟","picture":"/four.jpg"}],"answerId":"1"}}`
	second := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"1","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"4","label":"小鸟","picture":"/four.jpg"},{"id":"1","label":"苹果","picture":"/one.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"}],"answerId":"1"}}`
	updated := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"1","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.mp3","options":[{"id":"4","label":"小鸟","picture":"/four.jpg"},{"id":"1","label":"苹果","picture":"/new.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"}],"answerId":"1"}}`
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	p1 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-10", SeqNo: 81, SubjectCode: "english", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	p2 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-11", SeqNo: 81, SubjectCode: "english", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now.Add(24 * time.Hour)}
	require.NoError(t, d.Create(&p1).Error)
	require.NoError(t, d.Create(&p2).Error)
	item1 := model.PlanItem{PlanID: p1.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: "2", QuestionSnapshot: first}
	item2 := model.PlanItem{PlanID: p2.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 1, CostMs: 800, Picks: "1", QuestionSnapshot: second}
	require.NoError(t, d.Create(&item1).Error)
	require.NoError(t, d.Create(&item2).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "en-link-1", PlanItemID: &item1.ID, CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "en-link-2", PlanItemID: &item2.ID, CreatedAt: now.Add(24 * time.Hour)}).Error)
	require.NoError(t, d.Model(&item2).Update("question_snapshot", updated).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "en-unlinked", CreatedAt: now.Add(48 * time.Hour)}).Error)

	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 3)
	wrong := detail.History[0]
	require.NotNil(t, wrong.EnglishReview)
	require.NotNil(t, wrong.EnglishReview.Example)
	require.Equal(t, "2", wrong.EnglishReview.Selected)
	require.Equal(t, "小狗", wrong.EnglishReview.Example.Options[1].Label)
	require.Equal(t, []string{"1", "2", "3", "4"}, englishOptionIDs(wrong.EnglishReview.Example.Options))
	later := detail.History[1]
	require.Equal(t, "1", later.EnglishReview.Selected)
	require.Equal(t, []string{"4", "1", "3", "2"}, englishOptionIDs(later.EnglishReview.Example.Options))
	require.Contains(t, later.EnglishReview.Example.SpeechURL, "fff")
	unlinked := detail.History[2]
	require.NotNil(t, unlinked.EnglishReview)
	require.Contains(t, unlinked.EnglishReview.UnavailableReason, "无法还原")
	require.Nil(t, unlinked.EnglishReview.Example)
}

func TestEnglishHistorySamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='english' AND LOWER(kp.title)='apple'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "listen", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	snap := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/words/20/speech.mp3","options":[{"id":"20","label":"苹果"},{"id":"21","label":"小狗"},{"id":"22","label":"小猫"},{"id":"23","label":"小鸟"}],"answerId":"20"}}`
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-14", SeqNo: 82, SubjectCode: "english", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 2, CostMs: 1600, Picks: `[{"clientId":"a","optionIndex":1},{"clientId":"b","optionIndex":0}]`, QuestionSnapshot: snap}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "en-same-1", PlanItemID: &item.ID, Selected: "21", CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "en-same-2", PlanItemID: &item.ID, Selected: "20", CreatedAt: now.Add(time.Minute)}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 2)
	var wrong, right *service.HistoryItem
	for i := range detail.History {
		if detail.History[i].EnglishReview == nil {
			continue
		}
		if detail.History[i].EnglishReview.Selected == "21" {
			wrong = &detail.History[i]
		}
		if detail.History[i].EnglishReview.Selected == "20" && detail.History[i].IsCorrect {
			right = &detail.History[i]
		}
	}
	require.NotNil(t, wrong)
	require.NotNil(t, right)
	require.Equal(t, "21", wrong.EnglishReview.Selected)
	require.Equal(t, "20", right.EnglishReview.Selected)
	require.Equal(t, wrong.EnglishReview.Example.SpeechURL, right.EnglishReview.Example.SpeechURL)
}

func TestEnglishHistoryRejectsCodeTypeOnlySnapshot(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='english' AND LOWER(kp.title)='apple'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "listen", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	now := time.Date(2026, 9, 14, 11, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-14", SeqNo: 83, SubjectCode: "english", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: "2", QuestionSnapshot: `{"code":"listen","type":"choice"}`}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "en-code-type", PlanItemID: &item.ID, Selected: "2", CreatedAt: now}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	var found *service.HistoryItem
	for i := range detail.History {
		if detail.History[i].AttemptID == 0 {
			continue
		}
		if detail.History[i].EnglishReview != nil && detail.History[i].EnglishReview.UnavailableReason != "" {
			found = &detail.History[i]
		}
	}
	require.NotNil(t, found)
	require.Nil(t, found.EnglishReview.Example)
	require.Contains(t, found.EnglishReview.UnavailableReason, "无法还原")
}

func englishOptionIDs(opts []englishcontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}
