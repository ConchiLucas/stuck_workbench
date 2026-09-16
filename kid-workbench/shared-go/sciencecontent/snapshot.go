package sciencecontent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type LiveQuestion struct {
	Code          string
	Stem          string
	Options       string
	Answer        string
	Visual        string
	OptionOrder   string
	Payload       string
	TargetKpID    int64
	HabitatByCode map[string]int64
}

func SnapshotFromLiveQuestion(q LiveQuestion) (HistorySnapshot, error) {
	kind := KindForSkill(q.Code)
	if kind == "" {
		return HistorySnapshot{}, fmt.Errorf("unsupported science question %q", q.Code)
	}
	example, err := exampleFromStored(kind, q)
	if err != nil {
		return HistorySnapshot{}, err
	}
	if err := Validate(example); err != nil {
		return HistorySnapshot{}, err
	}
	return HistorySnapshot{
		Schema:       1,
		Kind:         kind,
		SkillCode:    SkillForKind(q.Code),
		Example:      example,
		ResponseKind: kind,
	}, nil
}

func exampleFromStored(kind string, q LiveQuestion) (ScienceExample, error) {
	switch kind {
	case KindChoice:
		return choiceFromStored(q)
	case KindMatch:
		return matchFromStored(q)
	case KindSequence:
		return sequenceFromStored(q)
	case KindLabel:
		return labelFromStored(q)
	}
	return ScienceExample{}, fmt.Errorf("unsupported science question %q", kind)
}

func choiceFromStored(q LiveQuestion) (ScienceExample, error) {
	var opts []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Icon  string `json:"icon"`
	}
	if err := json.Unmarshal([]byte(q.Options), &opts); err != nil || len(opts) < 2 {
		return ScienceExample{}, fmt.Errorf("invalid science options")
	}
	order, err := parseIDOrder(q.OptionOrder, optionIDs(opts))
	if err != nil {
		return ScienceExample{}, err
	}
	displayed := make([]Choice, 0, len(order))
	byID := map[string]Choice{}
	for _, o := range opts {
		id := strings.TrimSpace(o.ID)
		if id == "" {
			id = OptionIDFromLabel(o.Label)
		}
		byID[id] = Choice{ID: id, Label: strings.TrimSpace(o.Label), Icon: o.Icon}
	}
	for _, id := range order {
		c, ok := byID[id]
		if !ok {
			return ScienceExample{}, fmt.Errorf("option order missing id")
		}
		displayed = append(displayed, c)
	}
	answerID, err := parseChoiceAnswer(q.Answer, opts)
	if err != nil {
		return ScienceExample{}, err
	}
	visual := visualIcon(q.Visual)
	return ScienceExample{
		Kind: KindChoice, Prompt: strings.TrimSpace(q.Stem), Icon: visual.Icon, Visual: visual.Emoji,
		Options: displayed, AnswerID: answerID, OptionOrder: order, Explanation: payloadText(q.Payload, "explanation"),
		Tip: payloadText(q.Payload, "tip"), Rationale: payloadText(q.Payload, "rationale"),
	}, nil
}

func matchFromStored(q LiveQuestion) (ScienceExample, error) {
	var body struct {
		Sources     []Node            `json:"sources"`
		Targets     []Node            `json:"targets"`
		Answers     map[string]string `json:"answers"`
		SourceGroup string            `json:"sourceGroup"`
		TargetGroup string            `json:"targetGroup"`
	}
	if json.Unmarshal([]byte(q.Options), &body) != nil || len(body.Sources) < 2 {
		return ScienceExample{}, fmt.Errorf("invalid match options")
	}
	var answer struct {
		Pairs map[string]string `json:"pairs"`
	}
	_ = json.Unmarshal([]byte(q.Answer), &answer)
	pairs := body.Answers
	if len(answer.Pairs) > 0 {
		pairs = answer.Pairs
	}
	srcOrder := nodeIDs(body.Sources)
	tgtOrder := nodeIDs(body.Targets)
	if q.OptionOrder != "" {
		if parts := strings.Split(q.OptionOrder, "|"); len(parts) == 2 {
			srcOrder = splitIDs(parts[0])
			tgtOrder = splitIDs(parts[1])
			body.Sources = reorderNodes(body.Sources, srcOrder)
			body.Targets = reorderNodes(body.Targets, tgtOrder)
		}
	}
	attachHabitatImages(body.Targets, q.HabitatByCode)
	return ScienceExample{
		Kind: KindMatch, Prompt: strings.TrimSpace(q.Stem),
		MatchSources: body.Sources, MatchTargets: body.Targets, MatchAnswers: pairs,
		MatchSourceOrder: srcOrder, MatchTargetOrder: tgtOrder,
		MatchSourceGroup: body.SourceGroup, MatchTargetGroup: body.TargetGroup,
		Explanation: payloadText(q.Payload, "explanation"), Tip: payloadText(q.Payload, "tip"),
		Rationale: payloadText(q.Payload, "rationale"),
	}, nil
}

