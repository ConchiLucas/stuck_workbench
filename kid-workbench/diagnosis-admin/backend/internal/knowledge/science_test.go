package knowledge

import (
	"strings"
	"testing"

	"github.com/conchi/study-learning/sciencecontent"
	"github.com/stretchr/testify/require"
)

func TestDecodeScienceChoice(t *testing.T) {
	raw := `{"schema":1,"kind":"choice","skillCode":"choice","responseKind":"choice","selected":"cat","example":{"kind":"choice","prompt":"哪种动物的脚掌最适合在水里游泳？","options":[{"id":"cat","label":"猫"},{"id":"duck","label":"鸭子"},{"id":"rabbit","label":"兔子"}],"answerId":"duck"}}`
	r := Evidence{SubjectCode: "science", Title: "鸭子的脚掌", IsCorrect: false, Media: map[string]string{}}
	decodeScience(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "choice", Snapshot: raw, Picks: "cat"})
	require.Equal(t, "instance_snapshot", r.QuestionFidelity)
	require.Equal(t, "immutable", r.MediaFidelity)
	require.Equal(t, "cat", r.Response.SelectedOptionID)
	require.Equal(t, "选择题", r.SkillLabel)
	require.Equal(t, "鸭子", r.ScienceExample.Options[1].Label)
}

func TestDecodeScienceMatchFacts(t *testing.T) {
	raw := `{"schema":1,"kind":"match","skillCode":"match","responseKind":"match","example":{"kind":"match","prompt":"连","matchSources":[{"id":"frog","label":"青蛙"},{"id":"bear","label":"北极熊"}],"matchTargets":[{"id":"forest","label":"热带雨林","imageUrl":"/api/v1/science/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"},{"id":"ice","label":"冰原"}],"matchAnswers":{"frog":"forest","bear":"ice"}}}`
	r := Evidence{SubjectCode: "science", IsCorrect: false, Media: map[string]string{}}
	decodeScience(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "match", Snapshot: raw, Picks: `{"kind":"match","pairs":{"frog":"ice","bear":"forest"}}`})
	require.Equal(t, "match", r.Response.Kind)
	require.Equal(t, "science:/api/v1/science/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png", r.Media["match-target-0"])
	require.Contains(t, strings.Join(sciencecontent.ErrorFacts(*r.ScienceExample, sciencecontent.DecodeInput(r.Response.Value, "match")), "\n"), "青蛙")
}

func TestDecodeScienceRejectsLiveImageAsFrozen(t *testing.T) {
	raw := `{"schema":1,"kind":"match","skillCode":"match","example":{"kind":"match","prompt":"连","matchSources":[{"id":"frog","label":"青蛙"},{"id":"bear","label":"北极熊"}],"matchTargets":[{"id":"forest","label":"热带雨林","imageUrl":"/api/v1/science/items/9/diagram.png"},{"id":"ice","label":"冰原"}],"matchAnswers":{"frog":"forest","bear":"ice"}}}`
	r := Evidence{SubjectCode: "science", SkillCode: "match", Media: map[string]string{}}
	decodeScience(&r, mathRow{ItemID: 1, SkillCode: "match", Snapshot: raw, Picks: `{"kind":"match","pairs":{"frog":"forest","bear":"ice"}}`})
	require.Equal(t, "mutable_reference", r.MediaFidelity)
	require.Contains(t, r.EvidenceReasonCodes, "media_not_frozen")
}

func TestDecodeScienceRejectsCodeTypeSnapshot(t *testing.T) {
	r := Evidence{SubjectCode: "science", SkillCode: "choice", Media: map[string]string{}}
	decodeScience(&r, mathRow{ItemID: 1, SkillCode: "choice", Snapshot: `{"code":"choice","type":"choice"}`, Picks: "duck"})
	require.Contains(t, r.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, r.ScienceExample)
}

func TestHasFrozenScienceMedia(t *testing.T) {
	require.True(t, sciencecontent.HasFrozenMedia(sciencecontent.ScienceExample{Kind: "choice", Prompt: "q", Options: []sciencecontent.Choice{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}, AnswerID: "a"}))
	require.False(t, sciencecontent.HasFrozenMedia(sciencecontent.ScienceExample{Kind: "match", Prompt: "连", MatchSources: []sciencecontent.Node{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}, MatchTargets: []sciencecontent.Node{{ID: "x", Label: "X", ImageURL: "/api/v1/science/items/1/diagram.png"}, {ID: "y", Label: "Y"}}, MatchAnswers: map[string]string{"a": "x", "b": "y"}}))
}
