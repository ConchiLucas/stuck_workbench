package logiccontent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	SubjectCode  = "logic"
	GlyphVersion = "2"
	SnapshotSchema = 1
	TableName    = "logic_question_task_media"
	ItemMediaTable = "logic_item_media"
	LiveGlyphPath = "/api/v1/logic/items/%d/glyph/%s.svg"
	FrozenPath   = "/api/v1/logic/task-media/%s.svg"
)

var SkillCodes = []string{"pattern", "classify", "order", "shape_reason", "diff", "compare"}

var KindNames = map[string]string{
	"pattern":      "找规律",
	"classify":     "分类",
	"order":        "排序",
	"shape_reason": "图形推理",
	"diff":         "找不同",
	"compare":      "比较",
}

type Object struct {
	ID     string            `json:"id"`
	Caption string           `json:"caption"`
	Glyph  string            `json:"glyph"`
	Fill   string            `json:"fill,omitempty"`
	Scale  int               `json:"scale,omitempty"`
	Rotate int               `json:"rotate,omitempty"`
	Count  int               `json:"count,omitempty"`
	Attrs  map[string]string `json:"attrs,omitempty"`
}

type Rule struct {
	Type       string `json:"type"`
	Dimension  string `json:"dimension,omitempty"`
	Direction  string `json:"direction,omitempty"`
	Period     int    `json:"period,omitempty"`
	InGroup    string `json:"inGroup,omitempty"`
	Explain    string `json:"explain"`
}

type LogicExample struct {
	Kind             string   `json:"kind"`
	Prompt           string   `json:"prompt"`
	Rule             Rule      `json:"rule"`
	Objects          []Object `json:"objects"`
	Sequence         []string `json:"sequence,omitempty"`
	Options          []string `json:"options,omitempty"`
	AnswerID         string   `json:"answerId,omitempty"`
	CorrectSequence  []string `json:"correctSequence,omitempty"`
	DisplayOrder     []string `json:"displayOrder,omitempty"`
	KpID             int64   `json:"kpId,omitempty"`
	GlyphVersion     string   `json:"glyphVersion,omitempty"`
	ImageURLs        map[string]string `json:"imageUrls,omitempty"`
}

type AnswerInput struct {
	SelectedID string          `json:"selectedId,omitempty"`
	Sequence    []string       `json:"sequence,omitempty"`
	Rejected   []RejectedTap   `json:"rejected,omitempty"`
	ClientID   string          `json:"clientId,omitempty"`
	CostMs     int            `json:"costMs,omitempty"`
}

type RejectedTap struct {
	ID      string `json:"id"`
	AtIndex int    `json:"atIndex"`
}

type HistorySnapshot struct {
	Schema           int               `json:"schema"`
	Kind             string            `json:"kind"`
	Prompt           string            `json:"prompt"`
	Rule             Rule              `json:"rule"`
	Objects          []Object         `json:"objects"`
	Sequence         []string         `json:"sequence,omitempty"`
	Options          []string         `json:"options,omitempty"`
	AnswerID         string            `json:"answerId,omitempty"`
	CorrectSequence  []string         `json:"correctSequence,omitempty"`
	DisplayOrder     []string         `json:"displayOrder,omitempty"`
	GlyphVersion     string            `json:"glyphVersion,omitempty"`
	ImageURLs        map[string]string `json:"imageUrls,omitempty"`
	MediaSHA256      map[string]string `json:"mediaSha256,omitempty"`
	MediaImmutable   bool              `json:"mediaImmutable"`
}

func ObjectByID(objects []Object, id string) (Object, bool) {
	for _, o := range objects {
		if o.ID == id {
			return o, true
		}
	}
	return Object{}, false
}

func (e LogicExample) Object(id string) (Object, bool) {
	return ObjectByID(e.Objects, id)
}

func Judge(example LogicExample, in AnswerInput) (bool, error) {
	if err := Validate(example); err != nil {
		return false, err
	}
	if example.Kind == "order" {
		if len(in.Sequence) != len(example.CorrectSequence) {
			return false, nil
		}
		for i, id := range example.CorrectSequence {
			if in.Sequence[i] != id {
				return false, nil
			}
		}
		return true, nil
	}
	if in.SelectedID == "" {
		return false, nil
	}
	return in.SelectedID == example.AnswerID, nil
}

