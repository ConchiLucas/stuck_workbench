package service_test

import (
	"testing"
	"time"

	"github.com/conchi/study-learning/chengyucontent"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChengyuHistoryUsesLinkedPlanItemNotLatest(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='chengyu' AND kp.title='一心一意'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "meaning", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	first := `{"schema":1,"kind":"meaning","skillCode":"meaning","responseKind":"choice","selected":"label:心思不专一","example":{"kind":"meaning","speech":"一心一意","speechUrl":"/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"label:集中精神，做事专心","label":"集中精神，做事专心"},{"id":"label:心思不专一","label":"心思不专一"},{"id":"label:慢慢来","label":"慢慢来"},{"id":"label:随便玩玩","label":"随便玩玩"}],"answerId":"label:集中精神，做事专心"}}`
	second := `{"schema":1,"kind":"meaning","skillCode":"meaning","responseKind":"choice","selected":"label:集中精神，做事专心","example":{"kind":"meaning","speech":"一心一意","speechUrl":"/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"label:随便玩玩","label":"随便玩玩"},{"id":"label:集中精神，做事专心","label":"集中精神，做事专心"},{"id":"label:慢慢来","label":"慢慢来"},{"id":"label:心思不专一","label":"心思不专一"}],"answerId":"label:集中精神，做事专心"}}`
	updated := `{"schema":1,"kind":"meaning","skillCode":"meaning","responseKind":"choice","selected":"label:集中精神，做事专心","example":{"kind":"meaning","speech":"一心一意","speechUrl":"/api/v1/chengyu/task-media/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.mp3","options":[{"id":"label:随便玩玩","label":"随便玩玩"},{"id":"label:集中精神，做事专心","label":"集中精神，做事专心"},{"id":"label:慢慢来","label":"慢慢来"},{"id":"label:心思不专一","label":"心思不专一"}],"answerId":"label:集中精神，做事专心"}}`
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	p1 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-10", SeqNo: 91, SubjectCode: "chengyu", Status: "done", TargetCount: 1, DoneCount: 1, CreatedAt: now}
	p2 := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-11", SeqNo: 91, SubjectCode: "chengyu", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now.Add(24 * time.Hour)}
	require.NoError(t, d.Create(&p1).Error)
	require.NoError(t, d.Create(&p2).Error)
	item1 := model.PlanItem{PlanID: p1.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "wrong", Tries: 1, CostMs: 800, Picks: "label:心思不专一", QuestionSnapshot: first}
	item2 := model.PlanItem{PlanID: p2.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 1, CostMs: 800, Picks: "label:集中精神，做事专心", QuestionSnapshot: second}
	require.NoError(t, d.Create(&item1).Error)
	require.NoError(t, d.Create(&item2).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "cy-link-1", PlanItemID: &item1.ID, Selected: "label:心思不专一", CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "cy-link-2", PlanItemID: &item2.ID, Selected: "label:集中精神，做事专心", CreatedAt: now.Add(24 * time.Hour)}).Error)
	require.NoError(t, d.Model(&item2).Update("question_snapshot", updated).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "cy-unlinked", CreatedAt: now.Add(48 * time.Hour)}).Error)

	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 3)
	wrong := detail.History[0]
	require.NotNil(t, wrong.ChengyuReview)
	require.NotNil(t, wrong.ChengyuReview.Example)
	require.Equal(t, "label:心思不专一", wrong.ChengyuReview.Selected)
	require.Equal(t, "心思不专一", wrong.ChengyuReview.Example.Options[1].Label)
	later := detail.History[1]
	require.Equal(t, "label:集中精神，做事专心", later.ChengyuReview.Selected)
	require.Equal(t, []string{"label:随便玩玩", "label:集中精神，做事专心", "label:慢慢来", "label:心思不专一"}, chengyuOptionIDs(later.ChengyuReview.Example.Options))
	require.Contains(t, later.ChengyuReview.UnavailableReason, "无法可靠还原当时读音")
	require.Empty(t, later.ChengyuReview.Example.SpeechURL)
	unlinked := detail.History[2]
	require.NotNil(t, unlinked.ChengyuReview)
	require.Contains(t, unlinked.ChengyuReview.UnavailableReason, "无法还原")
	require.Nil(t, unlinked.ChengyuReview.Example)
}

func TestChengyuHistorySamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='chengyu' AND kp.title='一心一意'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	q := model.Question{KpID: kp, Code: "meaning", Type: "choice", Stem: "听", Options: "[]", Answer: "{}", Visual: "{}", Speech: "{}"}
	require.NoError(t, d.Create(&q).Error)
	snap := `{"schema":1,"kind":"meaning","skillCode":"meaning","example":{"kind":"meaning","speech":"一心一意","speechUrl":"/api/v1/chengyu/items/1/speech.mp3","options":[{"id":"label:集中精神，做事专心","label":"集中精神，做事专心"},{"id":"label:心思不专一","label":"心思不专一"},{"id":"label:慢慢来","label":"慢慢来"},{"id":"label:随便玩玩","label":"随便玩玩"}],"answerId":"label:集中精神，做事专心"}}`
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	p := model.StudyPlan{ChildID: 1, PlanDate: "2026-09-14", SeqNo: 92, SubjectCode: "chengyu", Status: "done", TargetCount: 1, DoneCount: 1, CorrectCount: 1, CreatedAt: now}
	require.NoError(t, d.Create(&p).Error)
	item := model.PlanItem{PlanID: p.ID, Seq: 1, KpID: kp, QuestionID: q.ID, Bucket: "demo", Status: "correct", Tries: 2, CostMs: 1600, Picks: "label:集中精神，做事专心", QuestionSnapshot: snap}
	require.NoError(t, d.Create(&item).Error)
	qid := q.ID
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: false, CostMs: 800, Source: "quiz", ClientID: "cy-same-1", PlanItemID: &item.ID, Selected: "label:心思不专一", CreatedAt: now}).Error)
	require.NoError(t, d.Create(&model.Attempt{ChildID: 1, KpID: kp, QuestionID: &qid, IsCorrect: true, CostMs: 800, Source: "quiz", ClientID: "cy-same-2", PlanItemID: &item.ID, Selected: "label:集中精神，做事专心", CreatedAt: now.Add(time.Minute)}).Error)
	svc := service.NewDashboardService(repo.New(d))
	detail, err := svc.KpDetail(1, kp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(detail.History), 2)
	var wrong, right *service.HistoryItem
	for i := range detail.History {
		if detail.History[i].ChengyuReview == nil {
			continue
		}
		if detail.History[i].ChengyuReview.Selected == "label:心思不专一" {
			wrong = &detail.History[i]
		}
		if detail.History[i].ChengyuReview.Selected == "label:集中精神，做事专心" && detail.History[i].IsCorrect {
			right = &detail.History[i]
		}
	}
	require.NotNil(t, wrong)
	require.NotNil(t, right)
	require.Equal(t, "label:心思不专一", wrong.ChengyuReview.Selected)
	require.Equal(t, "label:集中精神，做事专心", right.ChengyuReview.Selected)
}

func TestChengyuMatrixShowsOneSkillWithoutGrantingMastery(t *testing.T) {
	_, d := newAttemptSvc(t)
	var kp int64
	require.NoError(t, d.Raw(`
		SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id=kp.module_id
		JOIN subjects s ON s.id=m.subject_id
		WHERE s.code='chengyu' AND kp.title='一心一意'
		ORDER BY kp.id LIMIT 1`).Scan(&kp).Error)
	require.NotZero(t, kp)
	require.NoError(t, d.Create(&model.MasterySkill{
		ChildID: 1, KpID: kp, SkillCode: mastery.SkillChengyuMeaning,
		Status: string(mastery.StatusMastered), Attempts: 2, Correct: 1, Ease: 2.5,
	}).Error)
	matrix, err := service.NewDashboardService(repo.New(d)).Matrix(1, "chengyu")
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
	require.Equal(t, 2, codes["meaning"].Attempts)
	require.Equal(t, "learning", point.Status)
}

func chengyuOptionIDs(opts []chengyucontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}
