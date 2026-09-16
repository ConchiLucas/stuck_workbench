package sciencecontent

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	KindChoice   = "choice"
	KindMatch    = "match"
	KindSequence = "sequence"
	KindLabel    = "label"
	DiagramPlant = "plant-structure"
	DiagramVer   = 1
)

type Node struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Icon     string `json:"icon,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type Choice struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Icon     string `json:"icon,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type LabelTarget struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

type ScienceExample struct {
	Kind                 string            `json:"kind"`
	Prompt               string            `json:"prompt"`
	Visual               string            `json:"visual,omitempty"`
	Icon                 string            `json:"icon,omitempty"`
	ImageURL             string            `json:"imageUrl,omitempty"`
	Options              []Choice          `json:"options,omitempty"`
	AnswerID             string            `json:"answerId,omitempty"`
	OptionOrder          []string          `json:"optionOrder,omitempty"`
	MatchSources         []Node            `json:"matchSources,omitempty"`
	MatchTargets         []Node            `json:"matchTargets,omitempty"`
	MatchAnswers         map[string]string `json:"matchAnswers,omitempty"`
	MatchSourceOrder     []string          `json:"matchSourceOrder,omitempty"`
	MatchTargetOrder     []string          `json:"matchTargetOrder,omitempty"`
	MatchSourceGroup     string            `json:"matchSourceGroup,omitempty"`
	MatchTargetGroup     string            `json:"matchTargetGroup,omitempty"`
	SequenceItems        []Node            `json:"sequenceItems,omitempty"`
	SequenceDisplayOrder []string          `json:"sequenceDisplayOrder,omitempty"`
	CorrectSequence      []string          `json:"correctSequence,omitempty"`
	SequenceLoop         bool              `json:"sequenceLoop,omitempty"`
	SequenceStartID      string            `json:"sequenceStartId,omitempty"`
	DiagramKey           string            `json:"diagramKey,omitempty"`
	DiagramVersion       int               `json:"diagramVersion,omitempty"`
	LabelTargets         []LabelTarget     `json:"labelTargets,omitempty"`
	LabelTokens          []Node            `json:"labelTokens,omitempty"`
	LabelAnswers         map[string]string `json:"labelAnswers,omitempty"`
	Explanation          string            `json:"explanation,omitempty"`
	Tip                  string            `json:"tip,omitempty"`
	Rationale            string            `json:"rationale,omitempty"`
}

