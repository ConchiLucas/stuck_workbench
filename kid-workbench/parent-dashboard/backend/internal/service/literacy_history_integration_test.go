package service_test

import (
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const glyphSnapshot = `{"schemaVersion":2,"subjectCode":"literacy","questionType":"glyph_sense","interaction":"choice","stem":{"text":"山","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"glyph","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},"options":[{"id":"a","text":"山","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},{"id":"b","text":"水","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},{"id":"c","text":"火","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},{"id":"d","text":"木","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}],"answerOptionId":"a"}`

func TestGlyphReviewAndAttemptIsolation(t *testing.T) {
	svc, g := newPlanSvc(t)
	d, e := svc.Today(1)
	require.NoError(t, e)
	item := d.Items[0]
	require.NoError(t, g.Raw(`SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='literacy' LIMIT 1`).Scan(&item.KpID).Error)
	require.NoError(t, g.Table("plan_items").Where("id=?", item.ID).Updates(map[string]any{"kp_id": item.KpID, "question_id": nil, "question_version_id": 99, "question_snapshot": glyphSnapshot, "option_order": "1,0,3,2", "picks": "0,1"}).Error)
	review, e := svc.Review(1, d.Plan.ID)
	require.NoError(t, e)
	require.NotNil(t, review.Items[0].LiteracyReview)
	require.Equal(t, "a", review.Items[0].LiteracyReview.SelectedOptionID)
	// Use the item's knowledge point subject for this isolated fixture.
	for i, child := range []int64{1, 2} {
		a := model.Attempt{ChildID: 1, KpID: item.KpID, IsCorrect: i == 0, Source: "quiz", ClientID: []string{"glyph-good", "glyph-foreign"}[i], CreatedAt: time.Now()}
		require.NoError(t, g.Create(&a).Error)
		require.NoError(t, g.Exec(`INSERT INTO question_attempt_receipts(attempt_id,child_id,client_id,plan_id,plan_item_id,question_version_id,kp_id,skill_code,question_type,selected_option_id,display_index,original_index,is_correct,cost_ms,response_json) VALUES(?,?,?,?,?,?,?,'glyph_sense','glyph_sense','b',0,1,0,1,'{}')`, a.ID, child, a.ClientID, d.Plan.ID, item.ID, 99, item.KpID).Error)
	}
	out, e := service.NewDashboardService(repo.New(g)).KpDetail(1, item.KpID)
	require.NoError(t, e)
	require.Len(t, out.History, 2)
	require.NotNil(t, out.History[0].LiteracyReview)
	require.Equal(t, "b", out.History[0].LiteracyReview.SelectedOptionID)
	require.Empty(t, out.History[1].SelectedOptionID)
	require.Nil(t, out.History[1].LiteracyReview)
	require.NoError(t, g.Table("plan_items").Where("id=?", item.ID).Update("option_order", "0,0,2,3").Error)
	review, e = svc.Review(1, d.Plan.ID)
	require.NoError(t, e)
	require.NotEmpty(t, review.Items[0].LiteracyReview.UnavailableReason)
	require.NoError(t, g.Table("plan_items").Where("id=?", item.ID).Updates(map[string]any{"option_order": "0,1,2,3", "picks": "0,bad"}).Error)
	review, e = svc.Review(1, d.Plan.ID)
	require.NoError(t, e)
	require.NotEmpty(t, review.Items[0].LiteracyReview.UnavailableReason)
	require.NoError(t, g.Table("plan_items").Where("id=?", item.ID).Update("question_snapshot", "{}").Error)
	review, e = svc.Review(1, d.Plan.ID)
	require.NoError(t, e)
	require.NotEmpty(t, review.Items[0].LiteracyReview.UnavailableReason)
	out, e = service.NewDashboardService(repo.New(g)).KpDetail(1, item.KpID)
	require.NoError(t, e)
	require.Len(t, out.History, 2)
	require.NotEmpty(t, out.History[0].LiteracyReview.UnavailableReason)

}
