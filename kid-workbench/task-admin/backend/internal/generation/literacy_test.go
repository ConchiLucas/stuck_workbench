package generation

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func fixture() []Material {
	var out []Material
	for i, ch := range []string{"春", "雨", "花", "草", "树", "山", "水", "日"} {
		id := int64(i + 1)
		rev := fmt.Sprintf("%064x", id)
		out = append(out, Material{KpID: id, Text: ch, ModuleCode: "g1", RevisionID: rev, SourceRevision: rev,
			Glyph: &MediaRef{RevisionID: rev, Kind: "glyph", SHA256: rev}, Sense: &MediaRef{RevisionID: rev, Kind: "sense", SHA256: rev}, Speech: &MediaRef{RevisionID: rev, Kind: "speech", SHA256: rev}})
	}
	return out
}
func TestLiteracyExactBudgetAndStableSeed(t *testing.T) {
	spec := Spec{SubjectCode: "literacy", Kind: "practice", Scope: Scope{ModuleCodes: []string{"g1"}, KpIDs: []int64{1, 2, 3, 4}}, TargetCount: 8, TypeCounts: map[string]int{"glyph_sense": 4, "sense_char": 4}, DistractorScope: "module"}
	a, err := Generate(spec, fixture(), 42)
	require.NoError(t, err)
	require.Len(t, a, 8)
	b, err := Generate(spec, fixture(), 42)
	require.NoError(t, err)
	require.Equal(t, a, b)
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, q := range a {
		require.NoError(t, ValidateSnapshot(q))
		counts[q.QuestionType]++
		fp := Fingerprint(q)
		require.False(t, seen[fp])
		seen[fp] = true
	}
	require.Equal(t, spec.TypeCounts, counts)
}
func TestLiteracyRejectsMissingAssetsAndBadBudget(t *testing.T) {
	spec := Spec{SubjectCode: "literacy", Kind: "practice", Scope: Scope{ModuleCodes: []string{"g1"}}, TargetCount: 8, TypeCounts: map[string]int{"glyph_sense": 4, "sense_char": 3}}
	_, err := Generate(spec, fixture(), 42)
	require.Error(t, err)
	spec.TypeCounts["sense_char"] = 4
	_, err = Generate(spec, fixture()[:3], 42)
	require.Error(t, err)
	m := fixture()
	for i := range m {
		m[i].Speech = nil
	}
	_, err = Generate(spec, m, 42)
	require.Error(t, err)
}
func TestFingerprintIgnoresOptionOrder(t *testing.T) {
	spec := Spec{SubjectCode: "literacy", Scope: Scope{ModuleCodes: []string{"g1"}}, TargetCount: 1, TypeCounts: map[string]int{"glyph_sense": 1}}
	qs, err := Generate(spec, fixture(), 7)
	require.NoError(t, err)
	q := qs[0]
	fp := Fingerprint(q)
	q.Options[0], q.Options[1] = q.Options[1], q.Options[0]
	require.Equal(t, fp, Fingerprint(q))
}

func TestLiteracyHundredSeedsKeepQuestionInvariants(t *testing.T) {
	spec := Spec{SubjectCode: "literacy", Scope: Scope{ModuleCodes: []string{"g1"}, KpIDs: []int64{1, 2, 3, 4}}, TargetCount: 8, TypeCounts: map[string]int{"glyph_sense": 4, "sense_char": 4}}
	positions := map[int]bool{}
	for seed := int64(0); seed < 100; seed++ {
		qs, err := Generate(spec, fixture(), seed)
		require.NoError(t, err)
		counts := map[string]int{}
		seen := map[string]bool{}
		for _, q := range qs {
			require.NoError(t, ValidateSnapshot(q))
			counts[q.QuestionType]++
			require.False(t, seen[Fingerprint(q)])
			seen[Fingerprint(q)] = true
			for i, o := range q.Options {
				if o.ID == q.AnswerOptionID {
					positions[i] = true
				}
			}
		}
		require.Equal(t, spec.TypeCounts, counts)
	}
	require.Len(t, positions, 4)
}

func TestMixedWritingAndChoiceExactBudget(t *testing.T) {
	ms := fixture()
	for i := range ms {
		ms[i].WritingTemplate = &MediaRef{RevisionID: ms[i].RevisionID, Kind: "writing_template", SHA256: fmt.Sprintf("%064x", 99)}
	}
	spec := Spec{SubjectCode: "literacy", Scope: Scope{ModuleCodes: []string{"g1"}}, TargetCount: 12, TypeCounts: map[string]int{"glyph_sense": 4, "sense_char": 4, "write_char": 4}}
	qs, e := Generate(spec, ms, 42)
	require.NoError(t, e)
	require.Len(t, qs, 12)
	counts := map[string]int{}
	for _, q := range qs {
		counts[q.QuestionType]++
		require.NoError(t, ValidateSnapshot(q))
		if q.QuestionType == "write_char" {
			require.Empty(t, q.Options)
			require.Empty(t, q.AnswerOptionID)
			require.Empty(t, q.Stem.Text)
			require.NotNil(t, q.WritingTemplate)
		}
	}
	require.Equal(t, spec.TypeCounts, counts)
}
func TestWritingNeedsNeitherSenseNorDistractors(t *testing.T) {
	m := fixture()[0]
	m.Sense = nil
	m.Glyph = nil
	m.WritingTemplate = &MediaRef{RevisionID: m.RevisionID, Kind: "writing_template", SHA256: fmt.Sprintf("%064x", 99)}
	spec := Spec{Scope: Scope{ModuleCodes: []string{"g1"}}, TargetCount: 1, TypeCounts: map[string]int{"write_char": 1}}
	qs, e := Generate(spec, []Material{m}, 7)
	require.NoError(t, e)
	require.Len(t, qs, 1)
	m.WritingTemplate = nil
	_, e = Generate(spec, []Material{m}, 7)
	require.Error(t, e)
}
func TestWritingRejectsInvalidOrUnrelatedMediaRefs(t *testing.T) {
	h := fmt.Sprintf("%064x", 1)
	q := Snapshot{SchemaVersion: 2, SubjectCode: "literacy", KpID: 1, TargetText: "一", QuestionType: "write_char", SkillCode: "write_char", Interaction: "handwriting", ResponseSchemaVersion: 2, EvaluationPolicyVersion: "ink-match-v1", WritingTemplate: &MediaRef{RevisionID: h, Kind: "writing_template", SHA256: h}, Stem: Stem{Audio: &MediaRef{RevisionID: h, Kind: "speech", SHA256: h}}, MaterialRevisionIDs: []string{h}}
	require.NoError(t, ValidateSnapshot(q))
	for _, mutate := range []func(*Snapshot){func(s *Snapshot) { s.Stem.Audio = &MediaRef{RevisionID: h, Kind: "speech"} }, func(s *Snapshot) { s.Stem.Audio = &MediaRef{RevisionID: "bogus", Kind: "speech", SHA256: h} }, func(s *Snapshot) { s.MaterialRevisionIDs = []string{fmt.Sprintf("%064x", 2)} }, func(s *Snapshot) {
		s.WritingTemplate = &MediaRef{RevisionID: h, Kind: "writing_template", SHA256: "wrong"}
	}} {
		bad := q
		mutate(&bad)
		require.Error(t, ValidateSnapshot(bad))
	}
}
