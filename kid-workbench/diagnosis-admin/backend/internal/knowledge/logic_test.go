package knowledge

import (
	"strings"
	"testing"

	"github.com/conchi/study-learning/logiccontent"
	"github.com/stretchr/testify/require"
)

const logicClassifySnap = `{"schema":1,"kind":"classify","prompt":"哪个不属于这一类？","rule":{"type":"odd-one-out","dimension":"kingdom","inGroup":"animal","explain":"动物"},"objects":[{"id":"cat","caption":"猫","glyph":"cat","attrs":{"category":"animal"}},{"id":"car","caption":"汽车","glyph":"car","attrs":{"category":"vehicle"}}],"options":["cat","car"],"answerId":"car"}`

func TestDecodeLogicClassify(t *testing.T) {
	r := Evidence{SubjectCode: "logic", Title: "动物里的例外", IsCorrect: false, Media: map[string]string{}}
	decodeLogic(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "classify", Snapshot: logicClassifySnap, Picks: `{"selectedId":"cat"}`})
	require.Equal(t, "instance_snapshot", r.QuestionFidelity)
	require.Equal(t, "immutable", r.MediaFidelity)
	require.Equal(t, "cat", r.LogicExample.Options[0])
	require.Equal(t, "分类", r.SkillLabel)
	require.Contains(t, r.Response.SelectedOptionID, "cat")
}

func TestDecodeLogicOrderFacts(t *testing.T) {
	raw := `{"schema":1,"kind":"order","prompt":"按数量从小到大点一排","rule":{"type":"order","dimension":"count","direction":"asc","explain":"数量"},"objects":[{"id":"o1","caption":"1","glyph":"one","attrs":{"count":"1"}},{"id":"o2","caption":"2","glyph":"two","attrs":{"count":"2"}},{"id":"o3","caption":"3","glyph":"three","attrs":{"count":"3"}}],"correctSequence":["o1","o2","o3"],"displayOrder":["o2","o3","o1"]}`
	r := Evidence{SubjectCode: "logic", IsCorrect: false, Media: map[string]string{}}
	decodeLogic(&r, mathRow{ItemID: 1, PlanID: 1, SkillCode: "order", Snapshot: raw, Picks: `{"sequence":["o2","o1","o3"],"rejected":[{"id":"o3","atIndex":0}]}`})
	require.Equal(t, "order", r.Response.Kind)
	require.Contains(t, strings.Join(logiccontent.Facts(*r.LogicExample, convertedInput(r)), "\n"), "误点")
}

func convertedInput(r Evidence) logiccontent.AnswerInput {
	return logiccontent.DecodeSelected(r.Response.Value)
}

func TestDecodeLogicRejectsLiveImageAsFrozen(t *testing.T) {
	raw := `{"schema":1,"kind":"classify","prompt":"哪个不属于这一类？","rule":{"type":"odd-one-out","dimension":"kingdom","inGroup":"animal","explain":"动物"},"objects":[{"id":"cat","caption":"猫","glyph":"cat","attrs":{"category":"animal"}},{"id":"car","caption":"汽车","glyph":"car","attrs":{"category":"vehicle"}}],"options":["cat","car"],"answerId":"car","imageUrls":{"car":"/api/v1/logic/items/9/glyph/car.svg"}}`
	r := Evidence{SubjectCode: "logic", SkillCode: "classify", Media: map[string]string{}}
	decodeLogic(&r, mathRow{ItemID: 1, SkillCode: "classify", Snapshot: raw, Picks: `{"selectedId":"car"}`})
	require.Equal(t, "mutable_reference", r.MediaFidelity)
	require.Contains(t, r.EvidenceReasonCodes, "media_not_frozen")
}

func TestDecodeLogicRejectsCodeTypeSnapshot(t *testing.T) {
	r := Evidence{SubjectCode: "logic", SkillCode: "classify", Media: map[string]string{}}
	decodeLogic(&r, mathRow{ItemID: 1, SkillCode: "classify", Snapshot: `{"code":"classify","type":"choice"}`, Picks: "car"})
	require.Contains(t, r.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, r.LogicExample)
}

func TestHasFrozenLogicMedia(t *testing.T) {
	require.True(t, logiccontent.HasFrozenMedia(logiccontent.LogicExample{Kind: "classify", Prompt: "q", Rule: logiccontent.Rule{Explain: "e"}, Objects: []logiccontent.Object{{ID: "a", Caption: "A", Glyph: "circle"}, {ID: "b", Caption: "B", Glyph: "square"}}, Options: []string{"a", "b"}, AnswerID: "a"}))
	require.False(t, logiccontent.HasFrozenMedia(logiccontent.LogicExample{Kind: "classify", Prompt: "q", Rule: logiccontent.Rule{Explain: "e"}, Objects: []logiccontent.Object{{ID: "a", Caption: "A", Glyph: "circle"}, {ID: "b", Caption: "B", Glyph: "square"}}, Options: []string{"a", "b"}, AnswerID: "a", ImageURLs: map[string]string{"a": "/api/v1/logic/items/1/glyph/a.svg"}}))
}
