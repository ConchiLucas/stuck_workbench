package chengyucontent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type LiveOption struct {
	ID    string `json:"id"`
	KpID  int64  `json:"kpId"`
	Label string `json:"label"`
}

type LiveQuestion struct {
	Code        string
	Stem        string
	Options     string
	Answer      string
	Speech      string
	Visual      string
	OptionOrder string
	Chengyu     string
	Pinyin      string
	Meaning     string
	Example     string
	TargetKpID  int64
}

func SnapshotFromLiveQuestion(q LiveQuestion) (HistorySnapshot, error) {
	kind := KindForSkill(q.Code)
	if kind == "" {
		return HistorySnapshot{}, fmt.Errorf("unsupported chengyu question %q", q.Code)
	}
	var opts []LiveOption
	if err := json.Unmarshal([]byte(q.Options), &opts); err != nil || len(opts) < 2 {
		return HistorySnapshot{}, fmt.Errorf("invalid chengyu options")
	}
	var answer struct {
		Index int `json:"index"`
	}
	if json.Unmarshal([]byte(q.Answer), &answer) != nil || answer.Index < 0 || answer.Index >= len(opts) {
		return HistorySnapshot{}, fmt.Errorf("invalid chengyu answer")
	}
	order, err := parseOptionOrder(q.OptionOrder, len(opts))
	if err != nil {
		return HistorySnapshot{}, err
	}
	displayed := make([]Choice, 0, len(order))
	for _, original := range order {
		option := opts[original]
		id := stableOptionID(option)
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = id
		}
		if id == "" || id == "0" {
			return HistorySnapshot{}, fmt.Errorf("chengyu option missing stable id")
		}
		displayed = append(displayed, Choice{ID: id, Label: label})
	}
	visual := parseVisual(q.Visual)
	prompt := visual.Text
	speechText := strings.TrimSpace(q.Chengyu)
	var speech struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(q.Speech), &speech)
	if strings.TrimSpace(speech.Text) != "" {
		speechText = strings.TrimSpace(speech.Text)
	}
	answerID := stableOptionID(opts[answer.Index])
	example := ChengyuExample{
		Kind:     kind,
		Stem:     strings.TrimSpace(q.Stem),
		Prompt:   prompt,
		Speech:   speechText,
		Options:  displayed,
		AnswerID: answerID,
		Chengyu:  strings.TrimSpace(q.Chengyu),
		Pinyin:   strings.TrimSpace(q.Pinyin),
		Meaning:  strings.TrimSpace(q.Meaning),
		Example:  strings.TrimSpace(q.Example),
	}
	switch kind {
	case "pick":
		if example.Prompt == "" {
			example.Prompt = example.Meaning
		}
	case "pinyin":
		if example.Prompt == "" {
			example.Prompt = example.Pinyin
		}
	case "example":
		blank, err := blankFromVisual(visual, example.Example, example.Chengyu)
		if err != nil {
			return HistorySnapshot{}, err
		}
		example.Blank = &blank
		example.Prompt = blank.Blanked
	}
	if NeedsSpeech(kind) {
		if q.TargetKpID <= 0 {
			return HistorySnapshot{}, fmt.Errorf("%s 缺少成语读音知识点", kind)
		}
		example.SpeechURL = fmt.Sprintf("/api/v1/chengyu/items/%d/speech.mp3", q.TargetKpID)
	}
	if err := Validate(example); err != nil {
		return HistorySnapshot{}, err
	}
	return HistorySnapshot{
		Schema:       1,
		Kind:         kind,
		SkillCode:    q.Code,
		Example:      example,
		ResponseKind: "choice",
	}, nil
}

func OptionIDForDisplayIndex(optionsJSON, orderRaw string, displayIndex int) (string, error) {
	var opts []LiveOption
	if err := json.Unmarshal([]byte(optionsJSON), &opts); err != nil || len(opts) == 0 {
		return "", fmt.Errorf("invalid chengyu options")
	}
	order, err := parseOptionOrder(orderRaw, len(opts))
	if err != nil {
		return "", err
	}
	if displayIndex < 0 || displayIndex >= len(order) {
		return "", fmt.Errorf("option index out of range")
	}
	original := order[displayIndex]
	if original < 0 || original >= len(opts) {
		return "", fmt.Errorf("option index out of range")
	}
	id := stableOptionID(opts[original])
	if id == "" || id == "0" {
		return "", fmt.Errorf("option index out of range")
	}
	return id, nil
}

func stableOptionID(option LiveOption) string {
	if id := strings.TrimSpace(option.ID); id != "" && id != "0" {
		return id
	}
	if option.KpID > 0 {
		return strconv.FormatInt(option.KpID, 10)
	}
	return optionIDFromLabel(option.Label)
}

func optionIDFromLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	return "label:" + label
}

type visualPayload struct {
	Kind    string `json:"kind"`
	Text    string `json:"text"`
	Full    string `json:"full"`
	Blanked string `json:"blanked"`
	Target  string `json:"target"`
	Start   int    `json:"start"`
	Length  int    `json:"length"`
}

func parseVisual(raw string) visualPayload {
	var v visualPayload
	_ = json.Unmarshal([]byte(raw), &v)
	v.Text = strings.TrimSpace(v.Text)
	v.Full = strings.TrimSpace(v.Full)
	v.Blanked = strings.TrimSpace(v.Blanked)
	v.Target = strings.TrimSpace(v.Target)
	return v
}

func blankFromVisual(v visualPayload, example, chengyu string) (Blank, error) {
	if v.Full != "" && v.Blanked != "" && v.Target != "" && v.Length > 0 {
		return Blank{Full: v.Full, Blanked: v.Blanked, Target: v.Target, Start: v.Start, Length: v.Length}, nil
	}
	full := v.Full
	if full == "" {
		full = example
	}
	target := v.Target
	if target == "" {
		target = chengyu
	}
	return FirstBlank(full, target)
}

func parseOptionOrder(raw string, n int) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		order := make([]int, n)
		for i := range order {
			order[i] = i
		}
		return order, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) != n {
		return nil, fmt.Errorf("option order length")
	}
	seen := map[int]bool{}
	order := make([]int, n)
	for i, part := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || v < 0 || v >= n || seen[v] {
			return nil, fmt.Errorf("option order out of range")
		}
		seen[v] = true
		order[i] = v
	}
	return order, nil
}
