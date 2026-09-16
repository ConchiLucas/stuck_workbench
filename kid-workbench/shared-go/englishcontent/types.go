package englishcontent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Choice struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Picture string `json:"picture,omitempty"`
}

type EnglishExample struct {
	Kind      string   `json:"kind"`
	Prompt    string   `json:"prompt,omitempty"`
	Passage   string   `json:"passage,omitempty"`
	Speech    string   `json:"speech,omitempty"`
	SpeechURL string   `json:"speechUrl,omitempty"`
	Cue       string   `json:"cue,omitempty"`
	Options   []Choice `json:"options,omitempty"`
	AnswerID  string   `json:"answerId,omitempty"`
	Bank      []string `json:"bank,omitempty"`
	Answer    string   `json:"answer,omitempty"`
	Wide      bool     `json:"wide,omitempty"`
}

type HistorySnapshot struct {
	Schema       int               `json:"schema"`
	Kind         string             `json:"kind"`
	SkillCode    string            `json:"skillCode"`
	Example      EnglishExample    `json:"example"`
	Selected     string            `json:"selected"`
	ResponseKind string            `json:"responseKind"`
	MediaSHA256  map[string]string `json:"mediaSHA256,omitempty"`
}

type PlanExample struct {
	Example      EnglishExample
	Selected     string
	ResponseKind string
}

var kinds = map[string]string{
	"audio-choice": "listen",
	"image-text":   "picture",
	"card-builder": "build",
	"input-gap":    "type",
	"reading-qa":   "read",
}

func SkillForKind(kind string) string { return kinds[kind] }

func KindForSkill(code string) string {
	for kind, skill := range kinds {
		if skill == code {
			return kind
		}
	}
	return ""
}

// AttemptSelected returns the stable input for one attempt.
// Accumulated plan_items.picks JSON cannot be attributed to a single attempt.
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
	return value
}

func Validate(e EnglishExample) error {
	kind := strings.TrimSpace(e.Kind)
	if kinds[kind] == "" {
		return fmt.Errorf("未知英语题型 %q", e.Kind)
	}
	switch kind {
	case "audio-choice", "image-text", "reading-qa":
		if len(e.Options) < 2 {
			return fmt.Errorf("%s 至少需要两个选项", kind)
		}
		seen := map[string]bool{}
		found := false
		for _, o := range e.Options {
			if strings.TrimSpace(o.ID) == "" || seen[o.ID] {
				return fmt.Errorf("%s 选项编号无效", kind)
			}
			seen[o.ID] = true
			if o.ID == e.AnswerID {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("%s 缺少正确答案", kind)
		}
		if kind == "reading-qa" && strings.TrimSpace(e.Passage) == "" {
			return fmt.Errorf("读一读缺少短文")
		}
		if kind == "image-text" && strings.TrimSpace(e.Prompt) == "" {
			return fmt.Errorf("看图选词缺少题干")
		}
	case "card-builder":
		if len(e.Bank) < 2 || strings.TrimSpace(e.Answer) == "" {
			return fmt.Errorf("组句子缺少词卡或答案句")
		}
		if strings.TrimSpace(e.SpeechURL) == "" {
			return fmt.Errorf("组句子缺少整句读音")
		}
	case "input-gap":
		if strings.TrimSpace(e.Answer) == "" {
			return fmt.Errorf("写单词缺少答案")
		}
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
		snap.ResponseKind = responseKind(snap.Kind)
	}
	if err := Validate(snap.Example); err != nil {
		return nil, err
	}
	if snap.Example.Kind == "" {
		snap.Example.Kind = snap.Kind
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
	kind := snap.ResponseKind
	if kind == "" {
		kind = responseKind(snap.Kind)
	}
	return PlanExample{Example: snap.Example, Selected: picked, ResponseKind: kind}, nil
}

func responseKind(kind string) string {
	switch kind {
	case "card-builder":
		return "order"
	case "input-gap":
		return "input"
	default:
		return "choice"
	}
}
