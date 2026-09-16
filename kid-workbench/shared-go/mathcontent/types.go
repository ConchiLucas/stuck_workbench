// Package mathcontent is the versioned content contract shared by material
// publishing, arithmetic task generation and the child detail reader.
package mathcontent

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Bucket struct {
	Label string   `json:"label"`
	Items []string `json:"items"`
}
type MathExample struct {
	Kind           string            `json:"kind"`
	Prompt         string            `json:"prompt"`
	Options        []string          `json:"options,omitempty"`
	Answer         string            `json:"answer,omitempty"`
	Groups         []string          `json:"groups,omitempty"`
	Statements     []string          `json:"statements,omitempty"`
	Buckets        []Bucket          `json:"buckets,omitempty"`
	Operation      string            `json:"operation,omitempty"`
	Counts         []int             `json:"counts,omitempty"`
	Object         string            `json:"object,omitempty"`
	ShapeKeys      []string          `json:"shapeKeys,omitempty"`
	AudioURL       string            `json:"audioUrl,omitempty"`
	ObjectImageURL string            `json:"objectImageUrl,omitempty"`
	ShapeImageURLs map[string]string `json:"shapeImageUrls,omitempty"`
	OptionIDs      []string          `json:"optionIds,omitempty"`
	AnswerOptionID string            `json:"answerOptionId,omitempty"`
}
type MathDetail struct {
	ID                string      `json:"id"`
	GroupID           string      `json:"groupId"`
	Title             string      `json:"title"`
	ModuleTitle       string      `json:"moduleTitle"`
	LearningGoal      string      `json:"learningGoal"`
	Rules             []string    `json:"rules"`
	Revision          int         `json:"revision"`
	PublishedRevision int         `json:"publishedRevision,omitempty"`
	Example           MathExample `json:"example"`
}
type Catalog struct {
	SchemaVersion int          `json:"schemaVersion"`
	Items         []MathDetail `json:"items"`
	Stale         bool         `json:"stale,omitempty"`
}

//go:embed defaults.json
var defaults []byte

func Defaults() Catalog {
	var c Catalog
	if err := json.Unmarshal(defaults, &c); err != nil {
		panic(err)
	}
	return c
}

func Validate(d MathDetail) error {
	fail := func(s string) error { return fmt.Errorf("%s：%s", d.ID, s) }
	if d.ID == "" || len(d.ID) > 80 || d.Title == "" || d.ModuleTitle == "" || d.Revision < 1 {
		return fail("详情身份、标题或版本缺失")
	}
	if d.GroupID != "addition" && d.GroupID != "subtraction" && d.GroupID != "shape" {
		return fail("无效内容分组")
	}
	if strings.TrimSpace(d.LearningGoal) == "" || len(d.Rules) == 0 {
		return fail("请补充学习目标和说明")
	}
	e := d.Example
	if e.AudioURL != "" && !immutableAudio.MatchString(e.AudioURL) {
		return fail("音频必须引用素材后台生成的不可变版本")
	}
	if e.ObjectImageURL != "" && !immutableImage.MatchString(e.ObjectImageURL) {
		return fail("数量图必须引用不可变媒体版本")
	}
	for _, href := range e.ShapeImageURLs {
		if href != "" && !immutableImage.MatchString(href) {
			return fail("图形图必须引用不可变媒体版本")
		}
	}
	if strings.TrimSpace(e.Prompt) == "" {
		return fail("题干不能为空")
	}
	switch e.Kind {
	case "choice", "missing", "objects", "judgement", "audio-shape", "shape-name", "shape-feature":
		if len(e.Options) < 2 || len(e.Options) > 8 {
			return fail("需要2至8个候选项")
		}
		seen := map[string]bool{}
		found := false
		for _, o := range e.Options {
			if strings.TrimSpace(o) == "" || seen[o] {
				return fail("选项不能为空或重复")
			}
			seen[o] = true
			found = found || o == e.Answer
		}
		if !found {
			return fail("正确答案必须对应候选项")
		}
	case "shape-sort":
		if len(e.Buckets) < 2 {
			return fail("分类至少需要两组")
		}
		labels, items := map[string]bool{}, map[string]bool{}
		for _, b := range e.Buckets {
			if b.Label == "" || labels[b.Label] || len(b.Items) == 0 {
				return fail("分组名称须唯一且包含图形")
			}
			labels[b.Label] = true
			for _, x := range b.Items {
				if x == "" || items[x] {
					return fail("图形须唯一对应一个分组")
				}
				items[x] = true
			}
		}
	default:
		return fail("不支持的题面形式")
	}
	if e.Kind == "objects" {
		if (d.GroupID == "addition" && e.Operation != "add") || (d.GroupID == "subtraction" && e.Operation != "sub") {
			return fail("运算与课程分组不一致")
		}
		if len(e.Counts) != 2 || (e.Operation != "add" && e.Operation != "sub") || e.Object == "" {
			return fail("数量图需要物品、运算和两组数量")
		}
		for _, n := range e.Counts {
			if n < 0 || n > 20 {
				return fail("数量须在0至20之间")
			}
		}
		result := e.Counts[0] + e.Counts[1]
		if e.Operation == "sub" {
			result = e.Counts[0] - e.Counts[1]
		}
		if result < 0 || result > 20 || fmt.Sprint(result) != e.Answer {
			return fail("数量关系与答案不一致")
		}
	}
	if d.GroupID != "shape" && (e.Kind == "choice" || e.Kind == "missing") {
		if !courseOperation(d.GroupID, e.Prompt) {
			return fail("算式运算与课程分组不一致")
		}
		if !validEquationAnswer(e.Prompt, e.Answer) {
			return fail("算式与正确答案不一致")
		}
	}
	if e.Kind == "judgement" {
		if len(e.Statements) == 0 {
			return fail("判断题需要算式")
		}
		if len(e.Statements) == 1 {
			if !courseOperation(d.GroupID, e.Statements[0]) {
				return fail("判断算式与课程分组不一致")
			}
			valid, correct := equationTruth(e.Statements[0])
			if !valid {
				return fail("判断算式无效")
			}
			want := "错"
			if correct {
				want = "对"
			}
			if e.Answer != want {
				return fail("判断答案与算式不一致")
			}
		} else {
			wrong := ""
			n := 0
			for _, v := range e.Statements {
				if !courseOperation(d.GroupID, v) {
					return fail("判断算式与课程分组不一致")
				}
				valid, correct := equationTruth(v)
				if !valid {
					return fail("算式无效")
				}
				if !correct {
					wrong = v
					n++
				}
			}
			if n != 1 || wrong != e.Answer {
				return fail("找错题必须有唯一错误算式")
			}
		}
	}
	if e.Kind == "shape-name" {
		if len(e.ShapeKeys) != 1 || shapeNames[e.ShapeKeys[0]] == "" || shapeNames[e.ShapeKeys[0]] != e.Answer {
			return fail("目标图形与名称答案不一致")
		}
		if key := shapeAliases[e.Prompt]; key != "" && key != e.ShapeKeys[0] {
			return fail("图形题干与展示不一致")
		}
	}
	if e.Kind == "audio-shape" || e.Kind == "shape-feature" {
		if len(e.ShapeKeys) != len(e.Options) {
			return fail("图形选项与展示数量不一致")
		}
		seen := map[string]bool{}
		for i, key := range e.ShapeKeys {
			if shapeNames[key] == "" || seen[key] || shapeAliases[e.Options[i]] != key {
				return fail("图形选项无效或重复")
			}
			seen[key] = true
		}
	}
	return nil
}

