package knowledge

import (
	"strings"
	"testing"

	"github.com/conchi/study-learning/poemcontent"
	"github.com/stretchr/testify/require"
)

func TestDecodePoemFill(t *testing.T) {
	raw := `{"schema":1,"kind":"fill","skillCode":"fill","responseKind":"fill","selected":"char:天","example":{"kind":"fill","prompt":"缺的字是哪个？","line":"锄禾日当□","workId":"pm004","lineId":"pm004:L1","sourceLine":"锄禾日当午","gapIndexes":[4],"options":[{"id":"char:天","label":"天"},{"id":"char:午#4","label":"午"}],"answerId":"char:午#4"}}`
	r := Evidence{SubjectCode: "poem", Title: "悯农", IsCorrect: false, Media: map[string]string{}}
	decodePoem(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "fill", Snapshot: raw, Picks: "char:天"})
	require.Equal(t, "instance_snapshot", r.QuestionFidelity)
	require.Equal(t, "immutable", r.MediaFidelity)
	require.Equal(t, "char:天", r.Response.SelectedOptionID)
	require.Equal(t, "补字", r.SkillLabel)
	require.Equal(t, "天", r.PoemExample.Options[0].Label)
}

func TestDecodePoemReciteFacts(t *testing.T) {
	raw := `{"schema":1,"kind":"recite","skillCode":"recite","responseKind":"recite","example":{"kind":"recite","prompt":"按顺序点出这4句","line":"床前明月光","workId":"pm001","sequenceItems":[{"id":"pm001:L1","label":"床前明月光"},{"id":"pm001:L2","label":"疑是地上霜"},{"id":"pm001:L3","label":"举头望明月"},{"id":"pm001:L4","label":"低头思故乡"}],"sequenceDisplayOrder":["pm001:L2","pm001:L1","pm001:L4","pm001:L3"],"correctSequence":["pm001:L1","pm001:L2","pm001:L3","pm001:L4"]}}`
	r := Evidence{SubjectCode: "poem", IsCorrect: false, Media: map[string]string{}}
	decodePoem(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "recite", Snapshot: raw, Picks: `{"kind":"recite","sequence":["pm001:L2","pm001:L1","pm001:L3","pm001:L4"]}`})
	require.Equal(t, "recite", r.Response.Kind)
	require.Contains(t, strings.Join(poemcontent.ErrorFacts(*r.PoemExample, poemcontent.DecodeInput(r.Response.Value, "recite")), "\n"), "疑是地上霜")
}

func TestDecodePoemRejectsLiveSpeechAsFrozen(t *testing.T) {
	raw := `{"schema":1,"kind":"title","skillCode":"title","example":{"kind":"title","prompt":"这首诗叫什么？","line":"床前明月光","workId":"pm001","options":[{"id":"pm002","label":"春晓"},{"id":"pm001","label":"静夜思"}],"answerId":"pm001","speechUrl":"/api/v1/poem/items/1/speech/1.wav"}}`
	r := Evidence{SubjectCode: "poem", SkillCode: "title", Media: map[string]string{}}
	decodePoem(&r, mathRow{ItemID: 1, SkillCode: "title", Snapshot: raw, Picks: "pm002"})
	require.Equal(t, "mutable_reference", r.MediaFidelity)
	require.Contains(t, r.EvidenceReasonCodes, "media_not_frozen")
}

func TestDecodePoemRejectsCodeTypeSnapshot(t *testing.T) {
	r := Evidence{SubjectCode: "poem", SkillCode: "title", Media: map[string]string{}}
	decodePoem(&r, mathRow{ItemID: 1, SkillCode: "title", Snapshot: `{"code":"title","type":"choice"}`, Picks: "pm001"})
	require.Contains(t, r.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, r.PoemExample)
}

func TestHasFrozenPoemMedia(t *testing.T) {
	require.True(t, poemcontent.HasFrozenMedia(poemcontent.PoemExample{Kind: "fill", Prompt: "缺的字是哪个？", WorkID: "pm004", LineID: "pm004:L1", SourceLine: "锄禾日当午", GapIndexes: []int{4}, Options: []poemcontent.Choice{{ID: "a", Label: "天"}, {ID: "char:午#4", Label: "午"}}, AnswerID: "char:午#4"}))
	require.False(t, poemcontent.HasFrozenMedia(poemcontent.PoemExample{Kind: "title", Prompt: "这首诗叫什么？", WorkID: "pm001", Options: []poemcontent.Choice{{ID: "pm001", Label: "静夜思"}, {ID: "pm002", Label: "春晓"}}, AnswerID: "pm001", SpeechURL: "/api/v1/poem/items/1/speech/1.wav"}))
}