func sequenceFromStored(q LiveQuestion) (ScienceExample, error) {
	var body struct {
		Items []Node   `json:"items"`
		Order []string `json:"displayOrder"`
	}
	if json.Unmarshal([]byte(q.Options), &body) != nil || len(body.Items) < 2 {
		return ScienceExample{}, fmt.Errorf("invalid sequence options")
	}
	var answer struct {
		Order   []string `json:"order"`
		StartID string   `json:"startId"`
		Loop    bool     `json:"loop"`
	}
	if json.Unmarshal([]byte(q.Answer), &answer) != nil || len(answer.Order) == 0 {
		return ScienceExample{}, fmt.Errorf("invalid sequence answer")
	}
	display := body.Order
	if q.OptionOrder != "" {
		display = splitIDs(q.OptionOrder)
	}
	if len(display) == 0 {
		display = nodeIDs(body.Items)
	}
	body.Items = reorderNodes(body.Items, display)
	return ScienceExample{
		Kind: KindSequence, Prompt: strings.TrimSpace(q.Stem), SequenceItems: body.Items,
		SequenceDisplayOrder: display, CorrectSequence: answer.Order, SequenceLoop: answer.Loop,
		SequenceStartID: answer.StartID, Explanation: payloadText(q.Payload, "explanation"),
		Tip: payloadText(q.Payload, "tip"), Rationale: payloadText(q.Payload, "rationale"),
	}, nil
}

func labelFromStored(q LiveQuestion) (ScienceExample, error) {
	var body struct {
		DiagramKey     string        `json:"diagramKey"`
		DiagramVersion int           `json:"diagramVersion"`
		Targets        []LabelTarget `json:"targets"`
		Tokens         []Node        `json:"tokens"`
	}
	if json.Unmarshal([]byte(q.Options), &body) != nil {
		return ScienceExample{}, fmt.Errorf("invalid label options")
	}
	var answer struct {
		Labels map[string]string `json:"labels"`
	}
	if json.Unmarshal([]byte(q.Answer), &answer) != nil || len(answer.Labels) == 0 {
		return ScienceExample{}, fmt.Errorf("invalid label answer")
	}
	if body.DiagramKey == "" {
		body.DiagramKey = DiagramPlant
	}
	if body.DiagramVersion == 0 {
		body.DiagramVersion = DiagramVer
	}
	return ScienceExample{
		Kind: KindLabel, Prompt: strings.TrimSpace(q.Stem), DiagramKey: body.DiagramKey,
		DiagramVersion: body.DiagramVersion, LabelTargets: body.Targets, LabelTokens: body.Tokens,
		LabelAnswers: answer.Labels, Explanation: payloadText(q.Payload, "explanation"),
		Tip: payloadText(q.Payload, "tip"), Rationale: payloadText(q.Payload, "rationale"),
	}, nil
}

func parseChoiceAnswer(raw string, opts []struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}) (string, error) {
	var asID struct {
		ID    string `json:"id"`
		Index int    `json:"index"`
	}
	if json.Unmarshal([]byte(raw), &asID) != nil {
		return "", fmt.Errorf("invalid science answer")
	}
	if strings.TrimSpace(asID.ID) != "" {
		return strings.TrimSpace(asID.ID), nil
	}
	if asID.Index < 0 || asID.Index >= len(opts) {
		return "", fmt.Errorf("invalid science answer")
	}
	id := strings.TrimSpace(opts[asID.Index].ID)
	if id == "" {
		id = OptionIDFromLabel(opts[asID.Index].Label)
	}
	return id, nil
}

func optionIDs(opts []struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}) []string {
	ids := make([]string, len(opts))
	for i, o := range opts {
		id := strings.TrimSpace(o.ID)
		if id == "" {
			id = OptionIDFromLabel(o.Label)
		}
		ids[i] = id
	}
	return ids
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

func OptionIDFromLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	return "label:" + label
}

func visualIcon(raw string) struct{ Icon, Emoji string } {
	var v struct {
		Kind  string `json:"kind"`
		Key   string `json:"key"`
		Emoji string `json:"emoji"`
		Text  string `json:"text"`
	}
	_ = json.Unmarshal([]byte(raw), &v)
	icon := strings.TrimSpace(v.Key)
	if icon == "" && v.Kind == "icon" {
		icon = v.Text
	}
	return struct{ Icon, Emoji string }{Icon: icon, Emoji: v.Emoji}
}

func payloadText(raw, key string) string {
	var p map[string]any
	if json.Unmarshal([]byte(raw), &p) != nil {
		return ""
	}
	v, _ := p[key].(string)
	return strings.TrimSpace(v)
}

func nodeIDs(nodes []Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
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

func reorderNodes(nodes []Node, order []string) []Node {
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	out := make([]Node, 0, len(order))
	for _, id := range order {
		if n, ok := byID[id]; ok {
			out = append(out, n)
		}
	}
	return out
}

func attachHabitatImages(targets []Node, habitatByCode map[string]int64) {
	for i := range targets {
		if targets[i].ImageURL != "" {
			continue
		}
		if kpID := habitatByCode[targets[i].ID]; kpID > 0 {
			targets[i].ImageURL = fmt.Sprintf("/api/v1/science/items/%d/diagram.png", kpID)
		}
	}
}

func ShuffleIDs(ids []string, seed int64) []string {
	out := append([]string{}, ids...)
	// deterministic Fisher–Yates using seed
	x := uint64(seed)
	if x == 0 {
		x = 1
	}
	for i := len(out) - 1; i > 0; i-- {
		x = x*6364136223846793005 + 1
		j := int(x % uint64(i+1))
		out[i], out[j] = out[j], out[i]
	}
	return out
}