var shapeNames = map[string]string{"circle": "圆形", "triangle": "三角形", "square": "正方形", "rect": "长方形", "oval": "椭圆形", "trapezoid": "梯形", "rhombus": "菱形", "star": "五角星"}
var shapeAliases = map[string]string{"○": "circle", "◯": "oval", "△": "triangle", "□": "square", "◇": "rhombus", "▭": "rect", "圆形": "circle", "三角形": "triangle", "正方形": "square", "长方形": "rect", "椭圆形": "oval", "梯形": "trapezoid", "菱形": "rhombus", "五角星": "star", "circle": "circle", "triangle": "triangle", "square": "square", "rect": "rect", "oval": "oval", "trapezoid": "trapezoid", "rhombus": "rhombus", "star": "star"}

func courseOperation(group, text string) bool {
	p := equationPattern.FindStringSubmatch(text)
	if p == nil {
		p = choiceBarePattern.FindStringSubmatch(text)
	}
	return p != nil && ((group == "addition" && p[2] == "+") || (group == "subtraction" && p[2] != "+"))
}

var equationPattern = regexp.MustCompile(`^\s*(\d+|[?□])\s*([+−-])\s*(\d+|[?□])\s*=\s*(\d+|[?□])\s*$`)
var choiceBarePattern = regexp.MustCompile(`^\s*(\d+)\s*([+−-])\s*(\d+)(?:\s*=\s*[?？])?\s*$`)
var immutableAudio = regexp.MustCompile(`^/api/v1/math/(detail-audio|task-media)/[a-f0-9]{64}\.mp3$`)
var immutableImage = regexp.MustCompile(`^/api/v1/math/(detail-image|task-media)/[a-f0-9]{64}\.(png|jpe?g|webp)$`)

func validEquationAnswer(prompt, answer string) bool {
	if parts := choiceBarePattern.FindStringSubmatch(prompt); parts != nil && !strings.ContainsAny(prompt, "?？□") {
		answerN, e := strconv.Atoi(answer)
		if e != nil || answerN < 0 || answerN > 20 {
			return false
		}
		valid, correct := equationTruth(parts[1] + parts[2] + parts[3] + "=" + answer)
		return valid && correct
	}
	parts := equationPattern.FindStringSubmatch(prompt)
	if parts == nil {
		return false
	}
	answerN, e := strconv.Atoi(answer)
	if e != nil || answerN < 0 || answerN > 20 {
		return false
	}
	blank := 0
	for _, i := range []int{1, 3, 4} {
		if parts[i] == "?" || parts[i] == "□" {
			parts[i] = answer
			blank++
		}
	}
	if blank != 1 {
		return false
	}
	valid, correct := equationTruth(parts[1] + parts[2] + parts[3] + "=" + parts[4])
	return valid && correct
}
func equationTruth(text string) (bool, bool) {
	parts := equationPattern.FindStringSubmatch(text)
	if parts == nil {
		return false, false
	}
	nums := []int{}
	for _, i := range []int{1, 3, 4} {
		n, e := strconv.Atoi(parts[i])
		if e != nil || n < 0 || n > 20 {
			return false, false
		}
		nums = append(nums, n)
	}
	result := nums[0] + nums[1]
	if parts[2] != "+" {
		result = nums[0] - nums[1]
	}
	if result < 0 || result > 20 {
		return false, false
	}
	return true, result == nums[2]
}
