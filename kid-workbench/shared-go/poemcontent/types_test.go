package poemcontent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleWorks() []Work {
	return []Work{
		ParseWork(1, "pm001", "静夜思", `{"kind":"poem","author":"李白","dynasty":"唐","lines":["床前明月光","疑是地上霜","举头望明月","低头思故乡"]}`),
		ParseWork(2, "pm002", "春晓", `{"kind":"poem","author":"孟浩然","dynasty":"唐","lines":["春眠不觉晓","处处闻啼鸟","夜来风雨声","花落知多少"]}`),
		ParseWork(3, "pm003", "咏鹅", `{"kind":"poem","author":"骆宾王","dynasty":"唐","lines":["鹅鹅鹅","曲项向天歌","白毛浮绿水","红掌拨清波"]}`),
		ParseWork(4, "pm004", "悯农", `{"kind":"poem","author":"李绅","dynasty":"唐","lines":["锄禾日当午","汗滴禾下土","谁知盘中餐","粒粒皆辛苦"]}`),
		ParseWork(5, "pm005", "登鹳雀楼", `{"kind":"poem","author":"王之涣","dynasty":"唐","lines":["白日依山尽","黄河入海流","欲穷千里目","更上一层楼"]}`),
		ParseWork(12, "pm012", "草", `{"kind":"poem","author":"白居易","dynasty":"唐","lines":["离离原上草","一岁一枯荣","野火烧不尽","春风吹又生"]}`),
		ParseWork(39, "pm039", "赋得古原草送别", `{"kind":"poem","author":"白居易","dynasty":"唐","lines":["离离原上草","一岁一枯荣","野火烧不尽","春风吹又生","远芳侵古道","晴翠接荒城","又送王孙去","萋萋满别情"]}`),
	}
}

func TestGenerateFourKinds(t *testing.T) {
	all := sampleWorks()
	jingye := all[0]
	title, err := Generate(GenerateOpts{Kind: KindTitle, Work: jingye, All: all, Seed: 7, KpID: 1})
	require.NoError(t, err)
	require.Equal(t, "pm001", title.AnswerID)
	require.Equal(t, "pm001", title.WorkID)
	require.NotEqual(t, title.Options[0].ID, title.AnswerID)
	require.Contains(t, title.SpeechURL, "/speech/")

	fill, err := Generate(GenerateOpts{Kind: KindFill, Work: jingye, All: all, Seed: 3, KpID: 1})
	require.NoError(t, err)
	require.NotEmpty(t, fill.GapIndexes)
	require.Contains(t, fill.Line, "□")
	require.Empty(t, fill.SpeechURL)

	couplet, err := Generate(GenerateOpts{Kind: KindCouplet, Work: jingye, All: all, Seed: 3, KpID: 1})
	require.NoError(t, err)
	require.Equal(t, "pm001:L1", couplet.UpperLineID)
	require.Equal(t, "pm001:L2", couplet.NextLineID)
	require.Equal(t, couplet.NextLineID, couplet.AnswerID)

	recite, err := Generate(GenerateOpts{Kind: KindRecite, Work: jingye, All: all, Seed: 9, KpID: 1})
	require.NoError(t, err)
	require.Equal(t, []string{"pm001:L1", "pm001:L2", "pm001:L3", "pm001:L4"}, recite.CorrectSequence)
	require.Len(t, recite.SequenceDisplayOrder, 4)
	require.NotEqual(t, recite.CorrectSequence, recite.SequenceDisplayOrder)
	require.Empty(t, recite.SpeechURL)
}

func TestGrassTitleSkippedWhenPrefixOfLongerWork(t *testing.T) {
	all := sampleWorks()
	_, err := Generate(GenerateOpts{Kind: KindTitle, Work: all[5], All: all, Seed: 1, KpID: 12})
	require.Error(t, err)
	ex, err := Generate(GenerateOpts{Kind: KindTitle, Work: all[6], All: all, Seed: 1, KpID: 39})
	require.NoError(t, err)
	require.Equal(t, "pm039", ex.AnswerID)
	require.Equal(t, "pm039:L5", ex.LineID)
	require.Equal(t, "远芳侵古道", ex.Line)
}

func TestReciteKeepsAllLines(t *testing.T) {
	all := sampleWorks()
	ex, err := Generate(GenerateOpts{Kind: KindRecite, Work: all[6], All: all, Seed: 2, KpID: 39})
	require.NoError(t, err)
	require.Len(t, ex.CorrectSequence, 8)
	require.True(t, strings.Contains(ex.Prompt, "8"))
}

func TestFillGapIsRuneIndex(t *testing.T) {
	require.Equal(t, "锄禾日当□", DisplayFillLine("锄禾日当午", []int{4}))
	wrong := DecodeInput("char:天", KindFill)
	ex := PoemExample{Kind: KindFill, Prompt: "缺的字是哪个？", WorkID: "pm004", LineID: "pm004:L1", SourceLine: "锄禾日当午", GapIndexes: []int{4},
		Options: []Choice{{ID: "char:午#4", Label: "午"}, {ID: "char:天", Label: "天"}}, AnswerID: "char:午#4"}
	require.False(t, IsCorrect(ex, wrong))
	require.Contains(t, strings.Join(ErrorFacts(ex, wrong), ""), "天")
}

func TestHistoryDoesNotWriteEmptySelected(t *testing.T) {
	ex, err := Generate(GenerateOpts{Kind: KindTitle, Work: sampleWorks()[0], All: sampleWorks(), Seed: 4, KpID: 1})
	require.NoError(t, err)
	raw, err := HistoryBytes(HistorySnapshot{Schema: 1, Kind: KindTitle, Example: ex})
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"selected":"{`)
	got, err := PlanExampleFromSnapshot(string(raw), "")
	require.NoError(t, err)
	require.Empty(t, got.Selected)
}

func TestSpeechWAVIsRealWaveBytes(t *testing.T) {
	a := SpeechWAV("床前明月光")
	b := SpeechWAV("疑是地上霜")
	require.True(t, strings.HasPrefix(string(a[:4]), "RIFF"))
	require.NotEqual(t, a, b)
	ext, kind, err := detectAudio(a)
	require.NoError(t, err)
	require.Equal(t, ".wav", ext)
	require.Equal(t, "audio", kind)
}
