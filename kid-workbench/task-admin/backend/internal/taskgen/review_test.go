package taskgen

import (
	"context"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReviewUsesReceiptsAndDeduplicates(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	task, e := s.Create(ctx, "学习", testSpec())
	require.NoError(t, e)
	task, e = s.Generate(ctx, task.ID, 1, "base", 0)
	require.NoError(t, e)
	require.NoError(t, s.DB.Exec(`CREATE TABLE study_plans(id INTEGER PRIMARY KEY,child_id INTEGER,source_question_task_id INTEGER,status TEXT,completed_at DATETIME);CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,child_id INTEGER,plan_id INTEGER,question_version_id INTEGER,kp_id INTEGER,question_type TEXT,selected_option_id TEXT,is_correct BOOLEAN,created_at DATETIME);`).Error)
	require.NoError(t, s.DB.Exec(`INSERT INTO study_plans VALUES(1,1,?,'done',CURRENT_TIMESTAMP)`, task.ID).Error)
	v := task.Items[0]
	require.NoError(t, s.DB.Exec(`INSERT INTO question_attempt_receipts VALUES(1,1,1,?,?,?,'kp:8',false,CURRENT_TIMESTAMP)`, v.ID, v.KpID, v.QuestionType).Error)
	in := ReviewInput{ChildID: 1, SourcePlanID: 1, Mode: "mixed", TargetCount: 2}
	review, e := s.CreateReview(ctx, task.ID, in)
	require.NoError(t, e)
	require.Equal(t, "review", review.Kind)
	require.Len(t, review.Items, 2)
	require.NotEqual(t, review.Items[0].Fingerprint, review.Items[1].Fingerprint)
	again, e := s.CreateReview(ctx, task.ID, in)
	require.NoError(t, e)
	require.Equal(t, review.ID, again.ID)
	renamed, e := s.Update(ctx, review.ID, review.RowVersion, "复习新标题", review.Spec)
	require.NoError(t, e)
	require.Equal(t, review.ActiveRevisionID, renamed.ActiveRevisionID)
	require.Len(t, renamed.Items, 2)
	changed := renamed.Spec
	changed.TargetCount = 1
	changed.TypeCounts = map[string]int{renamed.Items[0].QuestionType: 1}
	_, e = s.Update(ctx, renamed.ID, renamed.RowVersion, "不应修改", changed)
	require.Error(t, e)
	reordered, e := s.Reorder(ctx, renamed.ID, renamed.RowVersion, []int{2, 1})
	require.NoError(t, e)
	for i, oldIndex := range []int{1, 0} {
		require.NotNil(t, reordered.Items[i].SourceQuestionVersionID)
		require.Equal(t, renamed.Items[oldIndex].SourceQuestionVersionID, reordered.Items[i].SourceQuestionVersionID)
	}
	in.ChildID = 2
	_, e = s.CreateReview(ctx, task.ID, in)
	require.Error(t, e)
}

type broadReviewMaterials struct{ fakeMaterials }

func (m broadReviewMaterials) List(ctx context.Context, module string) ([]generation.Material, error) {
	base, e := m.fakeMaterials.List(ctx, module)
	if e != nil {
		return nil, e
	}
	rows := []generation.Material{}
	for i := 0; i < 81; i++ {
		row := base[0]
		row.KpID = int64(100 + i)
		row.Text = fmt.Sprintf("其他字%d", i)
		row.ModuleCode = "other"
		row.RevisionID = fmt.Sprintf("other%d", i)
		row.Sense = &generation.MediaRef{RevisionID: row.RevisionID, Kind: "sense", SHA256: row.RevisionID}
		rows = append(rows, row)
	}
	return append(rows, base...), nil
}

func TestReviewBroadPoolKeepsSourceTarget(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	spec := testSpec()
	spec.DistractorScope = "subject"
	task, e := s.Create(ctx, "来源", spec)
	require.NoError(t, e)
	task, e = s.Generate(ctx, task.ID, 1, "base", 0)
	require.NoError(t, e)
	require.NoError(t, s.DB.Exec(`CREATE TABLE study_plans(id INTEGER PRIMARY KEY,child_id INTEGER,source_question_task_id INTEGER,status TEXT,completed_at DATETIME);CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,child_id INTEGER,plan_id INTEGER,question_version_id INTEGER,kp_id INTEGER,question_type TEXT,selected_option_id TEXT,is_correct BOOLEAN,created_at DATETIME);`).Error)
	require.NoError(t, s.DB.Exec(`INSERT INTO study_plans VALUES(1,1,?,'done',CURRENT_TIMESTAMP)`, task.ID).Error)
	v := task.Items[0]
	require.NoError(t, s.DB.Exec(`INSERT INTO question_attempt_receipts VALUES(1,1,1,?,?,?,'kp:8',false,CURRENT_TIMESTAMP)`, v.ID, v.KpID, v.QuestionType).Error)
	s.Materials = broadReviewMaterials{}
	preview, e := s.PreviewReview(ctx, task.ID, ReviewInput{ChildID: 1, SourcePlanID: 1, TargetCount: 2, Mode: "mixed"})
	require.NoError(t, e)
	require.Equal(t, 2, preview.Available)
	require.Equal(t, v.KpID, preview.Questions[1].KpID)
}

func TestWritingReviewIncludesHintedAndDoesNotInventVariants(t *testing.T) {
	s := setup(t)
	s.Materials = writingMaterials{}
	ctx := context.Background()
	spec := testSpec()
	spec.TargetCount = 1
	spec.Scope.KpIDs = []int64{1}
	spec.TypeCounts = map[string]int{"write_char": 1}
	task, e := s.Create(ctx, "听写", spec)
	require.NoError(t, e)
	task, e = s.Generate(ctx, task.ID, 1, "writing", 0)
	require.NoError(t, e)
	v := task.Items[0]
	require.NoError(t, s.DB.Exec(`CREATE TABLE study_plans(id INTEGER PRIMARY KEY,child_id INTEGER,source_question_task_id INTEGER,status TEXT,completed_at DATETIME);CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,child_id INTEGER,plan_id INTEGER,question_version_id INTEGER,kp_id INTEGER,question_type TEXT,selected_option_id TEXT,is_correct BOOLEAN,created_at DATETIME,response_kind TEXT,evaluation_json TEXT);`).Error)
	require.NoError(t, s.DB.Exec(`INSERT INTO study_plans VALUES(1,1,?,'done',CURRENT_TIMESTAMP)`, task.ID).Error)
	require.NoError(t, s.DB.Exec(`INSERT INTO question_attempt_receipts VALUES(1,1,1,?,?,?,NULL,true,CURRENT_TIMESTAMP,'handwriting','{"assistance":"hinted"}')`, v.ID, v.KpID, v.QuestionType).Error)
	review, e := s.CreateReview(ctx, task.ID, ReviewInput{ChildID: 1, SourcePlanID: 1, Mode: "original_only"})
	require.NoError(t, e)
	require.Len(t, review.Items, 1)
	p, e := s.PreviewReview(ctx, task.ID, ReviewInput{ChildID: 1, SourcePlanID: 1, Mode: "mixed", TargetCount: 2})
	require.NoError(t, e)
	require.Equal(t, 1, p.Available)
}
