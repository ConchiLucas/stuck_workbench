package poemcontent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type LiveQuestion struct {
	Code        string
	Stem        string
	Options     string
	Answer      string
	Visual      string
	Speech      string
	OptionOrder string
	Payload     string
	TargetKpID  int64
	Work        Work
}

func SnapshotFromLiveQuestion(q LiveQuestion) (HistorySnapshot, error) {
	kind := KindForSkill(q.Code)
	if kind == "" {
		return HistorySnapshot{}, fmt.Errorf("unsupported poem question %q", q.Code)
	}
	example, err := exampleFromStored(kind, q)
	if err != nil {
		return HistorySnapshot{}, err
	}
	if err := Validate(example); err != nil {
		return HistorySnapshot{}, err
	}
	return HistorySnapshot{
		Schema: Schema, Kind: kind, SkillCode: SkillForKind(q.Code),
		Example: example, ResponseKind: kind,
	}, nil
}

func exampleFromStored(kind string, q LiveQuestion) (PoemExample, error) {
	work := q.Work
	if work.WorkID == "" {
		work = ParseWork(q.TargetKpID, "", "", q.Payload)
	}
	switch kind {
	case KindTitle:
		return choiceFromStored(kind, q, work)
	case KindFill:
		ex, err := choiceFromStored(kind, q, work)
		if err != nil {
			return PoemExample{}, err
		}
		applyVisual(&ex, q.Visual)
		if len(ex.GapIndexes) == 0 || ex.SourceLine == "" {
			return PoemExample{}, fmt.Errorf("invalid fill snapshot")
		}
		return ex, nil
	case KindCouplet:
		ex, err := choiceFromStored(kind, q, work)
		if err != nil {
			return PoemExample{}, err
		}
		applyVisual(&ex, q.Visual)
		if ex.UpperLineID == "" || ex.NextLineID == "" {
			ex.UpperLineID = ex.LineID
			ex.NextLineID = ex.AnswerID
		}
		return ex, nil
	case KindRecite:
		return reciteFromStored(q, work)
	}
	return PoemExample{}, fmt.Errorf("unsupported poem question %q", kind)
}

func choiceFromStored(kind string, q LiveQuestion, work Work) (PoemExample, error) {
	var opts []Choice
	if json.Unmarshal([]byte(q.Options), &opts) != nil || len(opts) < 2 {
		return PoemExample{}, fmt.Errorf("invalid poem options")
	}
	ids := make([]string, len(opts))
	for i, o := range opts {
		ids[i] = strings.TrimSpace(o.ID)
		if ids[i] == "" {
			ids[i] = "label:" + o.Label
			opts[i].ID = ids[i]
		}
	}
	order, err := parseIDOrder(q.OptionOrder, ids)
	if err != nil {
		return PoemExample{}, err
	}
	opts = reorderChoices(opts, order)
	answerID, err := parseChoiceAnswer(q.Answer, opts)
	if err != nil {
		return PoemExample{}, err
	}
	ex := PoemExample{
		Kind: kind, Prompt: strings.TrimSpace(q.Stem), Options: opts, AnswerID: answerID,
		OptionOrder: order, WorkID: work.WorkID, WorkTitle: work.Title, Author: work.Author,
		Dynasty: work.Dynasty, Edition: work.Edition,
	}
	applyVisual(&ex, q.Visual)
	if ex.Line == "" {
		ex.Line = payloadLine(q.Payload)
	}
	attachSpeechFromStored(&ex, q)
	return ex, nil
}

func reciteFromStored(q LiveQuestion, work Work) (PoemExample, error) {
	var body struct {
		Items []Node   `json:"items"`
		Order []string `json:"displayOrder"`
	}
	if json.Unmarshal([]byte(q.Options), &body) != nil || len(body.Items) < 2 {
		return PoemExample{}, fmt.Errorf("invalid recite options")
	}
	var answer struct {
		Order []string `json:"order"`
	}
	if json.Unmarshal([]byte(q.Answer), &answer) != nil || len(answer.Order) == 0 {
		return PoemExample{}, fmt.Errorf("invalid recite answer")
	}
	display := body.Order
	if q.OptionOrder != "" {
		display = splitIDs(q.OptionOrder)
	}
	if len(display) == 0 {
		display = nodeIDs(body.Items)
	}
	body.Items = reorderNodes(body.Items, display)
	line := ""
	if len(work.Lines) > 0 {
		line = work.Lines[0].Text
	}
	ex := PoemExample{
		Kind: KindRecite, Prompt: strings.TrimSpace(q.Stem), Line: line, WorkID: work.WorkID,
		WorkTitle: work.Title, Author: work.Author, Dynasty: work.Dynasty, Edition: work.Edition,
		SequenceItems: body.Items, SequenceDisplayOrder: display, CorrectSequence: answer.Order,
		AudioMissingReason: "排顺序不按正确句序朗读，以免泄露答案。",
	}
	attachSpeechFromStored(&ex, q)
	return ex, nil
}

