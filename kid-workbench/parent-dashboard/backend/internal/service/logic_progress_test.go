package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
)

func TestLogicHistoryUsesLinkedPlanItemNotLatest(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='logic'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "classify", Type: "practice", Stem: "哪个不属于这一类？", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	first := `{"schema":1,"kind":"classify","prompt":"哪个不属于这一类？","rule":{"type":"odd-one-out","dimension":"kingdom","inGroup":"animal","explain":"动物"},"objects":[{"id":"cat","caption":"猫","glyph":"cat","attrs":{"category":"animal"}},{"id":"car","caption":"汽车","glyph":"car","attrs":{"category":"vehicle"}}],"options":["cat","car"],"answerId":"car"}`
	second := `{"schema":1,"kind":"classify","prompt":"哪个不属于这一类？","rule":{"type":"odd-one-out","dimension":"kingdom","inGroup":"animal","explain":"动物"},"objects":[{"id":"cat","caption":"猫","glyph":"cat","attrs":{"category":"animal"}},{"id":"car","caption":"汽车","glyph":"car","attrs":{"category":"vehicle"}}],"options":["car","cat"],"answerId":"car"}`
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	p1 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-10", SeqNo: 391, SubjectCode: "logic", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	p2 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-11", SeqNo: 391, SubjectCode: "logic", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now.Add(24 * time.Hour)}
	require.NoError(t, d.Create(&p1).Error)
	require.NoError(t, d.Create(&p2).Error)
	item1 := model.PlanItem{PlanID: p1.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: `{"selectedId":"cat"}`, QuestionSnapshot: first}
	item2 := model.PlanItem{PlanID: p2.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 1, CostMs: 800, Picks: `{"selectedId":"car"}`, QuestionSnapshot: second}
	require.NoError(t, d.Create(&item1).Error)
	require.NoError(t, d.Create(&item2).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "lg-link-1", PlanItemID: &item1.ID, Selected: `{"selectedId":"cat"}`, CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "lg-link-2", PlanItemID: &item2.ID, Selected: `{"selectedId":"car"}`, CreatedAt: now.Add(24 * time.Hour)}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "lg-unlinked", CreatedAt: now.Add(48 * time.Hour)}).Error)

	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 3)
	wrong := detail.History[0]
	require.NotNil(t, wrong.LogicReview)
	require.NotNil(t, wrong.LogicReview.Example)
	require.Equal(t, "cat", wrong.LogicReview.Example.Options[0])
	require.Contains(t, wrong.LogicReview.Selected, "cat")
	later := detail.History[1]
	require.Equal(t, "car", later.LogicReview.Example.Options[0])
	unlinked := detail.History[2]
	require.NotNil(t, unlinked.LogicReview)
	require.Contains(t, unlinked.LogicReview.UnavailableReason, "无法还原")
	require.Nil(t, unlinked.LogicReview.Example)
}

func TestLogicHistorySamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='logic'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "order", Type: "practice", Stem: "排", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	snap := `{"schema":1,"kind":"order","prompt":"按数量从小到大点一排","rule":{"type":"order","dimension":"count","direction":"asc","explain":"数量"},"objects":[{"id":"o1","caption":"1","glyph":"one","attrs":{"count":"1"}},{"id":"o2","caption":"2","glyph":"two","attrs":{"count":"2"}},{"id":"o3","caption":"3","glyph":"three","attrs":{"count":"3"}}],"correctSequence":["o1","o2","o3"],"displayOrder":["o2","o3","o1"]}`
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-15", SeqNo: 392, SubjectCode: "logic", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 2, CostMs: 1600, QuestionSnapshot: snap}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	wrongSel := `{"sequence":["o2","o1","o3"],"rejected":[{"id":"o3","atIndex":0}]}`
	rightSel := `{"sequence":["o1","o2","o3"]}`
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "lg-order-1", PlanItemID: &item.ID, Selected: wrongSel, CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "lg-order-2", PlanItemID: &item.ID, Selected: rightSel, CreatedAt: now.Add(time.Minute)}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.Len(t, detail.History, 2)
	require.Contains(t, detail.History[0].LogicReview.Selected, "o2")
	require.Contains(t, detail.History[1].LogicReview.Selected, `"o1"`)
	require.Contains(t, strings.Join(detail.History[0].LogicReview.Facts, "\n"), "误点")
}
