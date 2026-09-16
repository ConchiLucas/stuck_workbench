package taskgen

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPreviewDoesNotCreateTasksOrRuns(t *testing.T) {
	s := setup(t)
	s.Materials = previewReader{}
	q, e := s.BuildPreview(context.Background(), 1, "glyph_sense")
	require.NoError(t, e)
	out, e := s.EvaluatePreview(context.Background(), q, PreviewResponse{Kind: "choice", SelectedOptionID: q.AnswerOptionID})
	require.NoError(t, e)
	require.True(t, out.Correct)
	var n int64
	require.NoError(t, s.DB.Model(&Task{}).Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, s.DB.Model(&Run{}).Count(&n).Error)
	require.Zero(t, n)
}

type previewReader struct {
	fakeMaterials
	fail bool
}

func (m previewReader) ReadMedia(_ context.Context, ref *generation.MediaRef) ([]byte, error) {
	if m.fail {
		return nil, fmt.Errorf("object missing")
	}
	return []byte(ref.Kind + ref.RevisionID), nil
}
func TestPreviewRejectsUnavailableActualMedia(t *testing.T) {
	s := setup(t)
	s.Materials = previewReader{}
	q, e := s.BuildPreview(context.Background(), 1, "glyph_sense")
	require.NoError(t, e)
	s.Materials = previewReader{fail: true}
	_, e = s.EvaluatePreview(context.Background(), q, PreviewResponse{Kind: "choice", SelectedOptionID: q.AnswerOptionID})
	require.Error(t, e)
	_, e = s.BuildPreview(context.Background(), 1, "glyph_sense")
	require.Error(t, e)
}

func (m previewReader) List(ctx context.Context, module string) ([]generation.Material, error) {
	rows, e := m.fakeMaterials.List(ctx, module)
	for i := range rows {
		for _, ref := range []*generation.MediaRef{rows[i].Glyph, rows[i].Sense, rows[i].Speech} {
			sum := sha256.Sum256([]byte(ref.Kind + ref.RevisionID))
			ref.SHA256 = fmt.Sprintf("%x", sum)
		}
	}
	return rows, e
}

type corruptPreviewReader struct{ previewReader }

func (m corruptPreviewReader) ReadMedia(ctx context.Context, ref *generation.MediaRef) ([]byte, error) {
	return []byte("corrupt"), nil
}
func TestPreviewRejectsHashMismatchAndPreservesLegacyRead(t *testing.T) {
	s := setup(t)
	s.Materials = previewReader{}
	q, e := s.BuildPreview(context.Background(), 1, "glyph_sense")
	require.NoError(t, e)
	q.SchemaVersion = 1
	q.Interaction = ""
	q.ResponseSchemaVersion = 0
	q.Stem.Image = nil
	_, e = s.EvaluatePreview(context.Background(), q, PreviewResponse{Kind: "choice", SelectedOptionID: q.AnswerOptionID})
	require.NoError(t, e)
	s.Materials = corruptPreviewReader{}
	_, e = s.EvaluatePreview(context.Background(), q, PreviewResponse{Kind: "choice", SelectedOptionID: q.AnswerOptionID})
	require.Error(t, e)
}

type missingSpeechReader struct{ previewReader }

func (m missingSpeechReader) ReadMedia(ctx context.Context, ref *generation.MediaRef) ([]byte, error) {
	if ref.Kind == "speech" {
		return nil, fmt.Errorf("speech unavailable")
	}
	return m.previewReader.ReadMedia(ctx, ref)
}
func TestWritingPreviewDoesNotEvaluateWhenSpeechMissing(t *testing.T) {
	s := setup(t)
	s.Materials = missingSpeechReader{}
	h := fmt.Sprintf("%064x", 1)
	q := generation.Snapshot{SchemaVersion: 2, SubjectCode: "literacy", KpID: 1, TargetText: "一", QuestionType: "write_char", SkillCode: "write_char", Interaction: "handwriting", ResponseSchemaVersion: 2, EvaluationPolicyVersion: "ink-match-v1", WritingTemplate: &generation.MediaRef{RevisionID: h, Kind: "writing_template", SHA256: h}, Stem: generation.Stem{Audio: &generation.MediaRef{RevisionID: h, Kind: "speech", SHA256: h}}, MaterialRevisionIDs: []string{h}}
	_, e := s.EvaluatePreview(context.Background(), q, PreviewResponse{Kind: "handwriting"})
	require.Error(t, e)
	require.Contains(t, e.Error(), "speech unavailable")
}
