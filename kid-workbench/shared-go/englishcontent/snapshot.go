package englishcontent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type LiveOption struct {
	KpID      int64  `json:"kpId"`
	Label     string `json:"label"`
	AssetKind string `json:"assetKind"`
}

type AssetKey struct {
	Speech string
	Sense  string
}

type LiveQuestion struct {
	Code        string
	Stem        string
	Options     string
	Answer      string
	Speech      string
	OptionOrder string
	Word        string
	TargetKpID  int64
	Meanings    map[int64]string
	AssetKeys   map[int64]AssetKey
}

func SnapshotFromLiveQuestion(q LiveQuestion) (HistorySnapshot, error) {
	kind := KindForSkill(q.Code)
	if kind == "" {
		return HistorySnapshot{}, fmt.Errorf("unsupported english question %q", q.Code)
	}
	var opts []LiveOption
	if err := json.Unmarshal([]byte(q.Options), &opts); err != nil || len(opts) < 2 {
		return HistorySnapshot{}, fmt.Errorf("invalid english options")
	}
	var answer struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(q.Answer), &answer); err != nil || answer.Index < 0 || answer.Index >= len(opts) {
		return HistorySnapshot{}, fmt.Errorf("invalid english answer")
	}
	order, err := parseOptionOrder(q.OptionOrder, len(opts))
	if err != nil {
		return HistorySnapshot{}, err
	}
	displayed := make([]Choice, 0, len(order))

	for _, original := range order {
		option := opts[original]
		id := strconv.FormatInt(option.KpID, 10)
		label := strings.TrimSpace(q.Meanings[option.KpID])
		if label == "" {
			label = strings.TrimSpace(option.Label)
		}
		if label == "" {
			label = id
		}
		choice := Choice{ID: id, Label: label}
		if kind == "image-text" || option.AssetKind == "sense" {
			picture := fmt.Sprintf("/api/v1/english/words/%d/sense.png", option.KpID)
			choice.Picture = picture

		}
		displayed = append(displayed, choice)
	}
	speechText := strings.TrimSpace(q.Word)
	var speech struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(q.Speech), &speech)
	if strings.TrimSpace(speech.Text) != "" {
		speechText = strings.TrimSpace(speech.Text)
	}
	prompt := strings.TrimSpace(q.Stem)
	if prompt == "" && kind == "image-text" {
		prompt = "看图选词"
	}
	example := EnglishExample{
		Kind:     kind,
		Prompt:   prompt,
		Speech:   speechText,
		Options:  displayed,
		AnswerID: strconv.FormatInt(opts[answer.Index].KpID, 10),
	}
	if q.TargetKpID > 0 && (kind == "audio-choice" || kind == "image-text") {
		example.SpeechURL = fmt.Sprintf("/api/v1/english/words/%d/speech.mp3", q.TargetKpID)

	}
	if err := Validate(example); err != nil {
		return HistorySnapshot{}, err
	}
	return HistorySnapshot{
		Schema:       1,
		Kind:         kind,
		SkillCode:    q.Code,
		Example:      example,
		ResponseKind: responseKind(kind),
	}, nil
}

func OptionIDForDisplayIndex(optionsJSON, orderRaw string, displayIndex int) (string, error) {
	var opts []LiveOption
	if err := json.Unmarshal([]byte(optionsJSON), &opts); err != nil || len(opts) == 0 {
		return "", fmt.Errorf("invalid english options")
	}
	order, err := parseOptionOrder(orderRaw, len(opts))
	if err != nil {
		return "", err
	}
	if displayIndex < 0 || displayIndex >= len(order) {
		return "", fmt.Errorf("option index out of range")
	}
	original := order[displayIndex]
	if original < 0 || original >= len(opts) || opts[original].KpID == 0 {
		return "", fmt.Errorf("option index out of range")
	}
	return strconv.FormatInt(opts[original].KpID, 10), nil
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