func IsCorrect(example LogicExample, in AnswerInput) bool {
	ok, err := Judge(example, in)
	return err == nil && ok
}

func Validate(e LogicExample) error {
	if KindNames[e.Kind] == "" {
		return fmt.Errorf("未知逻辑题型 %s", e.Kind)
	}
	if strings.TrimSpace(e.Prompt) == "" {
		return fmt.Errorf("题干不能为空")
	}
	if strings.TrimSpace(e.Rule.Explain) == "" {
		return fmt.Errorf("规则说明不能为空")
	}
	if len(e.Objects) < 2 {
		return fmt.Errorf("至少需要两个对象")
	}
	seen := map[string]bool{}
	for _, o := range e.Objects {
		if o.ID == "" || o.Caption == "" || o.Glyph == "" {
			return fmt.Errorf("对象必须有稳定 id、caption、glyph")
		}
		if seen[o.ID] {
			return fmt.Errorf("对象 id 重复 %s", o.ID)
		}
		seen[o.ID] = true
		if o.Scale < 0 || o.Scale > 4 {
			return fmt.Errorf("对象 %s scale 必须是 0–4", o.ID)
		}
	}
	switch e.Kind {
	case "order":
		return validateOrder(e)
	default:
		return validateChoice(e)
	}
}

func validateChoice(e LogicExample) error {
	if len(e.Options) < 2 {
		return fmt.Errorf("%s 至少两个选项", e.Kind)
	}
	if e.AnswerID == "" {
		return fmt.Errorf("缺少 answerId")
	}
	if _, ok := ObjectByID(e.Objects, e.AnswerID); !ok {
		return fmt.Errorf("答案 %s 不在对象中", e.AnswerID)
	}
	optSeen := map[string]bool{}
	hasAnswer := false
	for _, id := range e.Options {
		if optSeen[id] {
			return fmt.Errorf("选项重复 %s", id)
		}
		optSeen[id] = true
		if _, ok := ObjectByID(e.Objects, id); !ok {
			return fmt.Errorf("选项 %s 不在对象中", id)
		}
		if id == e.AnswerID {
			hasAnswer = true
		}
	}
	if !hasAnswer {
		return fmt.Errorf("选项未包含答案")
	}
	switch e.Kind {
	case "pattern", "shape_reason":
		if len(e.Sequence) < 3 {
			return fmt.Errorf("%s 序列至少 3 项", e.Kind)
		}
		for _, id := range e.Sequence {
			if _, ok := ObjectByID(e.Objects, id); !ok {
				return fmt.Errorf("序列元素 %s 不在对象中", id)
			}
		}
		if e.Rule.Type == "" {
			return fmt.Errorf("规律类型不能为空")
		}
	case "classify":
		if e.Rule.Dimension == "" || e.Rule.InGroup == "" {
			return fmt.Errorf("分类必须标明维度和所属类")
		}
		odd := oddOneOut(e)
		if len(odd) != 1 || odd[0] != e.AnswerID {
			return fmt.Errorf("分类答案必须是唯一不属于该类的对象")
		}
	case "diff":
		odd := uniqueDiff(e)
		if len(odd) != 1 || odd[0] != e.AnswerID {
			return fmt.Errorf("找不同必须有且仅有一个异常项")
		}
	case "compare":
		if e.Rule.Dimension == "" || e.Rule.Direction == "" {
			return fmt.Errorf("比较必须标明属性和方向")
		}
		winner := uniqueCompare(e)
		if len(winner) != 1 || winner[0] != e.AnswerID {
			return fmt.Errorf("比较必须有且仅有一个符合条件的对象")
		}
		if e.Rule.Dimension == "visualSize" {
			for _, id := range e.Options {
				o, _ := ObjectByID(e.Objects, id)
				if o.Scale < 1 {
					return fmt.Errorf("画面尺寸比较必须给出 scale")
				}
			}
		}
	}
	return nil
}

