package practice_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/conchi/literacy-server/internal/plan"
	"github.com/conchi/literacy-server/internal/practice"
	"github.com/conchi/study-learning/handwriting"
	"github.com/conchi/study-learning/mastery"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestHandwritingReceiptTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, q := range practiceSchema() {
		require.NoError(t, db.Exec(q).Error)
	}
	require.NoError(t, db.Exec("ALTER TABLE plan_items ADD COLUMN question_version_id INTEGER").Error)
	_, err = plan.NewService(db).Create(context.Background(), 1, 1)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&practice.Receipt{}))
	require.NoError(t, db.Exec("CREATE TABLE literacy_writing_template_cache(revision_id TEXT PRIMARY KEY,sha256 TEXT,template_json TEXT)").Error)
	template := `{"schemaVersion":1,"character":"一","strokes":["M 100 500 L 900 500"],"medians":[[[100,500],[900,500]]]}`
	require.NoError(t, db.Exec("INSERT INTO literacy_writing_template_cache VALUES('rev','hash',?)", template).Error)
	snap := `{"schemaVersion":2,"subjectCode":"literacy","kpId":100,"targetText":"一","questionType":"write_char","skillCode":"write_char","interaction":"handwriting","responseSchemaVersion":2,"evaluationPolicyVersion":"ink-match-v1","stem":{"audio":{"revisionId":"rev","kind":"speech"}},"writingTemplate":{"revisionId":"rev","kind":"writing_template","sha256":"hash"}}`
	require.NoError(t, db.Exec("UPDATE plan_items SET question_version_id=1,option_order='',question_snapshot=? WHERE id=1", snap).Error)
	safeDetail, e := plan.NewService(db).Get(context.Background(), 1, 1)
	require.NoError(t, e)
	safeJSON, e := json.Marshal(safeDetail)
	require.NoError(t, e)
	require.NotContains(t, string(safeJSON), "targetText")
	require.NotContains(t, string(safeJSON), "writingTemplate")
	require.NotContains(t, string(safeJSON), "一")
	require.Equal(t, "handwriting", safeDetail.Items[0].Question.Type)
	svc := practice.NewService(db, mastery.Config{BaseMasterStreak: 2, MinAccuracy: .8, EaseMin: 1.3, EaseMax: 2.8, MaxIntervalDays: 60})
	ctx := context.Background()
	input := practice.AnswerInput{ClientID: "ink1", Response: &practice.ResponsePayload{Kind: "handwriting", Strokes: [][]handwriting.Point{{{X: .1, Y: .5, T: 0}, {X: .9, Y: .5, T: 1}}}, HintsUsed: 1}}
	first, err := svc.Answer(ctx, 1, 1, 1, input)
	require.NoError(t, err)
	require.True(t, first.Correct)
	require.Equal(t, "hinted", first.Evaluation.Assistance)
	replay, err := svc.Answer(ctx, 1, 1, 1, input)
	require.NoError(t, err)
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(replay)
	require.JSONEq(t, string(a), string(b))
	input.Response.HintsUsed = 0
	_, err = svc.Answer(ctx, 1, 1, 1, input)
	require.ErrorIs(t, err, practice.ErrIdempotencyConflict)
	var n int64
	require.NoError(t, db.Table("mastery_skills").Where("streak > 0").Count(&n).Error)
	require.Zero(t, n)
	// A receipt failure must roll back the attempt and counters.
	require.NoError(t, db.Exec("UPDATE plan_items SET status='pending',tries=0 WHERE id=1").Error)
	require.NoError(t, db.Exec("CREATE TRIGGER fail_ink_receipt BEFORE INSERT ON question_attempt_receipts BEGIN SELECT RAISE(ABORT,'receipt failure'); END").Error)
	input.ClientID = "rollback"
	_, err = svc.Answer(ctx, 1, 1, 1, input)
	require.Error(t, err)
	require.NoError(t, db.Table("attempts").Where("client_id='rollback'").Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, db.Exec("DROP TRIGGER fail_ink_receipt").Error)
	for i := 1; i <= 3; i++ {
		input.ClientID = fmt.Sprint("blank", i)
		input.Response.Strokes = nil
		r, err := svc.Answer(ctx, 1, 1, 1, input)
		require.NoError(t, err)
		require.False(t, r.Correct)
		require.Equal(t, i < 3, r.CanRetry)
	}
	input.ClientID = "fourth"
	_, err = svc.Answer(ctx, 1, 1, 1, input)
	require.ErrorIs(t, err, practice.ErrItemCompleted)
}
