package knowledge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeMathCalcSnapshot(t *testing.T) {
	r := Evidence{KpID: 10, ChildID: 1, SubjectCode: "math", ModuleCode: "add10", Title: "2+5", Media: map[string]string{}}
	decodeMath(&r, mathRow{AttemptID: 8, PlanID: 7, ItemID: 9, SkillCode: "calc", Picks: "2", Snapshot: `{"questionId":42,"code":"calc","stem":"2 + 5 = ?","options":[{"label":"5"},{"label":"6"},{"label":"7"},{"label":"8"}],"answerIndex":2,"visual":{"kind":"equation","a":2,"b":5,"operator":"+"}}`})
	require.NotNil(t, r.MathExample)
	require.Equal(t, "choice", r.MathExample.Kind)
	require.Equal(t, "2 + 5", r.MathExample.Prompt)
	require.Empty(t, r.MathExample.AudioURL)
	require.Equal(t, "instance_snapshot", r.QuestionFidelity)
	require.Equal(t, "o3", r.Response.SelectedOptionID)
	require.Equal(t, "算式计算", r.SkillLabel)
}

func TestDecodeMathFindRequiresAudio(t *testing.T) {
	r := Evidence{KpID: 12, ChildID: 1, SubjectCode: "math", ModuleCode: "shape", Media: map[string]string{}}
	decodeMath(&r, mathRow{SkillCode: "find", Snapshot: `{"code":"find","options":[{"shape":"circle"},{"shape":"square"},{"shape":"triangle"},{"shape":"star"}],"answerIndex":0}`})
	require.Contains(t, r.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, r.MathExample)
	r = Evidence{KpID: 12, ChildID: 1, SubjectCode: "math", ModuleCode: "shape", Media: map[string]string{}}
	decodeMath(&r, mathRow{PlanID: 7, ItemID: 9, SkillCode: "find", Picks: "0", Snapshot: `{"code":"find","options":[{"shape":"circle"},{"shape":"square"},{"shape":"triangle"},{"shape":"star"}],"answerIndex":0,"audioObjectKey":"math/questions/50.mp3"}`})
	require.NotNil(t, r.MathExample)
	require.Equal(t, "audio-shape", r.MathExample.Kind)
	require.True(t, r.AudioMutable)
	require.Equal(t, "math:/api/v1/children/1/math/plans/7/items/9/audio.mp3", r.Media["stem-audio"])
}
