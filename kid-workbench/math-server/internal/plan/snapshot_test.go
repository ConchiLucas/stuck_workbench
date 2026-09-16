package plan_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/math-server/internal/plan"
)

func TestBuildSnapshotNormalizesArithmeticVisuals(t *testing.T) {
	base := plan.Candidate{
		QuestionID: 42, KpID: 10, ModuleCode: "add10", Code: "calc", Stem: "2 + 5 = ?",
		Options: `[{"label":"5"},{"label":"6"},{"label":"7"},{"label":"8"}]`,
		Answer:  `{"index":2}`, Visual: `{"kind":"add","a":2,"b":5,"emoji":"🍎"}`,
		MediaURL: "math/questions/42.mp3",
	}
	calc, err := plan.BuildSnapshot(base)
	require.NoError(t, err)
	require.Equal(t, plan.Visual{Kind: "equation", A: 2, B: 5, Operator: "+"}, calc.Visual)

	base.QuestionID, base.Code, base.Stem = 43, "story", "一共有几个？"
	base.MediaURL = "math/questions/43.mp3"
	story, err := plan.BuildSnapshot(base)
	require.NoError(t, err)
	require.Equal(t, plan.Visual{Kind: "add", LeftCount: 2, RightCount: 5, Object: "apple"}, story.Visual)
}

func TestBuildSnapshotNormalizesShapeVisuals(t *testing.T) {
	find, err := plan.BuildSnapshot(plan.Candidate{
		QuestionID: 50, KpID: 12, ModuleCode: "shape", Code: "find", Stem: "点出圆形",
		Options: `[{"shape":"circle"},{"shape":"square"},{"shape":"triangle"},{"shape":"star"}]`,
		Answer:  `{"index":0}`, Visual: `{}`, MediaURL: "math/questions/50.mp3",
	})
	require.NoError(t, err)
	require.Equal(t, "none", find.Visual.Kind)

	name, err := plan.BuildSnapshot(plan.Candidate{
		QuestionID: 51, KpID: 12, ModuleCode: "shape", Code: "name", Stem: "这是什么形状？",
		Options: `[{"label":"圆形"},{"label":"正方形"},{"label":"三角形"},{"label":"五角星"}]`,
		Answer:  `{"index":0}`, Visual: `{"kind":"shape","text":"circle"}`, MediaURL: "math/questions/51.mp3",
	})
	require.NoError(t, err)
	require.Equal(t, plan.Visual{Kind: "shape", Shape: "circle"}, name.Visual)
}

func TestPublicQuestionDoesNotLeakAnswerOrObjectKey(t *testing.T) {
	snapshot, err := plan.BuildSnapshot(plan.Candidate{
		QuestionID: 42, KpID: 10, ModuleCode: "add10", Code: "calc", Stem: "2 + 5 = ?",
		Options: `[{"label":"5"},{"label":"6"},{"label":"7"},{"label":"8"}]`,
		Answer:  `{"index":2}`, Visual: `{"kind":"add","a":2,"b":5,"emoji":"🍎"}`,
		MediaURL: "math/questions/42.mp3",
	})
	require.NoError(t, err)
	raw, err := json.Marshal(plan.PublicQuestion(snapshot, "/audio.mp3"))
	require.NoError(t, err)
	require.NotContains(t, string(raw), "answerIndex")
	require.NotContains(t, string(raw), "audioObjectKey")
	require.NotContains(t, string(raw), "🍎")
}

func TestBuildSnapshotRejectsInvalidContract(t *testing.T) {
	base := plan.Candidate{QuestionID: 42, ModuleCode: "add10", Code: "calc", Stem: "x", Options: `[{"label":"1"}]`, Answer: `{"index":0}`, Visual: `{}`, MediaURL: "math/questions/42.mp3"}
	_, err := plan.BuildSnapshot(base)
	require.Error(t, err)
}