func validateOrder(e LogicExample) error {
	if len(e.CorrectSequence) < 3 {
		return fmt.Errorf("排序至少 3 项")
	}
	if e.Rule.Dimension == "" || e.Rule.Direction == "" {
		return fmt.Errorf("排序必须标明维度和方向")
	}
	seen := map[string]bool{}
	for _, id := range e.CorrectSequence {
		if seen[id] {
			return fmt.Errorf("正确序列 id 重复 %s", id)
		}
		seen[id] = true
		if _, ok := ObjectByID(e.Objects, id); !ok {
			return fmt.Errorf("序列元素 %s 不在对象中", id)
		}
	}
	if len(e.DisplayOrder) != len(e.CorrectSequence) {
		return fmt.Errorf("初始顺序必须覆盖全部元素")
	}
	disp := map[string]bool{}
	for _, id := range e.DisplayOrder {
		if disp[id] {
			return fmt.Errorf("初始顺序 id 重复")
		}
		disp[id] = true
		if !seen[id] {
			return fmt.Errorf("初始顺序含未知 id %s", id)
		}
	}
	return nil
}

func oddOneOut(e LogicExample) []string {
	var odd []string
	for _, id := range e.Options {
		o, _ := ObjectByID(e.Objects, id)
		cat := o.Attrs["category"]
		if cat != e.Rule.InGroup {
			odd = append(odd, id)
		}
	}
	return odd
}

func uniqueDiff(e LogicExample) []string {
	dim := e.Rule.Dimension
	if dim == "" {
		return nil
	}
	counts := map[string][]string{}
	for _, id := range e.Options {
		o, _ := ObjectByID(e.Objects, id)
		key := attrOr(o, dim)
		counts[key] = append(counts[key], id)
	}
	var unique []string
	for _, ids := range counts {
		if len(ids) == 1 {
			unique = append(unique, ids[0])
		}
	}
	sort.Strings(unique)
	return unique
}

func uniqueCompare(e LogicExample) []string {
	dim := e.Rule.Dimension
	dir := e.Rule.Direction
	bestIDs := []string{}
	best := -1
	first := true
	for _, id := range e.Options {
		o, _ := ObjectByID(e.Objects, id)
		v := compareValue(o, dim)
		if first || (dir == "max" && v > best) || (dir == "min" && v < best) {
			best = v
			bestIDs = []string{id}
			first = false
			continue
		}
		if v == best {
			bestIDs = append(bestIDs, id)
		}
	}
	return bestIDs
}

func attrOr(o Object, dim string) string {
	if o.Attrs != nil {
		if v := o.Attrs[dim]; v != "" {
			return v
		}
	}
	switch dim {
	case "fill", "color":
		return o.Fill
	case "glyph", "shape":
		return o.Glyph
	case "rotate", "direction":
		return fmt.Sprintf("%d", o.Rotate)
	}
	return ""
}

func compareValue(o Object, dim string) int {
	if o.Attrs != nil {
		if v := o.Attrs[dim]; v != "" {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				return n
			}
		}
	}
	if dim == "visualSize" || dim == "size" {
		return o.Scale
	}
	return o.Scale
}

func ShuffleIDs(ids []string, seed int64) []string {
	out := append([]string(nil), ids...)
	// deterministic LCG
	x := uint64(seed)
	if x == 0 {
		x = 1
	}
	for i := len(out) - 1; i > 0; i-- {
		x = x*1664525 + 1013904223
		j := int(x % uint64(i+1))
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func EncodeJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func DecodeExample(raw string) (LogicExample, error) {
	var e LogicExample
	if strings.TrimSpace(raw) == "" {
		return e, fmt.Errorf("空快照")
	}
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return e, err
	}
	return e, Validate(e)
}

func ExampleToSnapshot(e LogicExample) HistorySnapshot {
	return HistorySnapshot{
		Schema:          SnapshotSchema,
		Kind:            e.Kind,
		Prompt:          e.Prompt,
		Rule:            e.Rule,
		Objects:         e.Objects,
		Sequence:        e.Sequence,
		Options:         e.Options,
		AnswerID:        e.AnswerID,
		CorrectSequence: e.CorrectSequence,
		DisplayOrder:    e.DisplayOrder,
		GlyphVersion:    firstNonEmpty(e.GlyphVersion, GlyphVersion),
		ImageURLs:       e.ImageURLs,
	}
}

func SnapshotToExample(s HistorySnapshot) LogicExample {
	return LogicExample{
		Kind:            s.Kind,
		Prompt:          s.Prompt,
		Rule:            s.Rule,
		Objects:         s.Objects,
		Sequence:        s.Sequence,
		Options:         s.Options,
		AnswerID:        s.AnswerID,
		CorrectSequence: s.CorrectSequence,
		DisplayOrder:    s.DisplayOrder,
		GlyphVersion:    s.GlyphVersion,
		ImageURLs:       s.ImageURLs,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
