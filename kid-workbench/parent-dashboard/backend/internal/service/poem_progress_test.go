package service_test

import (
	"testing"
	"time"

	"github.com/conchi/study-learning/poemcontent"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPoemHistoryUsesLinkedPlanItemNotLatest(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='poem'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "fill", Type: "practice", Stem: "缺的字是哪个？", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	first := `{"schema":1,"kind":"fill","skillCode":"fill","responseKind":"fill","selected":"char:天","example":{"kind":"fill","prompt":"缺的字是哪个？","line":"锄禾日当□","workId":"pm004","lineId":"pm004:L1","sourceLine":"锄禾日当午","gapIndexes":[4],"options":[{"id":"char:天","label":"天"},{"id":"char:午#4","label":"午"}],"answerId":"char:午#4"}}`
	second := `{"schema":1,"kind":"fill","skillCode":"fill","responseKind":"fill","selected":"char:午#4","example":{"kind":"fill","prompt":"缺的字是哪个？","line":"锄禾日当□","workId":"pm004","lineId":"pm004:L1","sourceLine":"锄禾日当午","gapIndexes":[4],"options":[{"id":"char:午#4","label":"午"},{"id":"char:天","label":"天"}],"answerId":"char:午#4"}}`
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	p1 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-10", SeqNo: 291, SubjectCode: "poem", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	p2 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-11", SeqNo: 291, SubjectCode: "poem", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now.Add(24 * time.Hour)}
	require.NoError(t, d.Create(&p1).Error)
	require.NoError(t, d.Create(&p2).Error)
	item1 := model.PlanItem{PlanID: p1.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: "char:天", QuestionSnapshot: first}
	item2 := model.PlanItem{PlanID: p2.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 1, CostMs: 800, Picks: "char:午#4", QuestionSnapshot: second}
	require.NoError(t, d.Create(&item1).Error)
	require.NoError(t, d.Create(&item2).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "pm-link-1", PlanItemID: &item1.ID, Selected: "char:天", CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "pm-link-2", PlanItemID: &item2.ID, Selected: "char:午#4", CreatedAt: now.Add(24 * time.Hour)}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "pm-unlinked", CreatedAt: now.Add(48 * time.Hour)}).Error)

	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 3)
	wrong := detail.History[0]
	require.NotNil(t, wrong.PoemReview)
	require.NotNil(t, wrong.PoemReview.Example)
	require.Equal(t, "char:天", wrong.PoemReview.Selected)
	require.Equal(t, []string{"char:天", "char:午#4"}, poemOptionIDs(wrong.PoemReview.Example.Options))
	later := detail.History[1]
	require.Equal(t, "char:午#4", later.PoemReview.Selected)
	require.Equal(t, []string{"char:午#4", "char:天"}, poemOptionIDs(later.PoemReview.Example.Options))
	unlinked := detail.History[2]
	require.NotNil(t, unlinked.PoemReview)
	require.Contains(t, unlinked.PoemReview.UnavailableReason, "无法还原")
	require.Nil(t, unlinked.PoemReview.Example)
}

func TestPoemHistorySamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='poem'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "recite", Type: "practice", Stem: "按顺序点出这4句", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	snap := `{"schema":1,"kind":"recite","skillCode":"recite","responseKind":"recite","example":{"kind":"recite","prompt":"按顺序点出这4句","line":"床前明月光","workId":"pm001","sequenceItems":[{"id":"pm001:L1","label":"床前明月光"},{"id":"pm001:L2","label":"疑是地上霜"},{"id":"pm001:L3","label":"举头望明月"},{"id":"pm001:L4","label":"低头思故乡"}],"sequenceDisplayOrder":["pm001:L2","pm001:L1","pm001:L4","pm001:L3"],"correctSequence":["pm001:L1","pm001:L2","pm001:L3","pm001:L4"]}}`
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-14", SeqNo: 292, SubjectCode: "poem", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 2, CostMs: 1600, QuestionSnapshot: snap}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	wrongSel := `{"kind":"recite","sequence":["pm001:L2","pm001:L1","pm001:L3","pm001:L4"]}`
	rightSel := `{"kind":"recite","sequence":["pm001:L1","pm001:L2","pm001:L3","pm001:L4"]}`
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "pm-same-1", PlanItemID: &item.ID, Selected: wrongSel, CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "pm-same-2", PlanItemID: &item.ID, Selected: rightSel, CreatedAt: now.Add(time.Minute)}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	var wrong, right *service.HistoryItem
	for i := range detail.History {
		if detail.History[i].PoemReview == nil {
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
	require.Equal(t, poemcontent.DecodeInput(wrongSel, "recite").Sequence, poemcontent.DecodeInput(wrong.PoemReview.Selected, "recite").Sequence)
	require.Equal(t, poemcontent.DecodeInput(rightSel, "recite").Sequence, poemcontent.DecodeInput(right.PoemReview.Selected, "recite").Sequence)
	require.NotEmpty(t, wrong.PoemReview.Facts)
	require.Contains(t, stringsJoin(wrong.PoemReview.Facts), "疑是地上霜")
}

func poemOptionIDs(opts []poemcontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}
