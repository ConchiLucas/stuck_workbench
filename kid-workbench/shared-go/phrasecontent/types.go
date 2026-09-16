package phrasecontent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type PhraseExample struct {
	Kind      string   `json:"kind"`
	Stem      string   `json:"stem,omitempty"`
	Prompt    string   `json:"prompt,omitempty"`
	Speech    string   `json:"speech,omitempty"`
	SpeechURL string   `json:"speechUrl,omitempty"`
	Options   []Choice `json:"options,omitempty"`
	AnswerID  string   `json:"answerId,omitempty"`
	Phrase    string   `json:"phrase,omitempty"`
	MeaningZh string   `json:"meaningZh,omitempty"`
	Scene     string   `json:"scene,omitempty"`
	ReplyTo   string   `json:"replyTo,omitempty"`
}

type HistorySnapshot struct {
	Schema       int               `json:"schema"`
	Kind         string            `json:"kind"`
	SkillCode    string            `json:"skillCode"`
	Example      PhraseExample     `json:"example"`
	Selected     string            `json:"selected"`
	ResponseKind string            `json:"responseKind"`
	MediaSHA256  map[string]string `json:"mediaSHA256,omitempty"`
}

type PlanExample struct {
	Example      PhraseExample
	Selected     string
	ResponseKind string
}

var kinds = map[string]string{
	"listen_zh": "listen_zh",
	"listen_en": "listen_en",
	"scene":     "scene",
	"reply":     "reply",
}

func SkillForKind(kind string) string { return kinds[kind] }

func KindForSkill(code string) string {
	if kinds[code] != "" {
		return code
	}
	return ""
}

func NeedsSpeech(kind string) bool {
	return kind == "listen_zh" || kind == "listen_en" || kind == "reply"
}

// AttemptSelected returns the stable input for one attempt.
func AttemptSelected(attemptSelected, picks string) string {
	if id := stableInput(attemptSelected); id != "" {
		return id
	}
	return stableInput(picks)
}

func stableInput(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		return ""
	}
	if strings.Contains(value, ",") {
		return ""
	}
	return value
}

func Validate(e PhraseExample) error {
	kind := strings.TrimSpace(e.Kind)
	if kinds[kind] == "" {
		return fmt.Errorf("未知短句题型 %q", kind)
	}
	if len(e.Options) < 2 {
		return fmt.Errorf("%s 至少需要两个选项", kind)
	}
	seen := map[string]bool{}
	found := false
	for _, o := range e.Options {
		id := strings.TrimSpace(o.ID)
		label := strings.TrimSpace(o.Label)
		if id == "" || label == "" || seen[id] {
			return fmt.Errorf("%s 选项编号无效", kind)
		}
		seen[id] = true
		if id == e.AnswerID {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%s 缺少正确答案", kind)
	}
	if kind == "scene" && strings.TrimSpace(e.Prompt) == "" && strings.TrimSpace(e.Scene) == "" {
		return fmt.Errorf("什么时候说缺少场景")
	}
	if kind == "reply" && strings.TrimSpace(e.Prompt) == "" && strings.TrimSpace(e.ReplyTo) == "" {
		return fmt.Errorf("问与答缺少问句")
	}
	if NeedsSpeech(kind) && strings.TrimSpace(e.SpeechURL) == "" {
		return fmt.Errorf("%s 缺少整句读音", kind)
	}
	return nil
}

func HistoryBytes(snap HistorySnapshot) ([]byte, error) {
	if snap.Schema != 1 {
		snap.Schema = 1
	}
	if snap.SkillCode == "" {
		snap.SkillCode = SkillForKind(snap.Kind)
	}
	if snap.ResponseKind == "" {
		snap.ResponseKind = "choice"
	}
	if snap.Example.Kind == "" {
		snap.Example.Kind = snap.Kind
	}
	if err := Validate(snap.Example); err != nil {
		return nil, err
	}
	return json.Marshal(snap)
}

func PlanExampleFromSnapshot(raw, selected string) (PlanExample, error) {
	var snap HistorySnapshot
	if json.Unmarshal([]byte(raw), &snap) != nil || snap.Schema != 1 || kinds[snap.Kind] == "" {
		return PlanExample{}, fmt.Errorf("invalid snapshot")
	}
	if snap.Example.Kind == "" {
		snap.Example.Kind = snap.Kind
	}
	if err := Validate(snap.Example); err != nil {
		return PlanExample{}, err
	}
	picked := strings.TrimSpace(snap.Selected)
	if selected != "" {
		picked = selected
	}
	return PlanExample{Example: snap.Example, Selected: AttemptSelected(picked, ""), ResponseKind: "choice"}, nil
}
