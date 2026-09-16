package phrasecontent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type LiveOption struct {
	KpID  int64  `json:"kpId"`
	Label string `json:"label"`
}

type LiveQuestion struct {
	Code         string
	Stem         string
	Options      string
	Answer       string
	Speech       string
	Visual       string
	OptionOrder  string
	Phrase       string
	MeaningZh    string
	Scene        string
	ReplyTo      string
	TargetKpID   int64
	ReplyToKpID  int64
}

func SnapshotFromLiveQuestion(q LiveQuestion) (HistorySnapshot, error) {
	kind := KindForSkill(q.Code)
	if kind == "" {
		return HistorySnapshot{}, fmt.Errorf("unsupported phrase question %q", q.Code)
	}
	var opts []LiveOption
	if err := json.Unmarshal([]byte(q.Options), &opts); err != nil || len(opts) < 2 {
		return HistorySnapshot{}, fmt.Errorf("invalid phrase options")
	}
	var answer struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(q.Answer), &answer); err != nil || answer.Index < 0 || answer.Index >= len(opts) {
		return HistorySnapshot{}, fmt.Errorf("invalid phrase answer")
	}
	order, err := parseOptionOrder(q.OptionOrder, len(opts))
	if err != nil {
		return HistorySnapshot{}, err
	}
	displayed := make([]Choice, 0, len(order))
	for _, original := range order {
		option := opts[original]
		id := strings.TrimSpace(strconv.FormatInt(option.KpID, 10))
		if option.KpID == 0 {
			id = optionIDFromLabel(option.Label)
		}
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = id
		}
		if id == "" || id == "0" {
			return HistorySnapshot{}, fmt.Errorf("phrase option missing stable id")
		}
		displayed = append(displayed, Choice{ID: id, Label: label})
	}
	prompt := visualText(q.Visual)
	if kind == "scene" && prompt == "" {
		prompt = strings.TrimSpace(q.Scene)
	}
	if kind == "reply" && prompt == "" {
		prompt = strings.TrimSpace(q.ReplyTo)
	}
	speechText := strings.TrimSpace(q.Phrase)
	var speech struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(q.Speech), &speech)
	if strings.TrimSpace(speech.Text) != "" {
		speechText = strings.TrimSpace(speech.Text)
	}
	answerID := strings.TrimSpace(strconv.FormatInt(opts[answer.Index].KpID, 10))
	if opts[answer.Index].KpID == 0 {
		answerID = optionIDFromLabel(opts[answer.Index].Label)
	}
	example := PhraseExample{
		Kind:      kind,
		Stem:      strings.TrimSpace(q.Stem),
		Prompt:    prompt,
		Speech:    speechText,
		Options:   displayed,
		AnswerID:  answerID,
		Phrase:    strings.TrimSpace(q.Phrase),
		MeaningZh: strings.TrimSpace(q.MeaningZh),
		Scene:     strings.TrimSpace(q.Scene),
		ReplyTo:   strings.TrimSpace(q.ReplyTo),
	}
	if NeedsSpeech(kind) {
		kpID := q.TargetKpID
		if kind == "reply" && q.ReplyToKpID > 0 {
			kpID = q.ReplyToKpID
		}
		if kpID <= 0 {
			return HistorySnapshot{}, fmt.Errorf("%s 缺少读音知识点", kind)
		}
		example.SpeechURL = fmt.Sprintf("/api/v1/phrase/items/%d/speech.mp3", kpID)
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
		return "", fmt.Errorf("invalid phrase options")
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
	if opts[original].KpID > 0 {
		return strconv.FormatInt(opts[original].KpID, 10), nil
	}
	id := optionIDFromLabel(opts[original].Label)
	if id == "" {
		return "", fmt.Errorf("option index out of range")
	}
	return id, nil
}

func optionIDFromLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	return "label:" + label
}

func visualText(raw string) string {
	var v struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(raw), &v)
	return strings.TrimSpace(v.Text)
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
