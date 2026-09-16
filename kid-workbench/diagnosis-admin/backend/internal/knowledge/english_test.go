package knowledge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeEnglishChoiceAndInput(t *testing.T) {
	choice := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"2","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"苹果","picture":"/api/v1/english/task-media/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.jpg"},{"id":"2","label":"小狗","picture":"/api/v1/english/task-media/cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc.jpg"}],"answerId":"1"}}`
	r := Evidence{SubjectCode: "english", Title: "apple", IsCorrect: false, Media: map[string]string{}}
	decodeEnglish(&r, mathRow{AttemptID: 8, PlanID: 7, ItemID: 9, SkillCode: "listen", Picks: "2", Snapshot: choice})
	require.Equal(t, "instance_snapshot", r.QuestionFidelity)
	require.Equal(t, "2", r.Response.SelectedOptionID)
	require.NotNil(t, r.EnglishExample)
	require.Equal(t, "english:/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", r.Media["stem-audio"])

	typed := `{"schema":1,"kind":"input-gap","skillCode":"type","responseKind":"input","selected":"aple","example":{"kind":"input-gap","prompt":"写出这个单词","answer":"apple","cue":"/api/v1/english/task-media/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.jpg"}}`
	r = Evidence{SubjectCode: "english", Title: "apple", IsCorrect: false, Media: map[string]string{}}
	decodeEnglish(&r, mathRow{ItemID: 9, SkillCode: "type", Picks: "aple", Snapshot: typed})
	require.Equal(t, "input", r.Response.Kind)
	require.Equal(t, "aple", r.Response.Value)
	require.Equal(t, "apple", r.Question.TargetText)
}

func TestDecodeEnglishUnlinkedWhenItemMissing(t *testing.T) {
	r := Evidence{SubjectCode: "english", SkillCode: "listen", Media: map[string]string{}}
	decodeEnglish(&r, mathRow{SkillCode: "listen", Snapshot: `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"2","example":{"kind":"audio-choice","speech":"apple","options":[{"id":"1","label":"苹果"},{"id":"2","label":"小狗"}],"answerId":"1"}}`})
	require.Contains(t, r.EvidenceReasonCodes, "unlinked_plan_item")
	require.Nil(t, r.EnglishExample)
}

func TestDecodeEnglishRejectsLegacyIndex(t *testing.T) {
	r := Evidence{SubjectCode: "english", SkillCode: "listen", Media: map[string]string{}}
	decodeEnglish(&r, mathRow{ItemID: 9, SkillCode: "listen", Snapshot: `{"questionId":1,"code":"listen","answerIndex":0}`})
	require.Contains(t, r.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, r.EnglishExample)
}

func TestEnglishLiveMediaIsNotImmutable(t *testing.T) {
	raw := `{"schema":1,"kind":"audio-choice","skillCode":"listen","example":{"kind":"audio-choice","speechUrl":"/api/v1/english/words/1/speech.mp3","options":[{"id":"1","label":"cat"},{"id":"2","label":"dog"}],"answerId":"1"}}`
	r := Evidence{SubjectCode: "english", Media: map[string]string{}}
	decodeEnglish(&r, mathRow{ItemID: 1, Snapshot: raw, Picks: "2", SkillCode: "listen"})
	require.Equal(t, "mutable_reference", r.MediaFidelity)
}
