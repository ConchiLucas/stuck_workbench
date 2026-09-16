package chengyucontent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistoryBytesRequiresStableOptions(t *testing.T) {
	example := ChengyuExample{
		Kind: "meaning", Stem: "这个成语是什么意思？", Speech: "一心一意",
		SpeechURL: "/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3",
		Options: []Choice{
			{ID: "label:集中精神，做事专心", Label: "集中精神，做事专心"},
			{ID: "label:心思不专一", Label: "心思不专一"},
		},
		AnswerID: "label:集中精神，做事专心",
		Chengyu:  "一心一意",
		Meaning:  "集中精神，做事专心",
	}
	raw, err := HistoryBytes(HistorySnapshot{Kind: "meaning", Example: example, Selected: "label:心思不专一"})
	require.NoError(t, err)
	got, err := PlanExampleFromSnapshot(string(raw), "label:心思不专一")
	require.NoError(t, err)
	require.Equal(t, "label:心思不专一", got.Selected)
	require.Equal(t, "choice", got.ResponseKind)
}

func TestSnapshotFromLiveQuestionFreezesDisplayOrderAndStableIDs(t *testing.T) {
	q := LiveQuestion{
		Code: "pick", Stem: "看意思，点出这个成语",
		Options:     `[{"kpId":100,"label":"一心一意"},{"kpId":101,"label":"二话不说"},{"kpId":102,"label":"三心二意"},{"kpId":103,"label":"五颜六色"}]`,
		Answer:      `{"index":0}`,
		Speech:      `{"text":"一心一意","lang":"zh-CN"}`,
		Visual:      `{"kind":"meaning","text":"集中精神，做事专心"}`,
		OptionOrder: "1,0,3,2", Chengyu: "一心一意", Meaning: "集中精神，做事专心", TargetKpID: 100,
	}
	snap, err := SnapshotFromLiveQuestion(q)
	require.NoError(t, err)
	require.Equal(t, 1, snap.Schema)
	require.Equal(t, "pick", snap.Kind)
	require.Equal(t, []string{"101", "100", "103", "102"}, ids(snap.Example.Options))
	require.Equal(t, "100", snap.Example.AnswerID)
	require.Equal(t, "集中精神，做事专心", snap.Example.Prompt)
	require.Equal(t, "", snap.Example.SpeechURL)
	id, err := OptionIDForDisplayIndex(q.Options, q.OptionOrder, 0)
	require.NoError(t, err)
	require.Equal(t, "101", id)
}

func TestSnapshotMeaningUsesChengyuSpeech(t *testing.T) {
	q := LiveQuestion{
		Code: "meaning",
		Options: `[{"kpId":0,"label":"集中精神，做事专心"},{"kpId":0,"label":"心思不专一"}]`,
		Answer: `{"index":0}`, Speech: `{"text":"一心一意"}`, Visual: `{"kind":"char","text":"一心一意"}`,
		OptionOrder: "0,1", Chengyu: "一心一意", Meaning: "集中精神，做事专心", TargetKpID: 100,
	}
	snap, err := SnapshotFromLiveQuestion(q)
	require.NoError(t, err)
	require.Equal(t, "label:集中精神，做事专心", snap.Example.AnswerID)
	require.Equal(t, "/api/v1/chengyu/items/100/speech.mp3", snap.Example.SpeechURL)
}

func TestSnapshotExampleStoresFirstBlankOnly(t *testing.T) {
	q := LiveQuestion{
		Code: "example",
		Options: `[{"kpId":100,"label":"一心一意"},{"kpId":102,"label":"三心二意"}]`,
		Answer: `{"index":0}`, Visual: `{"kind":"example","text":"做作业要____。","full":"做作业要一心一意。","blanked":"做作业要____。","target":"一心一意","start":4,"length":4}`,
		OptionOrder: "0,1", Chengyu: "一心一意", Example: "做作业要一心一意。", TargetKpID: 100,
	}
	snap, err := SnapshotFromLiveQuestion(q)
	require.NoError(t, err)
	require.NotNil(t, snap.Example.Blank)
	require.Equal(t, "做作业要一心一意。", snap.Example.Blank.Full)
	require.Equal(t, "做作业要____。", snap.Example.Blank.Blanked)
	require.Equal(t, "一心一意", snap.Example.Blank.Target)
	require.Equal(t, 4, snap.Example.Blank.Start)
	require.Equal(t, 4, snap.Example.Blank.Length)
}

func TestFirstBlankRejectsMissingIdiom(t *testing.T) {
	_, err := FirstBlank("花园里开着花。", "五颜六色")
	require.Error(t, err)
}

func TestAttemptSelectedIgnoresAccumulatedPicks(t *testing.T) {
	require.Equal(t, "101", AttemptSelected("101", "0,1"))
	require.Empty(t, AttemptSelected("", "0,1"))
	require.Empty(t, AttemptSelected("", `[{"optionIndex":1}]`))
}

func TestPickDoesNotRequireSpeechURL(t *testing.T) {
	err := Validate(ChengyuExample{
		Kind: "pick", Prompt: "集中精神，做事专心",
		Options:  []Choice{{ID: "1", Label: "一心一意"}, {ID: "2", Label: "三心二意"}},
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
