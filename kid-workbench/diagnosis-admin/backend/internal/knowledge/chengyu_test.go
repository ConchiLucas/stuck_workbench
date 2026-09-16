package knowledge

import (
	"testing"

	"github.com/conchi/study-learning/chengyucontent"
	"github.com/stretchr/testify/require"
)

func TestDecodeChengyuChoice(t *testing.T) {
	raw := `{"schema":1,"kind":"meaning","skillCode":"meaning","responseKind":"choice","selected":"label:心思不专一","example":{"kind":"meaning","speech":"一心一意","speechUrl":"/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"label:集中精神，做事专心","label":"集中精神，做事专心"},{"id":"label:心思不专一","label":"心思不专一"}],"answerId":"label:集中精神，做事专心"}}`
	r := Evidence{SubjectCode: "chengyu", Title: "一心一意", IsCorrect: false, Media: map[string]string{}}
	decodeChengyu(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "meaning", Snapshot: raw, Picks: "label:心思不专一"})
	require.Equal(t, "instance_snapshot", r.QuestionFidelity)
	require.Equal(t, "immutable", r.MediaFidelity)
	require.Equal(t, "label:心思不专一", r.Response.SelectedOptionID)
	require.Equal(t, "chengyu:/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", r.Media["stem-audio"])
	require.Equal(t, "心思不专一", r.ChengyuExample.Options[1].Label)
}

func TestDecodeChengyuRejectsLiveSpeechAsFrozen(t *testing.T) {
	raw := `{"schema":1,"kind":"meaning","skillCode":"meaning","example":{"kind":"meaning","speechUrl":"/api/v1/chengyu/items/1/speech.mp3","options":[{"id":"label:a","label":"a"},{"id":"label:b","label":"b"}],"answerId":"label:a"}}`
	r := Evidence{SubjectCode: "chengyu", SkillCode: "meaning", Media: map[string]string{}}
	decodeChengyu(&r, mathRow{ItemID: 1, SkillCode: "meaning", Snapshot: raw, Picks: "label:a"})
	require.Equal(t, "mutable_reference", r.MediaFidelity)
	require.Contains(t, r.EvidenceReasonCodes, "media_not_frozen")
	require.Empty(t, r.Media["stem-audio"])
	require.Equal(t, "", r.ChengyuExample.SpeechURL)
}

func TestDecodeChengyuRejectsCodeTypeSnapshot(t *testing.T) {
	r := Evidence{SubjectCode: "chengyu", SkillCode: "meaning", Media: map[string]string{}}
	decodeChengyu(&r, mathRow{ItemID: 1, SkillCode: "meaning", Snapshot: `{"code":"meaning","type":"choice"}`, Picks: "2"})
	require.Contains(t, r.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, r.ChengyuExample)
}

func TestHasFrozenChengyuMedia(t *testing.T) {
	require.True(t, chengyucontent.HasFrozenMedia(chengyucontent.ChengyuExample{Kind: "pick", Options: []chengyucontent.Choice{{ID: "1", Label: "a"}, {ID: "2", Label: "b"}}, AnswerID: "1"}))
	require.False(t, chengyucontent.HasFrozenMedia(chengyucontent.ChengyuExample{Kind: "meaning", SpeechURL: "/api/v1/chengyu/items/1/speech.mp3"}))
}
