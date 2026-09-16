package quiz_test

import (
	"encoding/json"
	"testing"

	"github.com/conchi/chengyu-server/internal/quiz"
	"github.com/stretchr/testify/require"
)

func TestBuildMeaningAndGeneratedTypes(t *testing.T) {
	payload := quiz.Payload{Kind: "chengyu", Pinyin: "yì xīn yì yì", Meaning: "集中精神，做事专心", Example: "做作业要一心一意。", Wrong: []string{"心思不专一", "慢慢来", "随便玩玩"}}
	siblings := []quiz.Sibling{
		{KpID: 100, Title: "一心一意"},
		{KpID: 101, Title: "二话不说"},
		{KpID: 102, Title: "三心二意"},
		{KpID: 103, Title: "五颜六色"},
	}

	meaning, ok := quiz.Build("meaning", "一心一意", payload, siblings, 910, 100)
	require.True(t, ok)
	require.Equal(t, "meaning", meaning.Code)
	require.Contains(t, meaning.Options, "集中精神，做事专心")

	pick, ok := quiz.Build("pick", "一心一意", payload, siblings, 910, 100)
	require.True(t, ok)
	require.Equal(t, "pick", pick.Code)
	require.Contains(t, pick.Options, `"kpId":100`)
	require.Contains(t, pick.Options, "一心一意")

	pinyin, ok := quiz.Build("pinyin", "一心一意", payload, siblings, 910, 100)
	require.True(t, ok)
	var visual struct{ Kind, Text string }
	require.NoError(t, json.Unmarshal([]byte(pinyin.Visual), &visual))
	require.Equal(t, "pinyin", visual.Kind)
	require.Equal(t, "yì xīn yì yì", visual.Text)

	example, ok := quiz.Build("example", "一心一意", payload, siblings, 910, 100)
	require.True(t, ok)
	var blank struct {
		Kind, Text, Full, Blanked, Target string
		Start, Length                     int
	}
	require.NoError(t, json.Unmarshal([]byte(example.Visual), &blank))
	require.Equal(t, "做作业要____。", blank.Text)
	require.Equal(t, "做作业要一心一意。", blank.Full)
	require.Equal(t, "一心一意", blank.Target)
	require.Equal(t, 4, blank.Start)
	require.Equal(t, 4, blank.Length)
}

func TestBuildRejectsUnknownCode(t *testing.T) {
	_, ok := quiz.Build("listen", "一心一意", quiz.Payload{Kind: "chengyu", Meaning: "专心"}, nil, 1, 1)
	require.False(t, ok)
}

func TestBuildRejectsExampleWithoutIdiom(t *testing.T) {
	payload := quiz.Payload{Kind: "chengyu", Pinyin: "wǔ yán liù sè", Meaning: "形容色彩繁多", Example: "花园里开着花。", Wrong: []string{"只有一种颜色", "黑漆漆的", "灰蒙蒙的"}}
	siblings := []quiz.Sibling{{KpID: 1, Title: "五颜六色"}, {KpID: 2, Title: "一心一意"}, {KpID: 3, Title: "二话不说"}, {KpID: 4, Title: "三心二意"}}
	_, ok := quiz.Build("example", "五颜六色", payload, siblings, 1, 1)
	require.False(t, ok)
}
