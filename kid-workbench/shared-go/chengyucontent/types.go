package chengyucontent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Blank struct {
	Full    string `json:"full"`
	Blanked string `json:"blanked"`
	Target  string `json:"target"`
	Start   int    `json:"start"`
	Length  int    `json:"length"`
}

type ChengyuExample struct {
	Kind      string   `json:"kind"`
	Stem      string   `json:"stem,omitempty"`
	Prompt    string   `json:"prompt,omitempty"`
	Speech    string   `json:"speech,omitempty"`
	SpeechURL string   `json:"speechUrl,omitempty"`
	Options   []Choice `json:"options,omitempty"`
	AnswerID  string   `json:"answerId,omitempty"`
	Chengyu   string   `json:"chengyu,omitempty"`
	Pinyin    string   `json:"pinyin,omitempty"`
	Meaning   string   `json:"meaning,omitempty"`
	Example   string   `json:"example,omitempty"`
	Blank     *Blank   `json:"blank,omitempty"`
}

type HistorySnapshot struct {
	Schema       int               `json:"schema"`
	Kind         string            `json:"kind"`
	SkillCode    string            `json:"skillCode"`
	Example      ChengyuExample    `json:"example"`
	Selected     string            `json:"selected"`
	ResponseKind string            `json:"responseKind"`
	MediaSHA256  map[string]string `json:"mediaSHA256,omitempty"`
}

type PlanExample struct {
	Example      ChengyuExample
	Selected     string
	ResponseKind string
}

var kinds = map[string]string{
	"meaning": "meaning",
	"pick":    "pick",
	"pinyin":  "pinyin",
	"example": "example",
}

func SkillForKind(kind string) string { return kinds[kind] }

func KindForSkill(code string) string {
	if kinds[code] != "" {
		return code
	}
	return ""
}

func NeedsSpeech(kind string) bool { return kind == "meaning" }

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

func FirstBlank(full, target string) (Blank, error) {
	full = strings.TrimSpace(full)
	target = strings.TrimSpace(target)
	if full == "" || target == "" {
		return Blank{}, fmt.Errorf("例句或成语为空")
	}
	runes := []rune(full)
	word := []rune(target)
	start := -1
	for i := 0; i+len(word) <= len(runes); i++ {
		if string(runes[i:i+len(word)]) == target {
			start = i
			break
		}
	}
	if start < 0 {
		return Blank{}, fmt.Errorf("例句未出现成语")
	}
	blanked := string(runes[:start]) + "____" + string(runes[start+len(word):])
	return Blank{Full: full, Blanked: blanked, Target: target, Start: start, Length: len(word)}, nil
}

func Validate(e ChengyuExample) error {
	kind := strings.TrimSpace(e.Kind)
	if kinds[kind] == "" {
		return fmt.Errorf("未知成语题型 %q", kind)
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
	switch kind {
	case "pick":
		if strings.TrimSpace(e.Prompt) == "" && strings.TrimSpace(e.Meaning) == "" {
			return fmt.Errorf("选成语缺少释义")
		}
	case "pinyin":
		if strings.TrimSpace(e.Prompt) == "" && strings.TrimSpace(e.Pinyin) == "" {
			return fmt.Errorf("看拼音缺少拼音")
		}
	case "example":
		if e.Blank == nil || strings.TrimSpace(e.Blank.Blanked) == "" || strings.TrimSpace(e.Blank.Full) == "" {
			return fmt.Errorf("看句子缺少设空结构")
		}
		if e.Blank.Length <= 0 || e.Blank.Start < 0 {
			return fmt.Errorf("看句子设空位置无效")
		}
	}
	if NeedsSpeech(kind) && strings.TrimSpace(e.SpeechURL) == "" {
		return fmt.Errorf("%s 缺少成语读音", kind)
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
