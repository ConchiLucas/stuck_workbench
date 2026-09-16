package mathtask

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/conchi/study-learning/mathcontent"
)

func generate(source mathcontent.MathDetail, max, sequence int) (mathcontent.MathDetail, error) {
	// Clone slices as well as scalars so source revisions remain immutable.
	raw, _ := json.Marshal(source)
	var d mathcontent.MathDetail
	_ = json.Unmarshal(raw, &d)
	e := &d.Example
	if d.GroupID == "shape" {
		// Keep source-defined shape semantics, only permute presentation.
		if len(e.Options) > 1 {
			perm := rand.Perm(len(e.Options))
			opts := append([]string{}, e.Options...)
			keys := append([]string{}, e.ShapeKeys...)
			for i, j := range perm {
				e.Options[i] = opts[j]
				if len(keys) == len(opts) {
					e.ShapeKeys[i] = keys[j]
				}
			}
		}
		stampOptionIDs(e)
		return d, validateGenerated(d, max)
	}
	a, b := rand.IntN(max+1), 0
	symbol, operation := "+", "add"
	if d.GroupID == "addition" {
		b = rand.IntN(max - a + 1)
	} else if d.GroupID == "subtraction" {
		b = rand.IntN(a + 1)
		symbol = "−"
		operation = "sub"
	} else {
		return d, fmt.Errorf("不支持的内容分组")
	}
	result := a + b
	if operation == "sub" {
		result = a - b
	}
	e.Counts = []int{a, b}
	e.Operation = operation
	e.Groups = nil
	e.AudioURL = ""
	answer := result
	switch e.Kind {
	case "choice":
		e.Prompt = fmt.Sprintf("%d %s %d", a, symbol, b)
	case "missing":
		e.Prompt = fmt.Sprintf("%d %s □ = %d", a, symbol, result)
		answer = b
	case "objects":
		if e.Object == "" {
			return d, fmt.Errorf("素材缺少数量图物品")
		}
		if operation == "sub" {
			e.Prompt = fmt.Sprintf("原来有 %d 个，拿走 %d 个，还剩几个？", a, b)
		} else {
			e.Prompt = fmt.Sprintf("一组 %d 个，另一组 %d 个，一共有几个？", a, b)
		}
	case "judgement":
		if len(source.Example.Statements) > 1 {
			e.Prompt = "找出计算错误的一项"
			e.Statements = nil
			badIndex := sequence % 3
			for j := 0; j < 3; j++ {
				x := (a + j) % (max + 1)
				y := rand.IntN(x + 1)
				r := x - y
				if operation == "add" {
					y = rand.IntN(max - x + 1)
					r = x + y
				}
				if j == badIndex {
					r = (r + 1) % (max + 1)
				}
				statement := fmt.Sprintf("%d %s %d = %d", x, symbol, y, r)
				e.Statements = append(e.Statements, statement)
				if j == badIndex {
					e.Answer = statement
				}
			}
			e.Options = append([]string{}, e.Statements...)
		} else {
			right := sequence%2 == 0
			r := result
			if !right {
				r = (r + 1) % (max + 1)
			}
			e.Prompt = "判断下面的算式"
			e.Statements = []string{fmt.Sprintf("%d %s %d = %d", a, symbol, b, r)}
			e.Options = []string{"对", "错"}
			e.Answer = "错"
			if right {
				e.Answer = "对"
			}
		}
	default:
		return d, fmt.Errorf("素材 %s 的题面不适用于算术生成", d.ID)
	}
	if e.Kind != "judgement" {
		e.Answer = strconv.Itoa(answer)
		e.Options = numberOptions(answer, max)
	}
	d.ModuleTitle = fmt.Sprintf("%d 以内%s", max, map[string]string{"addition": "加法", "subtraction": "减法"}[d.GroupID])
	stampOptionIDs(e)
	return d, validateGenerated(d, max)
}
func stampOptionIDs(e *mathcontent.MathExample) {
	if len(e.Options) == 0 {
		e.OptionIDs = nil
		e.AnswerOptionID = ""
		return
	}
	e.OptionIDs = make([]string, len(e.Options))
	e.AnswerOptionID = ""
	for i, option := range e.Options {
		e.OptionIDs[i] = fmt.Sprintf("o%d", i+1)
		if option == e.Answer {
			e.AnswerOptionID = e.OptionIDs[i]
		}
	}
}
func numberOptions(answer, max int) []string {
	values := []int{answer}
	for _, v := range rand.Perm(max + 1) {
		if v != answer {
			values = append(values, v)
		}
		if len(values) == 4 {
			break
		}
	}
	rand.Shuffle(len(values), func(i, j int) { values[i], values[j] = values[j], values[i] })
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strconv.Itoa(v)
	}
	return out
}
func validateGenerated(d mathcontent.MathDetail, max int) error {
	if err := mathcontent.Validate(d); err != nil {
		return err
	}
	if d.GroupID == "shape" {
		return nil
	}
	e := d.Example
	if len(e.Counts) != 2 {
		return fmt.Errorf("算术题缺少运算参数")
	}
	a, b := e.Counts[0], e.Counts[1]
	r := a + b
	if d.GroupID == "subtraction" {
		r = a - b
		if e.Operation != "sub" {
			return fmt.Errorf("减法运算不一致")
		}
	} else if e.Operation != "add" {
		return fmt.Errorf("加法运算不一致")
	}
	if a < 0 || b < 0 || a > max || b > max || r < 0 || r > max {
		return fmt.Errorf("算术题超出指定范围")
	}
	if e.Kind == "judgement" {
		wrong := []string{}
		for _, s := range e.Statements {
			var x, y, z int
			normalized := strings.ReplaceAll(s, "−", "-")
			op := "+"
			if e.Operation == "sub" {
				op = "-"
			}
			if n, err := fmt.Sscanf(normalized, "%d "+op+" %d = %d", &x, &y, &z); err != nil || n != 3 {
				return fmt.Errorf("无效判断算式")
			}
			actual := x + y
			if op == "-" {
				actual = x - y
			}
			if x < 0 || y < 0 || x > max || y > max || actual < 0 || actual > max || z < 0 || z > max {
				return fmt.Errorf("判断算式超出范围")
			}
			if actual != z {
				wrong = append(wrong, s)
			}
		}
		if len(e.Statements) == 1 {
			answer := "对"
			if len(wrong) == 1 {
				answer = "错"
			}
			if e.Answer != answer {
				return fmt.Errorf("判断答案不正确")
			}
		} else if len(wrong) != 1 || e.Answer != wrong[0] {
			return fmt.Errorf("须且仅有一个错误算式")
		}
	} else {
		answer := r
		if e.Kind == "missing" {
			answer = b
		}
		if e.Answer != strconv.Itoa(answer) {
			return fmt.Errorf("运算答案不正确")
		}
	}
	return nil
}
