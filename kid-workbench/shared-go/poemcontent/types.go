package poemcontent

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const (
	KindTitle   = "title"
	KindFill    = "fill"
	KindCouplet = "couplet"
	KindRecite  = "recite"
	Schema      = 1
	Edition     = "通行小学课文"
)

var PoemSkills = []string{KindTitle, KindFill, KindCouplet, KindRecite}

type Line struct {
	ID   string `json:"id"`
	Ord  int    `json:"ord"`
	Text string `json:"text"`
}

type Work struct {
	WorkID   string `json:"workId"`
	KpID     int64  `json:"kpId,omitempty"`
	Code     string `json:"code,omitempty"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Dynasty  string `json:"dynasty,omitempty"`
	Edition  string `json:"edition,omitempty"`
	Citation string `json:"citation,omitempty"`
	Lines    []Line `json:"lines"`
}

type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type PoemExample struct {
	Kind                 string   `json:"kind"`
	Prompt               string   `json:"prompt"`
	Line                 string   `json:"line,omitempty"`
	SpeechURL            string   `json:"speechUrl,omitempty"`
	SpeechText           string   `json:"speechText,omitempty"`
	AudioMissingReason   string   `json:"audioMissingReason,omitempty"`
	Options              []Choice `json:"options,omitempty"`
	AnswerID             string   `json:"answerId,omitempty"`
	OptionOrder          []string `json:"optionOrder,omitempty"`
	WorkID               string   `json:"workId,omitempty"`
	WorkTitle            string   `json:"workTitle,omitempty"`
	Author               string   `json:"author,omitempty"`
	Dynasty              string   `json:"dynasty,omitempty"`
	Edition              string   `json:"edition,omitempty"`
	LineID               string   `json:"lineId,omitempty"`
	SourceLine           string   `json:"sourceLine,omitempty"`
	GapIndexes           []int    `json:"gapIndexes,omitempty"`
	UpperLineID          string   `json:"upperLineId,omitempty"`
	NextLineID           string   `json:"nextLineId,omitempty"`
	SequenceItems        []Node   `json:"sequenceItems,omitempty"`
	SequenceDisplayOrder []string `json:"sequenceDisplayOrder,omitempty"`
	CorrectSequence      []string `json:"correctSequence,omitempty"`
	Explanation          string   `json:"explanation,omitempty"`
	Tip                  string   `json:"tip,omitempty"`
}

type AnswerInput struct {
	Kind       string   `json:"kind"`
	SelectedID string   `json:"selectedId,omitempty"`
	Sequence   []string `json:"sequence,omitempty"`
}

type HistorySnapshot struct {
	Schema       int            `json:"schema"`
	Kind         string          `json:"kind"`
	SkillCode    string          `json:"skillCode"`
	Example      PoemExample     `json:"example"`
	Input        AnswerInput     `json:"input,omitempty"`
	Selected     string         `json:"selected,omitempty"`
	ResponseKind string          `json:"responseKind"`
	MediaSHA256  map[string]string `json:"mediaSHA256,omitempty"`
}

type PlanExample struct {
	Example      PoemExample
	Input        AnswerInput
	Selected     string
	ResponseKind string
}

func SkillForKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case KindTitle, KindFill, KindCouplet, KindRecite:
		return kind
	case "nextline":
		return KindCouplet
	}
	return ""
}

func KindForSkill(code string) string { return SkillForKind(code) }

func NeedsSpeech(kind string) bool {
	switch KindForSkill(kind) {
	case KindTitle, KindCouplet:
		return true
	}
	return false
}

func Validate(e PoemExample) error {
	kind := KindForSkill(e.Kind)
	if kind == "" {
		return fmt.Errorf("未知古诗题型 %q", e.Kind)
	}
	if strings.TrimSpace(e.Prompt) == "" {
		return fmt.Errorf("%s 缺少题干", kind)
	}
	if strings.TrimSpace(e.WorkID) == "" {
		return fmt.Errorf("%s 缺少作品身份", kind)
	}
	switch kind {
	case KindTitle:
		return validateChoice(e, true)
	case KindFill:
		if strings.TrimSpace(e.LineID) == "" || strings.TrimSpace(e.SourceLine) == "" || len(e.GapIndexes) == 0 {
			return fmt.Errorf("补字题缺少诗行或缺字位置")
		}
		runes := []rune(e.SourceLine)
		for _, idx := range e.GapIndexes {
			if idx < 0 || idx >= len(runes) {
				return fmt.Errorf("缺字位置超出原文")
			}
		}
		return validateChoice(e, false)
	case KindCouplet:
		if strings.TrimSpace(e.UpperLineID) == "" || strings.TrimSpace(e.NextLineID) == "" {
			return fmt.Errorf("选下一句缺少上下句身份")
		}
		return validateChoice(e, false)
	case KindRecite:
		return validateRecite(e)
	}
	return nil
}

func validateChoice(e PoemExample, requireWorkOpts bool) error {
	if len(e.Options) < 2 {
		return fmt.Errorf("选择题至少需要两个选项")
	}
	seen := map[string]bool{}
	found := false
	for _, o := range e.Options {
		id := strings.TrimSpace(o.ID)
		if id == "" || strings.TrimSpace(o.Label) == "" || seen[id] {
			return fmt.Errorf("选项编号无效")
		}
		seen[id] = true
		if id == e.AnswerID {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("缺少正确答案")
	}
	if requireWorkOpts && e.AnswerID != e.WorkID && !strings.HasPrefix(e.AnswerID, e.WorkID) {
		// title answers use workId
		if e.Kind == KindTitle && e.AnswerID != e.WorkID {
			return fmt.Errorf("选诗名答案必须是作品稳定编号")
		}
	}
	return nil
}

func validateRecite(e PoemExample) error {
	if len(e.SequenceItems) < 2 {
		return fmt.Errorf("排顺序至少需要两句")
	}
	seen := map[string]bool{}
	for _, n := range e.SequenceItems {
		if strings.TrimSpace(n.ID) == "" || strings.TrimSpace(n.Label) == "" || seen[n.ID] {
			return fmt.Errorf("诗行编号无效")
		}
		seen[n.ID] = true
	}
	if len(e.CorrectSequence) != len(e.SequenceItems) {
		return fmt.Errorf("正确顺序不完整")
	}
	for _, id := range e.CorrectSequence {
		if !seen[id] {
			return fmt.Errorf("正确答案引用了未知诗行")
		}
	}
	return nil
}

func DisplayFillLine(source string, gaps []int) string {
	runes := []rune(source)
	mark := map[int]bool{}
	for _, idx := range gaps {
		if idx >= 0 && idx < len(runes) {
			mark[idx] = true
		}
	}
	out := make([]rune, 0, len(runes))
	for i, r := range runes {
		if mark[i] {
			out = append(out, '□')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func IsHanContent(r rune) bool {
	if unicode.Is(unicode.Han, r) {
		switch r {
		case '的', '了', '吗', '呢', '啊', '呀':
			return true
		}
		return true
	}
	return false
}

func IsPunct(r rune) bool {
	switch r {
	case '，', '。', '？', '！', '、', '；', '：', ',', '.', '?', '!', ' ':
		return true
	}
	return unicode.IsPunct(r)
}

func inputHasAnswer(in AnswerInput) bool {
	switch KindForSkill(in.Kind) {
	case KindTitle, KindFill, KindCouplet:
		return strings.TrimSpace(in.SelectedID) != ""
	case KindRecite:
		return len(in.Sequence) > 0
	}
	return strings.TrimSpace(in.SelectedID) != "" || len(in.Sequence) > 0
}

func EncodeInput(in AnswerInput) string {
	if !inputHasAnswer(in) {
		return ""
	}
	kind := KindForSkill(in.Kind)
	if (kind == KindTitle || kind == KindFill || kind == KindCouplet) && strings.TrimSpace(in.SelectedID) != "" && len(in.Sequence) == 0 {
		return strings.TrimSpace(in.SelectedID)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return ""
	}
	return string(raw)
}

func DecodeInput(raw, kind string) AnswerInput {
	raw = strings.TrimSpace(raw)
	kind = KindForSkill(kind)
	if raw == "" {
		return AnswerInput{Kind: kind}
	}
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		var in AnswerInput
		if json.Unmarshal([]byte(raw), &in) == nil {
			if in.Kind == "" {
				in.Kind = kind
			}
			if kind == KindRecite && len(in.Sequence) == 0 {
				var seq []string
				if json.Unmarshal([]byte(raw), &seq) == nil {
					in.Sequence = seq
				}
			}
			return in
		}
		var seq []string
		if json.Unmarshal([]byte(raw), &seq) == nil {
			return AnswerInput{Kind: KindRecite, Sequence: seq}
		}
	}
	if kind == KindRecite {
		parts := strings.Split(raw, ",")
		seq := make([]string, 0, len(parts))
		for _, p := range parts {
			if v := strings.TrimSpace(p); v != "" && v != "done" {
				seq = append(seq, v)
			}
		}
		if len(seq) > 0 {
			return AnswerInput{Kind: KindRecite, Sequence: seq}
		}
	}
	return AnswerInput{Kind: kind, SelectedID: raw}
}

func AttemptSelected(attemptSelected, picks string) string {
	if v := meaningfulSelected(attemptSelected); v != "" {
		return v
	}
	v := strings.TrimSpace(picks)
	if v == "" || (!strings.HasPrefix(v, "{") && strings.Contains(v, ",") && !strings.Contains(v, ":")) {
		return ""
	}
	return meaningfulSelected(v)
}

func meaningfulSelected(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !inputHasAnswer(DecodeInput(raw, "")) {
		return ""
	}
	return raw
}

func IsCorrect(e PoemExample, in AnswerInput) bool {
	switch KindForSkill(e.Kind) {
	case KindTitle, KindFill, KindCouplet:
		return strings.TrimSpace(in.SelectedID) != "" && in.SelectedID == e.AnswerID
	case KindRecite:
		if len(in.Sequence) != len(e.CorrectSequence) {
			return false
		}
		for i, id := range e.CorrectSequence {
			if in.Sequence[i] != id {
				return false
			}
		}
		return true
	}
	return false
}

func ErrorFacts(e PoemExample, in AnswerInput) []string {
	var facts []string
	switch KindForSkill(e.Kind) {
	case KindTitle:
		if in.SelectedID != "" && in.SelectedID != e.AnswerID {
			facts = append(facts, fmt.Sprintf("孩子选了「%s」，这首作品是「%s」。", labelOf(e, in.SelectedID), labelOf(e, e.AnswerID)))
		}
	case KindFill:
		if in.SelectedID != "" && in.SelectedID != e.AnswerID {
			facts = append(facts, fmt.Sprintf("孩子选了「%s」，缺的字是「%s」。", labelOf(e, in.SelectedID), labelOf(e, e.AnswerID)))
		}
	case KindCouplet:
		if in.SelectedID != "" && in.SelectedID != e.AnswerID {
			facts = append(facts, fmt.Sprintf("孩子选了「%s」，下一句是「%s」。", labelOf(e, in.SelectedID), labelOf(e, e.AnswerID)))
		}
	case KindRecite:
		if !IsCorrect(e, in) && len(in.Sequence) > 0 {
			facts = append(facts, fmt.Sprintf("孩子排出的顺序是 %s，正确顺序是 %s。", joinLabels(e.SequenceItems, in.Sequence), joinLabels(e.SequenceItems, e.CorrectSequence)))
		}
	}
	return facts
}

func HistoryBytes(snap HistorySnapshot) ([]byte, error) {
	if snap.Schema != 1 {
		snap.Schema = 1
	}
	snap.Kind = KindForSkill(snap.Kind)
	if snap.Kind == "" {
		snap.Kind = KindForSkill(snap.Example.Kind)
	}
	if snap.SkillCode == "" {
		snap.SkillCode = SkillForKind(snap.Kind)
	}
	snap.ResponseKind = snap.Kind
	if snap.Example.Kind == "" {
		snap.Example.Kind = snap.Kind
	}
	if err := Validate(snap.Example); err != nil {
		return nil, err
	}
	if snap.Input.Kind == "" {
		snap.Input.Kind = snap.Kind
	}
	if snap.Selected == "" && inputHasAnswer(snap.Input) {
		snap.Selected = EncodeInput(snap.Input)
	}
	return json.Marshal(snap)
}

func PlanExampleFromSnapshot(raw, selected string) (PlanExample, error) {
	var snap HistorySnapshot
	if json.Unmarshal([]byte(raw), &snap) != nil || snap.Schema != 1 || KindForSkill(snap.Kind) == "" {
		return PlanExample{}, fmt.Errorf("invalid snapshot")
	}
	if snap.Example.Kind == "" {
		snap.Example.Kind = snap.Kind
	}
	if err := Validate(snap.Example); err != nil {
		return PlanExample{}, err
	}
	in := snap.Input
	picked := AttemptSelected(selected, snap.Selected)
	if picked != "" {
		in = DecodeInput(picked, snap.Kind)
	}
	if in.Kind == "" {
		in.Kind = snap.Kind
	}
	out := EncodeInput(in)
	if picked != "" {
		out = picked
	}
	return PlanExample{Example: snap.Example, Input: in, Selected: out, ResponseKind: snap.Kind}, nil
}

func labelOf(e PoemExample, id string) string {
	for _, o := range e.Options {
		if o.ID == id {
			return o.Label
		}
	}
	for _, n := range e.SequenceItems {
		if n.ID == id {
			return n.Label
		}
	}
	return id
}

func joinLabels(items []Node, ids []string) string {
	labels := map[string]string{}
	for _, n := range items {
		labels[n.ID] = n.Label
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if labels[id] != "" {
			parts = append(parts, labels[id])
		} else {
			parts = append(parts, id)
		}
	}
	return strings.Join(parts, "→")
}

func CollectMediaURLs(e PoemExample) []string {
	if u := strings.TrimSpace(e.SpeechURL); u != "" {
		return []string{u}
	}
	return nil
}

func RewriteMediaURLs(e *PoemExample, rewrite func(string) string) {
	e.SpeechURL = rewrite(e.SpeechURL)
}

func LineID(workID string, ord int) string {
	return fmt.Sprintf("%s:L%d", workID, ord)
}

func Texts(w Work) []string {
	out := make([]string, 0, len(w.Lines))
	for _, line := range w.Lines {
		out = append(out, line.Text)
	}
	return out
}
