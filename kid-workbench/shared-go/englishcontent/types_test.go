package englishcontent_test

import (
	"testing"

	"github.com/conchi/study-learning/englishcontent"
	"github.com/stretchr/testify/require"
)

func sample() englishcontent.EnglishExample {
	return englishcontent.EnglishExample{
		Kind: "audio-choice", Speech: "apple", SpeechURL: "/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3",
		Options: []englishcontent.Choice{
			{ID: "apple", Label: "苹果", Picture: "/api/v1/english/task-media/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.jpg"},
			{ID: "dog", Label: "小狗", Picture: "/api/v1/english/task-media/cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc.jpg"},
		},
		AnswerID: "apple",
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	raw, err := englishcontent.HistoryBytes(englishcontent.HistorySnapshot{Kind: "audio-choice", Example: sample(), Selected: "dog"})
	require.NoError(t, err)
	got, err := englishcontent.PlanExampleFromSnapshot(string(raw), "")
	require.NoError(t, err)
	require.Equal(t, "dog", got.Selected)
	require.Equal(t, "choice", got.ResponseKind)
	require.Equal(t, "listen", englishcontent.SkillForKind(got.Example.Kind))
}

func TestCardBuilderRequiresPreparedSentenceSpeech(t *testing.T) {
	err := englishcontent.Validate(englishcontent.EnglishExample{
		Kind: "card-builder", Prompt: "把单词排成一句话", Bank: []string{"This", "is", "an", "apple"}, Answer: "This is an apple",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "整句读音")
}

func TestRejectsLegacyIndexSnapshots(t *testing.T) {
	_, err := englishcontent.PlanExampleFromSnapshot(`{"questionId":1,"code":"listen","answerIndex":0}`, "0")
	require.Error(t, err)
}

func TestRejectsCodeTypeOnlySnapshots(t *testing.T) {
	_, err := englishcontent.PlanExampleFromSnapshot(`{"code":"listen","type":"choice"}`, "")
	require.Error(t, err)
}

func TestAttemptSelectedIgnoresAccumulatedPicks(t *testing.T) {
	require.Equal(t, "101", englishcontent.AttemptSelected("101", `[{"clientId":"a","optionIndex":1}]`))
	require.Equal(t, "102", englishcontent.AttemptSelected("", "102"))
	require.Empty(t, englishcontent.AttemptSelected("", `[{"clientId":"a","optionIndex":1,"correct":false}]`))
	require.Equal(t, "1", englishcontent.AttemptSelected("[]", "1"))
}

func TestSnapshotFromLiveQuestionFreezesDisplayOrderAndStableIDs(t *testing.T) {
	q := englishcontent.LiveQuestion{
		Code: "listen", Stem: "听一听", Word: "cat", TargetKpID: 100,
		Options:     `[{"kpId":100,"label":"cat"},{"kpId":101,"label":"dog"},{"kpId":102,"label":"bird"},{"kpId":103,"label":"fish"}]`,
		Answer:      `{"index":0}`,
		Speech:      `{"text":"cat"}`,
		OptionOrder: "1,0,3,2",
		Meanings:    map[int64]string{100: "猫", 101: "小狗", 102: "小鸟", 103: "小鱼"},
		AssetKeys:   map[int64]englishcontent.AssetKey{100: {Speech: "english/speech/100.mp3", Sense: "/s100"}},
	}
	snap, err := englishcontent.SnapshotFromLiveQuestion(q)
	require.NoError(t, err)
	require.Equal(t, 1, snap.Schema)
	require.Equal(t, "audio-choice", snap.Kind)
	require.Equal(t, "listen", snap.SkillCode)
	require.Equal(t, []string{"101", "100", "103", "102"}, idsOf(snap.Example.Options))
	require.Equal(t, "猫", snap.Example.Options[1].Label)
	require.Equal(t, "100", snap.Example.AnswerID)
	require.Equal(t, "/api/v1/english/words/100/speech.mp3", snap.Example.SpeechURL)
	require.Empty(t, snap.MediaSHA256, "only freezing actual bytes may produce a digest")
	id, err := englishcontent.OptionIDForDisplayIndex(q.Options, q.OptionOrder, 1)
	require.NoError(t, err)
	require.Equal(t, "100", id)
	wrong, err := englishcontent.OptionIDForDisplayIndex(q.Options, q.OptionOrder, 0)
	require.NoError(t, err)
	require.Equal(t, "101", wrong)
}

func idsOf(opts []englishcontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}

func TestSnapshotFromLiveQuestionPictureFreezesPromptAndSense(t *testing.T) {
	q := englishcontent.LiveQuestion{
		Code: "picture", Stem: "", Word: "cat", TargetKpID: 100,
		Options:     `[{"kpId":100,"assetKind":"sense"},{"kpId":101,"assetKind":"sense"},{"kpId":102,"assetKind":"sense"},{"kpId":103,"assetKind":"sense"}]`,
		Answer:      `{"index":0}`,
		Speech:      `{"text":"cat"}`,
		OptionOrder: "0,1,2,3",
		Meanings:    map[int64]string{100: "猫", 101: "小狗", 102: "小鸟", 103: "小鱼"},
		AssetKeys:   map[int64]englishcontent.AssetKey{100: {Speech: "english/speech/100.mp3", Sense: "english/sense/100.png"}, 101: {Sense: "english/sense/101.png"}},
	}
	snap, err := englishcontent.SnapshotFromLiveQuestion(q)
	require.NoError(t, err)
	require.Equal(t, "image-text", snap.Kind)
	require.Equal(t, "看图选词", snap.Example.Prompt)
	require.Equal(t, "100", snap.Example.AnswerID)
	require.Equal(t, "/api/v1/english/words/100/sense.png", snap.Example.Options[0].Picture)
	require.Empty(t, snap.MediaSHA256)
}

func TestOrderAndInputKeepRecordedValues(t *testing.T) {
	order := englishcontent.EnglishExample{Kind: "card-builder", Prompt: "把单词排成一句话", SpeechURL: "/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", Bank: []string{"This", "is", "an", "apple"}, Answer: "This is an apple"}
	raw, err := englishcontent.HistoryBytes(englishcontent.HistorySnapshot{Kind: "card-builder", Example: order, Selected: "is This an apple"})
	require.NoError(t, err)
	got, err := englishcontent.PlanExampleFromSnapshot(string(raw), "This is an apple")
	require.NoError(t, err)
	require.Equal(t, "order", got.ResponseKind)
	require.Equal(t, "This is an apple", got.Selected)

	input := englishcontent.EnglishExample{Kind: "input-gap", Prompt: "写出这个单词", Answer: "apple"}
	raw, err = englishcontent.HistoryBytes(englishcontent.HistorySnapshot{Kind: "input-gap", Example: input, Selected: "aple"})
	require.NoError(t, err)
	got, err = englishcontent.PlanExampleFromSnapshot(string(raw), "")
	require.NoError(t, err)
	require.Equal(t, "input", got.ResponseKind)
	require.Equal(t, "aple", got.Selected)
}
