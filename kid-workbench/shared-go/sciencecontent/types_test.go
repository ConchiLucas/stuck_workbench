package sciencecontent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAndShuffleMatchKeepsIDs(t *testing.T) {
	spec := FilterKind(KindMatch)[0]
	example, err := ExampleFromSpec(spec, nil)
	require.NoError(t, err)
	require.NoError(t, Validate(example))
	shuffled := ShuffleExample(example, 42)
	require.Equal(t, example.MatchAnswers["frog"], shuffled.MatchAnswers["frog"])
	require.True(t, IsCorrect(shuffled, AnswerInput{Kind: KindMatch, Pairs: map[string]string{"frog": "forest", "bear": "ice", "camel": "desert"}}))
	wrong := AnswerInput{Kind: KindMatch, Pairs: map[string]string{"frog": "desert", "bear": "ice", "camel": "forest"}}
	require.False(t, IsCorrect(example, wrong))
	facts := ErrorFacts(example, wrong)
	require.NotEmpty(t, facts)
	require.Contains(t, strings.Join(facts, "\n"), "青蛙")
}

func TestSequenceUsesStableIDs(t *testing.T) {
	spec := FilterKind(KindSequence)[0]
	example, err := ExampleFromSpec(spec, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"seed", "sprout", "seedling"}, example.CorrectSequence)
	require.True(t, IsCorrect(example, AnswerInput{Kind: KindSequence, Sequence: []string{"seed", "sprout", "seedling"}}))
	require.False(t, IsCorrect(example, AnswerInput{Kind: KindSequence, Sequence: []string{"sprout", "seed", "seedling"}}))
}

func TestLabelRelativeCoords(t *testing.T) {
	spec := FilterKind(KindLabel)[0]
	example, err := ExampleFromSpec(spec, nil)
	require.NoError(t, err)
	require.Equal(t, DiagramPlant, example.DiagramKey)
	require.Equal(t, 1, example.DiagramVersion)
	for _, target := range example.LabelTargets {
		require.GreaterOrEqual(t, target.X, 0.0)
		require.LessOrEqual(t, target.X, 1.0)
	}
	require.True(t, IsCorrect(example, AnswerInput{Kind: KindLabel, Labels: map[string]string{"root": "root"}}))
	require.False(t, IsCorrect(example, AnswerInput{Kind: KindLabel, Labels: map[string]string{"flower": "root"}}))
}

func TestHistoryRoundTrip(t *testing.T) {
	spec := FilterKind(KindChoice)[0]
	example, err := ExampleFromSpec(spec, nil)
	require.NoError(t, err)
	raw, err := HistoryBytes(HistorySnapshot{Kind: KindChoice, Example: example, Input: AnswerInput{Kind: KindChoice, SelectedID: "cat"}})
	require.NoError(t, err)
	converted, err := PlanExampleFromSnapshot(string(raw), "")
	require.NoError(t, err)
	require.Equal(t, "cat", converted.Input.SelectedID)
	require.False(t, IsCorrect(converted.Example, converted.Input))
}

func TestLegacyRecognizeMapsToChoice(t *testing.T) {
	snap, err := SnapshotFromLiveQuestion(LiveQuestion{
		Code: "recognize", Stem: "冰变成水，通常是因为什么？",
		Options: `[{"label":"变热了"},{"label":"变冷了"}]`, Answer: `{"index":0}`,
	})
	require.NoError(t, err)
	require.Equal(t, KindChoice, snap.Kind)
	require.Equal(t, "label:变热了", snap.Example.AnswerID)
}

func TestJSONTagsStayCamelCase(t *testing.T) {
	example, err := ExampleFromSpec(FilterKind(KindMatch)[0], nil)
	require.NoError(t, err)
	raw, err := json.Marshal(example)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"matchAnswers"`)
	require.Contains(t, string(raw), `"matchSources"`)
}

func TestUnansweredSnapshotOmitsSelected(t *testing.T) {
	example, err := ExampleFromSpec(FilterKind(KindChoice)[0], nil)
	require.NoError(t, err)
	raw, err := HistoryBytes(HistorySnapshot{Kind: KindChoice, Example: example})
	require.NoError(t, err)
	var snap HistorySnapshot
	require.NoError(t, json.Unmarshal(raw, &snap))
	require.Empty(t, snap.Selected)
	require.Empty(t, EncodeInput(AnswerInput{Kind: KindMatch}))
	require.Equal(t, "duck", EncodeInput(AnswerInput{Kind: KindChoice, SelectedID: "duck"}))
}