func attachSpeechFromStored(ex *PoemExample, q LiveQuestion) {
	kind := KindForSkill(ex.Kind)
	if !NeedsSpeech(kind) {
		if strings.TrimSpace(ex.AudioMissingReason) == "" {
			switch kind {
			case KindFill:
				ex.AudioMissingReason = "补字题不朗读缺字，以免直接说出答案。"
			case KindRecite:
				ex.AudioMissingReason = "排顺序不按正确句序朗读，以免泄露答案。"
			}
		}
		ex.SpeechURL = ""
		return
	}
	var speech struct {
		URL  string `json:"url"`
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(q.Speech), &speech)
	if strings.TrimSpace(speech.URL) != "" {
		ex.SpeechURL = speech.URL
		ex.SpeechText = speech.Text
	}
	if strings.TrimSpace(ex.SpeechURL) == "" {
		attachLiveSpeech(ex, q.TargetKpID)
	}
}

func applyVisual(ex *PoemExample, raw string) {
	var v struct {
		Kind        string `json:"kind"`
		Text        string `json:"text"`
		LineID      string `json:"lineId"`
		SourceLine  string `json:"sourceLine"`
		GapIndexes  []int  `json:"gapIndexes"`
		WorkID      string `json:"workId"`
		UpperLineID string `json:"upperLineId"`
		NextLineID  string `json:"nextLineId"`
	}
	_ = json.Unmarshal([]byte(raw), &v)
	if ex.Line == "" {
		ex.Line = v.Text
	}
	if ex.LineID == "" {
		ex.LineID = v.LineID
	}
	if ex.SourceLine == "" {
		ex.SourceLine = v.SourceLine
	}
	if len(ex.GapIndexes) == 0 {
		ex.GapIndexes = v.GapIndexes
	}
	if ex.WorkID == "" {
		ex.WorkID = v.WorkID
	}
	if ex.UpperLineID == "" {
		ex.UpperLineID = v.UpperLineID
	}
	if ex.NextLineID == "" {
		ex.NextLineID = v.NextLineID
	}
}

func parseChoiceAnswer(raw string, opts []Choice) (string, error) {
	var as struct {
		ID    string `json:"id"`
		Index int    `json:"index"`
	}
	if json.Unmarshal([]byte(raw), &as) != nil {
		return "", fmt.Errorf("invalid poem answer")
	}
	if strings.TrimSpace(as.ID) != "" {
		return strings.TrimSpace(as.ID), nil
	}
	if as.Index < 0 || as.Index >= len(opts) {
		return "", fmt.Errorf("invalid poem answer")
	}
	return opts[as.Index].ID, nil
}

func parseIDOrder(raw string, ids []string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return append([]string{}, ids...), nil
	}
	if strings.Contains(raw, ",") && !strings.Contains(raw, ":") {
		parts := strings.Split(raw, ",")
		if len(parts) == len(ids) {
			out := make([]string, 0, len(parts))
			seen := map[int]bool{}
			for _, p := range parts {
				v, err := strconv.Atoi(strings.TrimSpace(p))
				if err != nil || v < 0 || v >= len(ids) || seen[v] {
					return nil, fmt.Errorf("option order out of range")
				}
				seen[v] = true
				out = append(out, ids[v])
			}
			return out, nil
		}
	}
	out := splitIDs(raw)
	if len(out) != len(ids) {
		return nil, fmt.Errorf("option order length")
	}
	return out, nil
}

func splitIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func nodeIDs(nodes []Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}

func payloadLine(raw string) string {
	w := ParseWork(0, "", "", raw)
	if len(w.Lines) > 0 {
		return w.Lines[0].Text
	}
	return ""
}

func OrderFromExample(e PoemExample) string {
	switch KindForSkill(e.Kind) {
	case KindRecite:
		return strings.Join(e.SequenceDisplayOrder, ",")
	default:
		return strings.Join(e.OptionOrder, ",")
	}
}

func SourceHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	hex := fmt.Sprintf("%x", sum)
	if len(hex) > 16 {
		return hex[:16]
	}
	return hex
}
