package taskgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/handwriting"
	"github.com/conchi/study-learning/literacycontract"
	"github.com/conchi/study-task-admin/internal/generation"
	"io"
	"net/http"
	"net/url"
)

type PreviewResponse struct {
	Kind             string                `json:"kind"`
	SelectedOptionID string                `json:"selectedOptionId,omitempty"`
	Strokes          [][]handwriting.Point `json:"strokes,omitempty"`
	HintsUsed        int                   `json:"hintsUsed,omitempty"`
}
type PreviewResult struct {
	Correct        bool                          `json:"correct"`
	CanRetry       bool                          `json:"canRetry"`
	AnswerOptionID string                        `json:"answerOptionId,omitempty"`
	Evaluation     *handwriting.EvaluationResult `json:"evaluation,omitempty"`
}

func (s *Service) BuildPreview(ctx context.Context, kp int64, typ string) (generation.Snapshot, error) {
	if kp <= 0 || !literacycontract.Supports(typ) {
		return generation.Snapshot{}, fault(400, "invalid_preview", "请选择汉字和题型")
	}
	if s.Materials == nil {
		return generation.Snapshot{}, fault(503, "materials_unavailable", "素材服务未配置")
	}
	rows, e := s.Materials.List(ctx, "")
	if e != nil {
		return generation.Snapshot{}, e
	}
	var target generation.Material
	for _, m := range rows {
		if m.KpID == kp {
			target = m
			break
		}
	}
	if target.KpID == 0 {
		return generation.Snapshot{}, fault(404, "material_not_found", "找不到素材")
	}
	spec := generation.Spec{SubjectCode: "literacy", Scope: generation.Scope{ModuleCodes: []string{target.ModuleCode}, KpIDs: []int64{kp}}, TargetCount: 1, TypeCounts: map[string]int{typ: 1}, DistractorScope: "module"}
	group := []generation.Material{}
	for _, m := range rows {
		if m.ModuleCode == target.ModuleCode {
			group = append(group, m)
		}
	}
	usable := usableMaterials(group, spec)
	if len(usable) > 80 {
		return generation.Snapshot{}, fault(422, "material_limit", "预览素材超出限制")
	}
	frozen, e := s.Materials.Freeze(ctx, usable)
	if e != nil {
		return generation.Snapshot{}, e
	}
	qs, e := generation.Generate(spec, frozen, kp)
	if e != nil {
		return generation.Snapshot{}, fault(422, "preview_unavailable", e.Error())
	}
	if _, e := s.previewMedia(ctx, qs[0]); e != nil {
		return generation.Snapshot{}, e
	}
	return qs[0], nil
}
func (s *Service) EvaluatePreview(ctx context.Context, q generation.Snapshot, in PreviewResponse) (PreviewResult, error) {
	out := PreviewResult{CanRetry: true}
	if e := generation.ValidateSnapshot(q); e != nil {
		return out, fault(422, "invalid_snapshot", e.Error())
	}
	if !literacycontract.ValidateResponse(q.QuestionType, in.Kind) {
		return out, fault(400, "response_mismatch", "作答方式与题型不一致")
	}
	media, e := s.previewMedia(ctx, q)
	if e != nil {
		return out, e
	}
	if in.Kind == "choice" {
		found := false
		for _, o := range q.Options {
			if o.ID == in.SelectedOptionID {
				found = true
			}
		}
		if !found {
			return out, fault(400, "invalid_option", "选项不存在")
		}
		out.Correct = in.SelectedOptionID == q.AnswerOptionID
		out.AnswerOptionID = q.AnswerOptionID
		return out, nil
	}
	if in.HintsUsed < 0 || in.HintsUsed > 100 {
		return out, fault(400, "invalid_hints", "提示次数无效")
	}
	if e := handwriting.ValidateStrokes(in.Strokes); e != nil {
		return out, fault(400, "invalid_strokes", e.Error())
	}
	raw := media[q.WritingTemplate.RevisionID+":"+q.WritingTemplate.Kind]
	var template handwriting.Template
	if e = json.Unmarshal(raw, &template); e != nil || template.Character != q.TargetText {
		return out, fault(503, "template_mismatch", "书写模板与题目不符")
	}
	result, e := handwriting.Evaluate(in.Strokes, template, q.EvaluationPolicyVersion)
	if e != nil {
		return out, fault(503, "evaluation_unavailable", e.Error())
	}
	if in.HintsUsed > 0 {
		result.Assistance = "hinted"
	}
	out.Correct = result.Outcome == "passed"
	out.Evaluation = &result
	return out, nil
}
func (m *MaterialClient) ReadMedia(ctx context.Context, ref *generation.MediaRef) ([]byte, error) {
	if ref == nil {
		return nil, fmt.Errorf("missing media")
	}
	switch ref.Kind {
	case "glyph", "sense", "speech", "writing_template":
	default:
		return nil, fmt.Errorf("invalid media kind")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", m.Base+"/api/v1/material-revisions/"+url.PathEscape(ref.RevisionID)+"/media/"+ref.Kind, nil)
	if e != nil {
		return nil, e
	}
	res, e := m.Client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	limit := int64(20 << 20)
	if ref.Kind == "writing_template" {
		limit = 2 << 20
	}
	raw, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if e != nil {
		return nil, e
	}
	if res.StatusCode != 200 || len(raw) == 0 || int64(len(raw)) > limit {
		return nil, fmt.Errorf("media unavailable")
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != ref.SHA256 {
		return nil, fmt.Errorf("media hash mismatch")
	}
	return raw, nil
}

// Preview verifies the same immutable bytes on construction and submission;
// no question task or learning record is created by this read-only check.
func (s *Service) previewMedia(ctx context.Context, q generation.Snapshot) (map[string][]byte, error) {
	loader, ok := s.Materials.(interface {
		ReadMedia(context.Context, *generation.MediaRef) ([]byte, error)
	})
	if !ok {
		return nil, fault(503, "materials_unavailable", "预览素材不可读取")
	}
	refs := []*generation.MediaRef{q.Stem.Image, q.Stem.Audio, q.WritingTemplate}
	for _, o := range q.Options {
		refs = append(refs, o.Image, o.Audio)
	}
	seen := map[string]string{}
	out := map[string][]byte{}
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		key := ref.RevisionID + ":" + ref.Kind
		if expected, exists := seen[key]; exists {
			if expected != ref.SHA256 {
				return nil, fault(422, "invalid_snapshot", "同一素材存在冲突摘要")
			}
			continue
		}
		seen[key] = ref.SHA256
		b, e := loader.ReadMedia(ctx, ref)
		if e != nil {
			return nil, fault(503, "materials_unavailable", e.Error())
		}
		sum := sha256.Sum256(b)
		if len(b) == 0 || hex.EncodeToString(sum[:]) != ref.SHA256 {
			return nil, fault(503, "materials_unavailable", "预览素材摘要不匹配")
		}
		out[key] = b
	}
	return out, nil
}