type AnswerInput struct {
	Kind       string            `json:"kind"`
	SelectedID string            `json:"selectedId,omitempty"`
	Pairs      map[string]string `json:"pairs,omitempty"`
	Sequence   []string          `json:"sequence,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type HistorySnapshot struct {
	Schema       int               `json:"schema"`
	Kind         string            `json:"kind"`
	SkillCode    string            `json:"skillCode"`
	Example      ScienceExample    `json:"example"`
	Input        AnswerInput       `json:"input"`
	Selected     string            `json:"selected,omitempty"`
	ResponseKind string            `json:"responseKind"`
	MediaSHA256  map[string]string `json:"mediaSHA256,omitempty"`
}

type PlanExample struct {
	Example      ScienceExample
	Input        AnswerInput
	Selected     string
	ResponseKind string
}

var kinds = map[string]string{
	KindChoice:   KindChoice,
	KindMatch:    KindMatch,
	KindSequence: KindSequence,
	KindLabel:    KindLabel,
	"recognize":  KindChoice,
}

func SkillForKind(kind string) string {
	if kind == "recognize" {
		return KindChoice
	}
	return kinds[kind]
}

func KindForSkill(code string) string {
	if code == "recognize" {
		return KindChoice
	}
	return kinds[code]
}

func Validate(e ScienceExample) error {
	kind := KindForSkill(strings.TrimSpace(e.Kind))
	if kind == "" {
		return fmt.Errorf("未知科普题型 %q", e.Kind)
	}
	if strings.TrimSpace(e.Prompt) == "" {
		return fmt.Errorf("%s 缺少题干", kind)
	}
	switch kind {
	case KindChoice:
		return validateChoice(e)
	case KindMatch:
		return validateMatch(e)
	case KindSequence:
		return validateSequence(e)
	case KindLabel:
		return validateLabel(e)
	}
	return nil
}

func validateChoice(e ScienceExample) error {
	if len(e.Options) < 2 {
		return fmt.Errorf("选择题至少需要两个选项")
	}
	seen := map[string]bool{}
	found := false
	for _, o := range e.Options {
		id := strings.TrimSpace(o.ID)
		if id == "" || strings.TrimSpace(o.Label) == "" || seen[id] {
			return fmt.Errorf("选择题选项编号无效")
		}
		seen[id] = true
		if id == e.AnswerID {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("选择题缺少正确答案")
	}
	return nil
}

func validateMatch(e ScienceExample) error {
	if len(e.MatchSources) < 2 || len(e.MatchTargets) < 2 {
		return fmt.Errorf("连线题至少需要两组节点")
	}
	src, tgt := map[string]bool{}, map[string]bool{}
	for _, n := range e.MatchSources {
		if strings.TrimSpace(n.ID) == "" || strings.TrimSpace(n.Label) == "" || src[n.ID] {
			return fmt.Errorf("连线题来源编号无效")
		}
		src[n.ID] = true
	}
	for _, n := range e.MatchTargets {
		if strings.TrimSpace(n.ID) == "" || strings.TrimSpace(n.Label) == "" || tgt[n.ID] {
			return fmt.Errorf("连线题目标编号无效")
		}
		tgt[n.ID] = true
	}
	if len(e.MatchAnswers) != len(e.MatchSources) {
		return fmt.Errorf("连线题必须给出全部来源的正确配对")
	}
	used := map[string]bool{}
	for from, to := range e.MatchAnswers {
		if !src[from] || !tgt[to] || used[to] {
			return fmt.Errorf("连线题配对不是稳定的一一对应")
		}
		used[to] = true
	}
	return nil
}

func validateSequence(e ScienceExample) error {
	if len(e.SequenceItems) < 2 {
		return fmt.Errorf("排序题至少需要两个步骤")
	}
	seen := map[string]bool{}
	for _, n := range e.SequenceItems {
		if strings.TrimSpace(n.ID) == "" || strings.TrimSpace(n.Label) == "" || seen[n.ID] {
			return fmt.Errorf("排序题元素编号无效")
		}
		seen[n.ID] = true
	}
	if len(e.CorrectSequence) != len(e.SequenceItems) {
		return fmt.Errorf("排序题正确顺序不完整")
	}
	for _, id := range e.CorrectSequence {
		if !seen[id] {
			return fmt.Errorf("排序题正确答案引用了未知元素")
		}
	}
	if e.SequenceLoop && strings.TrimSpace(e.SequenceStartID) == "" {
		return fmt.Errorf("循环过程必须指定起点")
	}
	return nil
}

func validateLabel(e ScienceExample) error {
	if strings.TrimSpace(e.DiagramKey) == "" || e.DiagramVersion < 1 {
		return fmt.Errorf("结构标注题缺少结构图定义")
	}
	if len(e.LabelTargets) < 1 || len(e.LabelTokens) < 1 {
		return fmt.Errorf("结构标注题缺少目标和标签")
	}
	targets, tokens := map[string]bool{}, map[string]bool{}
	for _, t := range e.LabelTargets {
		if strings.TrimSpace(t.ID) == "" || strings.TrimSpace(t.Label) == "" || targets[t.ID] {
			return fmt.Errorf("标注目标编号无效")
		}
		if t.X < 0 || t.X > 1 || t.Y < 0 || t.Y > 1 {
			return fmt.Errorf("标注坐标必须是相对图幅的 0–1")
		}
		targets[t.ID] = true
	}
	for _, n := range e.LabelTokens {
		if strings.TrimSpace(n.ID) == "" || strings.TrimSpace(n.Label) == "" || tokens[n.ID] {
			return fmt.Errorf("标注标签编号无效")
		}
		tokens[n.ID] = true
	}
	if len(e.LabelAnswers) == 0 {
		return fmt.Errorf("结构标注题缺少正确映射")
	}
	for target, token := range e.LabelAnswers {
		if !targets[target] || !tokens[token] {
			return fmt.Errorf("结构标注题映射无效")
		}
	}
	return nil
}

func inputHasAnswer(in AnswerInput) bool {
	switch KindForSkill(in.Kind) {
	case KindChoice:
		return strings.TrimSpace(in.SelectedID) != ""
	case KindMatch:
		return len(in.Pairs) > 0
	case KindSequence:
		return len(in.Sequence) > 0
	case KindLabel:
		return len(in.Labels) > 0
	}
	return strings.TrimSpace(in.SelectedID) != "" || len(in.Pairs) > 0 || len(in.Sequence) > 0 || len(in.Labels) > 0
}

func EncodeInput(in AnswerInput) string {
	if !inputHasAnswer(in) {
		return ""
	}
	kind := KindForSkill(in.Kind)
	if kind == KindChoice && strings.TrimSpace(in.SelectedID) != "" && in.Pairs == nil && len(in.Sequence) == 0 && len(in.Labels) == 0 {
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
			return in
		}
	}
	if kind == KindChoice || kind == "" {
		return AnswerInput{Kind: KindChoice, SelectedID: raw}
	}
	return AnswerInput{Kind: kind}
}

func AttemptSelected(attemptSelected, picks string) string {
	if v := meaningfulSelected(attemptSelected); v != "" {
		return v
	}
	v := strings.TrimSpace(picks)
	if v == "" || (!strings.HasPrefix(v, "{") && strings.Contains(v, ",")) {
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

func IsCorrect(e ScienceExample, in AnswerInput) bool {
	switch KindForSkill(e.Kind) {
	case KindChoice:
		return strings.TrimSpace(in.SelectedID) != "" && in.SelectedID == e.AnswerID
	case KindMatch:
		if len(in.Pairs) != len(e.MatchAnswers) {
			return false
		}
		for from, to := range e.MatchAnswers {
			if in.Pairs[from] != to {
				return false
			}
		}
		return true
	case KindSequence:
		if len(in.Sequence) != len(e.CorrectSequence) {
			return false
		}
		for i, id := range e.CorrectSequence {
			if in.Sequence[i] != id {
				return false
			}
		}
		return true
	case KindLabel:
		if len(in.Labels) != len(e.LabelAnswers) {
			return false
		}
		for target, token := range e.LabelAnswers {
			if in.Labels[target] != token {
				return false
			}
		}
		for target := range in.Labels {
			if e.LabelAnswers[target] == "" {
				return false
			}
		}
		return true
	}
	return false
}

func ErrorFacts(e ScienceExample, in AnswerInput) []string {
	var facts []string
	switch KindForSkill(e.Kind) {
	case KindChoice:
		got := labelOf(e, in.SelectedID)
		want := labelOf(e, e.AnswerID)
		if in.SelectedID != "" && in.SelectedID != e.AnswerID {
			facts = append(facts, fmt.Sprintf("孩子选了「%s」，正确答案是「%s」。", got, want))
		}
	case KindMatch:
		src, tgt := nodeLabels(e.MatchSources), nodeLabels(e.MatchTargets)
		for from, want := range e.MatchAnswers {
			got := in.Pairs[from]
			if got == "" {
				facts = append(facts, fmt.Sprintf("「%s」没有连线。", src[from]))
				continue
			}
			if got != want {
				facts = append(facts, fmt.Sprintf("「%s」连到了「%s」，应连到「%s」。", src[from], tgt[got], tgt[want]))
			}
		}
	case KindSequence:
		if !IsCorrect(e, in) && len(in.Sequence) > 0 {
			facts = append(facts, fmt.Sprintf("孩子排出的顺序是 %s，正确顺序是 %s。", joinLabels(e.SequenceItems, in.Sequence), joinLabels(e.SequenceItems, e.CorrectSequence)))
		}
	case KindLabel:
		token := tokenLabels(e)
		target := targetLabels(e)
		for id, want := range e.LabelAnswers {
			got := in.Labels[id]
			if got == "" {
				facts = append(facts, fmt.Sprintf("「%s」位置没有放标签。", target[id]))
				continue
			}
			if got != want {
				facts = append(facts, fmt.Sprintf("「%s」位置放了「%s」，应放「%s」。", target[id], token[got], token[want]))
			}
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
	if snap.Selected == "" {
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

func labelOf(e ScienceExample, id string) string {
	for _, o := range e.Options {
		if o.ID == id {
			return o.Label
		}
	}
	return id
}

func nodeLabels(nodes []Node) map[string]string {
	out := map[string]string{}
	for _, n := range nodes {
		out[n.ID] = n.Label
	}
	return out
}

func tokenLabels(e ScienceExample) map[string]string {
	return nodeLabels(e.LabelTokens)
}

func targetLabels(e ScienceExample) map[string]string {
	out := map[string]string{}
	for _, t := range e.LabelTargets {
		out[t.ID] = t.Label
	}
	return out
}

func joinLabels(items []Node, ids []string) string {
	labels := nodeLabels(items)
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

func CollectImageURLs(e ScienceExample) []string {
	var urls []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u != "" {
			urls = append(urls, u)
		}
	}
	add(e.ImageURL)
	for _, o := range e.Options {
		add(o.ImageURL)
	}
	for _, n := range e.MatchSources {
		add(n.ImageURL)
	}
	for _, n := range e.MatchTargets {
		add(n.ImageURL)
	}
	for _, n := range e.SequenceItems {
		add(n.ImageURL)
	}
	return urls
}

func RewriteImageURLs(e *ScienceExample, rewrite func(string) string) {
	e.ImageURL = rewrite(e.ImageURL)
	for i := range e.Options {
		e.Options[i].ImageURL = rewrite(e.Options[i].ImageURL)
	}
	for i := range e.MatchSources {
		e.MatchSources[i].ImageURL = rewrite(e.MatchSources[i].ImageURL)
	}
	for i := range e.MatchTargets {
		e.MatchTargets[i].ImageURL = rewrite(e.MatchTargets[i].ImageURL)
	}
	for i := range e.SequenceItems {
		e.SequenceItems[i].ImageURL = rewrite(e.SequenceItems[i].ImageURL)
	}
}
