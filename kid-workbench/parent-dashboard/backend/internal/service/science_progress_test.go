package service_test

import (
	"testing"
	"time"

	"github.com/conchi/study-learning/sciencecontent"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
)

func TestScienceHistoryUsesLinkedPlanItemNotLatest(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='science'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "choice", Type: "practice", Stem: "选", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	first := `{"schema":1,"kind":"choice","skillCode":"choice","responseKind":"choice","selected":"cat","example":{"kind":"choice","prompt":"哪种动物的脚掌最适合在水里游泳？","options":[{"id":"cat","label":"猫"},{"id":"duck","label":"鸭子"},{"id":"rabbit","label":"兔子"}],"answerId":"duck"}}`
	second := `{"schema":1,"kind":"choice","skillCode":"choice","responseKind":"choice","selected":"duck","example":{"kind":"choice","prompt":"哪种动物的脚掌最适合在水里游泳？","options":[{"id":"rabbit","label":"兔子"},{"id":"duck","label":"鸭子"},{"id":"cat","label":"猫"}],"answerId":"duck"}}`
	updated := `{"schema":1,"kind":"choice","skillCode":"choice","responseKind":"choice","selected":"duck","example":{"kind":"choice","prompt":"哪种动物的脚掌最适合在水里游泳？","options":[{"id":"rabbit","label":"兔子"},{"id":"duck","label":"鸭子"},{"id":"cat","label":"猫"}],"answerId":"duck","imageUrl":"/api/v1/science/task-media/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.png"}}`
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	p1 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-10", SeqNo: 191, SubjectCode: "science", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	p2 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-11", SeqNo: 191, SubjectCode: "science", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now.Add(24 * time.Hour)}
	require.NoError(t, d.Create(&p1).Error)
	require.NoError(t, d.Create(&p2).Error)
	item1 := model.PlanItem{PlanID: p1.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: "cat", QuestionSnapshot: first}
	item2 := model.PlanItem{PlanID: p2.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 1, CostMs: 800, Picks: "duck", QuestionSnapshot: second}
	require.NoError(t, d.Create(&item1).Error)
	require.NoError(t, d.Create(&item2).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "sc-link-1", PlanItemID: &item1.ID, Selected: "cat", CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "sc-link-2", PlanItemID: &item2.ID, Selected: "duck", CreatedAt: now.Add(24 * time.Hour)}).Error)
	require.NoError(t, d.Model(&item2).Update("question_snapshot", updated).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "sc-unlinked", CreatedAt: now.Add(48 * time.Hour)}).Error)

	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 3)
	wrong := detail.History[0]
	require.NotNil(t, wrong.ScienceReview)
	require.NotNil(t, wrong.ScienceReview.Example)
	require.Equal(t, "cat", wrong.ScienceReview.Selected)
	require.Equal(t, []string{"cat", "duck", "rabbit"}, scienceOptionIDs(wrong.ScienceReview.Example.Options))
	later := detail.History[1]
	require.Equal(t, "duck", later.ScienceReview.Selected)
	require.Equal(t, []string{"rabbit", "duck", "cat"}, scienceOptionIDs(later.ScienceReview.Example.Options))
	require.Contains(t, later.ScienceReview.Example.ImageURL, "fff")
	unlinked := detail.History[2]
	require.NotNil(t, unlinked.ScienceReview)
	require.Contains(t, unlinked.ScienceReview.UnavailableReason, "无法还原")
	require.Nil(t, unlinked.ScienceReview.Example)
}

func TestScienceHistorySamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='science'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "match", Type: "practice", Stem: "连", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	snap := `{"schema":1,"kind":"match","skillCode":"match","responseKind":"match","example":{"kind":"match","prompt":"把器官和它们的本领连在一起。","matchSources":[{"id":"ear","label":"耳朵"},{"id":"eye","label":"眼睛"},{"id":"nose","label":"鼻子"}],"matchTargets":[{"id":"see","label":"看"},{"id":"hear","label":"听"},{"id":"smell","label":"闻"}],"matchAnswers":{"eye":"see","ear":"hear","nose":"smell"}}}`
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-14", SeqNo: 192, SubjectCode: "science", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 2, CostMs: 1600, QuestionSnapshot: snap}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	wrongSel := `{"kind":"match","pairs":{"eye":"hear","ear":"see","nose":"smell"}}`
	rightSel := `{"kind":"match","pairs":{"eye":"see","ear":"hear","nose":"smell"}}`
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "sc-same-1", PlanItemID: &item.ID, Selected: wrongSel, CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "sc-same-2", PlanItemID: &item.ID, Selected: rightSel, CreatedAt: now.Add(time.Minute)}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	var wrong, right *service.HistoryItem
	for i := range detail.History {
		if detail.History[i].ScienceReview == nil {
			continue
		}
		if !detail.History[i].IsCorrect {
			wrong = &detail.History[i]
		}
		if detail.History[i].IsCorrect {
			right = &detail.History[i]
		}
	}
	require.NotNil(t, wrong)
	require.NotNil(t, right)
	require.Equal(t, sciencecontent.DecodeInput(wrongSel, "match").Pairs, sciencecontent.DecodeInput(wrong.ScienceReview.Selected, "match").Pairs)
	require.Equal(t, sciencecontent.DecodeInput(rightSel, "match").Pairs, sciencecontent.DecodeInput(right.ScienceReview.Selected, "match").Pairs)
	require.NotEmpty(t, wrong.ScienceReview.Facts)
	require.Contains(t, stringsJoin(wrong.ScienceReview.Facts), "眼睛")
}

func scienceOptionIDs(opts []sciencecontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}

func stringsJoin(items []string) string {
	out := ""
	for _, item := range items {
		out += item
	}
	return out
}
