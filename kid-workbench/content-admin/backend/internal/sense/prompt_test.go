package sense_test

import (
	"testing"

	"github.com/conchi/study-content-admin/internal/sense"
	"github.com/stretchr/testify/require"
)

func TestPromptLocksStyle(t *testing.T) {
	p := sense.Prompt("火")
	require.Contains(t, p, "flame")
	require.Contains(t, p, "pure white background")
	require.Contains(t, p, "never write any characters")
	require.False(t, sense.ContainsHan(p))
}

func TestNoHanInAnyMappedPrompt(t *testing.T) {
	chars := []string{
		"一", "尺", "金", "铜", "虫", "大", "来", "红", "车", "牛", "刀", "银", "铁",
		"头", "衣", "饭", "早", "走", "眼", "树", "刷", "里", "冷",
	}
	for _, ch := range chars {
		p := sense.Prompt(ch)
		require.False(t, sense.ContainsHan(p), "prompt for %s leaked Han: %s", ch, p)
		require.False(t, sense.ContainsHan(sense.SubjectEN(ch)), "subject for %s leaked Han", ch)
	}
}

func TestProblemCharsAreConcreteObjects(t *testing.T) {
	require.Contains(t, sense.SubjectEN("尺"), "ruler")
	require.Contains(t, sense.SubjectEN("金"), "gold")
	require.Contains(t, sense.SubjectEN("铜"), "copper")
	require.Contains(t, sense.SubjectEN("虫"), "caterpillar")
}

func TestSameGroupSensesStayDistinguishable(t *testing.T) {
	front := sense.SubjectEN("前")
	back := sense.SubjectEN("后")
	require.Contains(t, front, "face")
	require.NotContains(t, front, "back")
	require.Contains(t, back, "back")
	require.NotContains(t, back, "face")

	face := sense.SubjectEN("脸")
	laugh := sense.SubjectEN("笑")
	require.Contains(t, face, "hair")
	require.Contains(t, face, "human face")
	require.Contains(t, laugh, "laugh")

	require.NotContains(t, sense.SubjectEN("字"), "child")
	require.NotContains(t, sense.SubjectEN("文"), "child")
	require.Contains(t, sense.SubjectEN("文"), "paper")

	require.Contains(t, sense.SubjectEN("果"), "banana")
	require.Contains(t, sense.SubjectEN("果"), "grapes")
	require.Contains(t, sense.SubjectEN("苹"), "apple")

	east := sense.SubjectEN("东")
	west := sense.SubjectEN("西")
	require.Contains(t, east, "RIGHT")
	require.Contains(t, west, "LEFT")
	require.Contains(t, west, "orange")

	require.Contains(t, sense.SubjectEN("白"), "gray")
}
