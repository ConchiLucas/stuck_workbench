package phrasecontent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistoryBytesRequiresStableOptions(t *testing.T) {
	example := PhraseExample{
		Kind: "listen_zh", Stem: "听一听，选出中文意思", Speech: "Good morning.",
		SpeechURL: "/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3",
		Options: []Choice{
			{ID: "1", Label: "早上好。"},
			{ID: "2", Label: "下午好。"},
		},
		AnswerID: "1",
	}
	raw, err := HistoryBytes(HistorySnapshot{Kind: "listen_zh", Example: example, Selected: "2"})
	require.NoError(t, err)
	got, err := PlanExampleFromSnapshot(string(raw), "2")
	require.NoError(t, err)
	require.Equal(t, "2", got.Selected)
	require.Equal(t, "choice", got.ResponseKind)
}

func TestSnapshotFromLiveQuestionFreezesDisplayOrderAndStableIDs(t *testing.T) {
	q := LiveQuestion{
		Code: "listen_zh", Stem: "听一听，选出中文意思",
		Options: `[{"kpId":100,"label":"早上好。"},{"kpId":101,"label":"下午好。"},{"kpId":102,"label":"晚上好。"},{"kpId":103,"label":"晚安。"}]`,
		Answer: `{"index":0}`, Speech: `{"text":"Good morning.","lang":"en-US"}`, Visual: `{}`,
		OptionOrder: "1,0,3,2", Phrase: "Good morning.", MeaningZh: "早上好。", Scene: "早上见到老师", TargetKpID: 100,
	}
	snap, err := SnapshotFromLiveQuestion(q)
	require.NoError(t, err)
	require.Equal(t, 1, snap.Schema)
	require.Equal(t, "listen_zh", snap.Kind)
	require.Equal(t, []string{"101", "100", "103", "102"}, ids(snap.Example.Options))
	require.Equal(t, "100", snap.Example.AnswerID)
	require.Equal(t, "/api/v1/phrase/items/100/speech.mp3", snap.Example.SpeechURL)
	id, err := OptionIDForDisplayIndex(q.Options, q.OptionOrder, 0)
	require.NoError(t, err)
	require.Equal(t, "101", id)
}

func TestSnapshotReplyUsesQuestionAudio(t *testing.T) {
	q := LiveQuestion{
		Code: "reply", Stem: "对方说了这句话，你怎么答？",
		Options: `[{"kpId":101,"label":"I'm fine."},{"kpId":100,"label":"Good morning."}]`,
		Answer: `{"index":0}`, Speech: `{"text":"How are you?"}`, Visual: `{"kind":"prompt","text":"How are you?"}`,
		OptionOrder: "0,1", Phrase: "I'm fine.", ReplyTo: "How are you?", TargetKpID: 101, ReplyToKpID: 106,
	}
	snap, err := SnapshotFromLiveQuestion(q)
	require.NoError(t, err)
	require.Equal(t, "How are you?", snap.Example.Prompt)
	require.Equal(t, "/api/v1/phrase/items/106/speech.mp3", snap.Example.SpeechURL)
}

func TestAttemptSelectedIgnoresAccumulatedPicks(t *testing.T) {
	require.Equal(t, "101", AttemptSelected("101", "0,1"))
	require.Empty(t, AttemptSelected("", "0,1"))
	require.Empty(t, AttemptSelected("", `[{"optionIndex":1}]`))
}

func TestSceneDoesNotRequireSpeechURL(t *testing.T) {
	err := Validate(PhraseExample{
		Kind: "scene", Prompt: "早上见到老师",
		Options:  []Choice{{ID: "1", Label: "Good morning."}, {ID: "2", Label: "Hello!"}},
		AnswerID: "1",
	})
	require.NoError(t, err)
}

func ids(opts []Choice) []string {
	out := make([]string, len(opts))
	for i, o := range opts {
		out[i] = o.ID
	}
	return out
}
