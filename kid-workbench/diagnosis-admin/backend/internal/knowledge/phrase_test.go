package knowledge

import (
	"testing"

	"github.com/conchi/study-learning/phrasecontent"
	"github.com/stretchr/testify/require"
)

func TestDecodePhraseChoice(t *testing.T) {
	raw := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","selected":"2","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"早上好。"},{"id":"2","label":"下午好。"}],"answerId":"1"}}`
	r := Evidence{SubjectCode: "phrase", Title: "Good morning.", IsCorrect: false, Media: map[string]string{}}
	decodePhrase(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "listen_zh", Snapshot: raw, Picks: "2"})
	require.Equal(t, "instance_snapshot", r.QuestionFidelity)
	require.Equal(t, "immutable", r.MediaFidelity)
	require.Equal(t, "2", r.Response.SelectedOptionID)
	require.Equal(t, "phrase:/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", r.Media["stem-audio"])
	require.Equal(t, "下午好。", r.PhraseExample.Options[1].Label)
}

func TestDecodePhraseRejectsLiveSpeechAsFrozen(t *testing.T) {
	raw := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","example":{"kind":"listen_zh","speechUrl":"/api/v1/phrase/items/1/speech.mp3","options":[{"id":"1","label":"早上好。"},{"id":"2","label":"下午好。"}],"answerId":"1"}}`
	r := Evidence{SubjectCode: "phrase", SkillCode: "listen_zh", Media: map[string]string{}}
	decodePhrase(&r, mathRow{ItemID: 1, SkillCode: "listen_zh", Snapshot: raw, Picks: "1"})
	require.Equal(t, "mutable_reference", r.MediaFidelity)
	require.Contains(t, r.EvidenceReasonCodes, "media_not_frozen")
}

func TestDecodePhraseRejectsCodeTypeSnapshot(t *testing.T) {
	r := Evidence{SubjectCode: "phrase", SkillCode: "listen_zh", Media: map[string]string{}}
	decodePhrase(&r, mathRow{ItemID: 1, SkillCode: "listen_zh", Snapshot: `{"code":"listen_zh","type":"choice"}`, Picks: "2"})
	require.Contains(t, r.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, r.PhraseExample)
}

func TestHasFrozenPhraseMedia(t *testing.T) {
	require.True(t, phrasecontent.HasFrozenMedia(phrasecontent.PhraseExample{Kind: "scene", Options: []phrasecontent.Choice{{ID: "1", Label: "a"}, {ID: "2", Label: "b"}}, AnswerID: "1"}))
	require.False(t, phrasecontent.HasFrozenMedia(phrasecontent.PhraseExample{Kind: "listen_zh", SpeechURL: "/api/v1/phrase/items/1/speech.mp3"}))
}
