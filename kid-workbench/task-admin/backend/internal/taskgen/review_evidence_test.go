package taskgen

import (
	"context"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEvidenceReviewNeverPadsIdenticalOriginals(t *testing.T) {
	s := New(nil, nil)
	_, e := s.PrepareEvidenceReview(context.Background(), "g1", 1, []EvidenceReviewTarget{{Key: "1:glyph_sense", KpID: 1, QuestionType: "glyph_sense", Mode: "original_only", Count: 2, Originals: []EvidenceOriginal{{VersionID: 1, Snapshot: generation.Snapshot{KpID: 1, QuestionType: "glyph_sense"}}}}})
	require.Error(t, e)
}
func TestEvidenceReviewReportsUnretainedPreference(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	task, e := s.Create(ctx, "source", testSpec())
	require.NoError(t, e)
	task, e = s.Generate(ctx, task.ID, task.RowVersion, "source", 0)
	require.NoError(t, e)
	q := task.Items[0]
	prepared, e := s.PrepareEvidenceReview(ctx, "g1", 4, []EvidenceReviewTarget{{Key: "target", KpID: q.KpID, QuestionType: q.QuestionType, Mode: "mixed", Count: 2, PreferredDistractorKpIDs: []int64{99999}, Originals: []EvidenceOriginal{{VersionID: q.ID, Snapshot: q.Snapshot}}}})
	require.NoError(t, e)
	require.Len(t, prepared.Warnings, 1)
	require.Equal(t, "preferred_distractor_unavailable", prepared.Warnings[0].Code)
}
