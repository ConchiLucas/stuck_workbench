package logiccontent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

func ExampleFromPayload(payload, visual, optionOrder string, kpID int64) (LogicExample, error) {
	example, err := decodeStoredExample(payload, visual)
	if err != nil {
		return LogicExample{}, err
	}
	example.KpID = kpID
	example.GlyphVersion = firstNonEmpty(example.GlyphVersion, GlyphVersion)
	if example.ImageURLs == nil && kpID > 0 {
		example.ImageURLs = LiveImageURLs(kpID, example.Objects)
	}
	if err := applyOptionOrder(&example, optionOrder); err != nil {
		return LogicExample{}, err
	}
	if err := Validate(example); err != nil {
		return LogicExample{}, err
	}
	return example, nil
}

func decodeStoredExample(payload, visual string) (LogicExample, error) {
	var fromVisual struct {
		Example LogicExample `json:"example"`
	}
	if strings.TrimSpace(visual) != "" && json.Unmarshal([]byte(visual), &fromVisual) == nil && fromVisual.Example.Kind != "" {
		return fromVisual.Example, nil
	}
	var fromPayload struct {
		Example LogicExample `json:"example"`
		Kind    string        `json:"kind"`
		Rule    Rule          `json:"rule"`
	}
	if strings.TrimSpace(payload) != "" && json.Unmarshal([]byte(payload), &fromPayload) == nil && fromPayload.Example.Kind != "" {
		return fromPayload.Example, nil
	}
	return LogicExample{}, fmt.Errorf("素材缺少可校验规则")
}

func applyOptionOrder(e *LogicExample, order string) error {
	order = strings.TrimSpace(optionOrderSafe(order))
	if order == "" {
		return nil
	}
	ids := e.Options
	if e.Kind == "order" {
		ids = e.DisplayOrder
	}
	parts := strings.Split(order, ",")
	if len(parts) != len(ids) {
		return fmt.Errorf("选项顺序长度不匹配")
	}
	next := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" {
			return fmt.Errorf("选项顺序空值")
		}
		if seen[id] {
			return fmt.Errorf("选项顺序重复")
		}
		ok := false
		for _, existing := range ids {
			if existing == id {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("选项顺序含未知 id")
		}
		seen[id] = true
		next = append(next, id)
	}
	if e.Kind == "order" {
		e.DisplayOrder = next
	} else {
		e.Options = next
	}
	return nil
}

func optionOrderSafe(v string) string {
	return strings.ReplaceAll(v, "|", ",")
}

func SnapshotJSON(e LogicExample) string {
	snap := ExampleToSnapshot(e)
	b, _ := json.Marshal(snap)
	return string(b)
}

func ParseSnapshot(raw string) (HistorySnapshot, error) {
	var s HistorySnapshot
	if strings.TrimSpace(raw) == "" {
		return s, fmt.Errorf("空快照")
	}
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return s, err
	}
	if s.Kind == "" && s.Prompt != "" {
		return s, fmt.Errorf("快照缺少题型")
	}
	ex := SnapshotToExample(s)
	if err := Validate(ex); err != nil {
		return s, err
	}
	return s, nil
}

func EncodeSelected(in AnswerInput) string {
	if in.SelectedID == "" && len(in.Sequence) == 0 && len(in.Rejected) == 0 {
		return ""
	}
	b, _ := json.Marshal(in)
	return string(b)
}

func DecodeSelected(raw string) AnswerInput {
	var in AnswerInput
	_ = json.Unmarshal([]byte(raw), &in)
	return in
}

type Converted struct {
	Example      LogicExample
	Input        AnswerInput
	Selected     string
	ResponseKind string
}

func ShuffleExample(e LogicExample, seed int64) LogicExample {
	if e.Kind == "order" {
		e.DisplayOrder = ShuffleIDs(e.DisplayOrder, seed)
		return e
	}
	e.Options = ShuffleIDs(e.Options, seed)
	return e
}

func HistoryBytes(s HistorySnapshot) ([]byte, error) {
	s.Schema = SnapshotSchema
	return json.Marshal(s)
}

func PlanExampleFromSnapshot(raw, picks string) (Converted, error) {
	snap, err := ParseSnapshot(raw)
	if err != nil {
		return Converted{}, err
	}
	example := SnapshotToExample(snap)
	in := DecodeSelected(picks)
	selected := picks
	if example.Kind == "order" {
		if len(in.Sequence) == 0 && strings.TrimSpace(picks) != "" && !strings.HasPrefix(strings.TrimSpace(picks), "{") {
			in.Sequence = strings.Split(picks, ",")
		}
		selected = EncodeSelected(in)
	} else if in.SelectedID != "" {
		selected = in.SelectedID
	}
	kind := "choice"
	if example.Kind == "order" {
		kind = "order"
	}
	return Converted{Example: example, Input: in, Selected: selected, ResponseKind: kind}, nil
}

func Facts(e LogicExample, in AnswerInput) []string {
	var facts []string
	if e.Kind == "order" {
		if len(in.Sequence) > 0 {
			facts = append(facts, "孩子排出了「"+joinCaptions(e, in.Sequence)+"」。")
		}
		facts = append(facts, "正确顺序是「"+joinCaptions(e, e.CorrectSequence)+"」。")
		for _, tap := range in.Rejected {
			o, _ := ObjectByID(e.Objects, tap.ID)
			facts = append(facts, fmt.Sprintf("第 %d 位误点了「%s」。", tap.AtIndex+1, o.Caption))
		}
		return facts
	}
	if in.SelectedID != "" {
		got, _ := ObjectByID(e.Objects, in.SelectedID)
		want, _ := ObjectByID(e.Objects, e.AnswerID)
		if in.SelectedID == e.AnswerID {
			facts = append(facts, "孩子选了「"+got.Caption+"」。")
		} else {
			facts = append(facts, "孩子选了「"+got.Caption+"」，正确答案是「"+want.Caption+"」。")
		}
	}
	return facts
}

func joinCaptions(e LogicExample, ids []string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		o, _ := ObjectByID(e.Objects, id)
		parts = append(parts, o.Caption)
	}
	return strings.Join(parts, " → ")
}

func SourceHash(parts ...string) string {
	return fmt.Sprintf("%x", sha256Sum(strings.Join(parts, "\n")))
}

func sha256Sum(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}
