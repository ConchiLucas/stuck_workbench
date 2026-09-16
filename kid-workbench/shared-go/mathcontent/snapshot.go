package mathcontent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type PlanVisual struct {
	Kind       string `json:"kind"`
	A          int    `json:"a"`
	B          int    `json:"b"`
	Operator   string `json:"operator"`
	LeftCount  int    `json:"leftCount"`
	RightCount int    `json:"rightCount"`
	Object     string `json:"object"`
	Shape      string `json:"shape"`
}

type PlanSnapshot struct {
	QuestionID     int64           `json:"questionId"`
	Code           string          `json:"code"`
	Stem           string          `json:"stem"`
	Options        json.RawMessage `json:"options"`
	AnswerIndex    int             `json:"answerIndex"`
	Visual         PlanVisual      `json:"visual"`
	AudioObjectKey string          `json:"audioObjectKey"`
}

type PlanExample struct {
	Example        MathExample
	Selected       string
	AudioObjectKey string
	AudioMutable   bool
}

type planOption struct {
	Label string `json:"label"`
	Shape string `json:"shape"`
}

func LastPickIndex(picks string) int {
	picks = strings.TrimSpace(picks)
	if picks == "" {
		return -1
	}
	parts := strings.Split(picks, ",")
	n, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1]))
	if err != nil || n < 0 {
		return -1
	}
	return n
}

func PlanExampleFromSnapshot(raw string, pickIndex int) (PlanExample, error) {
	var snap PlanSnapshot
	if json.Unmarshal([]byte(raw), &snap) != nil || snap.Code == "" {
		return PlanExample{}, fmt.Errorf("invalid snapshot")
	}
	opts := []planOption{}
	if json.Unmarshal(snap.Options, &opts) != nil || len(opts) == 0 {
		return PlanExample{}, fmt.Errorf("invalid snapshot options")
	}
	labels := make([]string, len(opts))
	keys := make([]string, len(opts))
	ids := make([]string, len(opts))
	for i, o := range opts {
		ids[i] = fmt.Sprintf("o%d", i+1)
		key := o.Shape
		if key == "" {
			key = shapeAliases[o.Label]
		}
		keys[i] = key
		labels[i] = o.Label
		if labels[i] == "" && key != "" && shapeNames[key] != "" {
			labels[i] = shapeNames[key]
		}
		if labels[i] == "" {
			return PlanExample{}, fmt.Errorf("invalid snapshot options")
		}
	}
	if snap.AnswerIndex < 0 || snap.AnswerIndex >= len(labels) {
		return PlanExample{}, fmt.Errorf("invalid snapshot answer")
	}
	e := MathExample{Prompt: strings.TrimSpace(snap.Stem), Options: labels, Answer: labels[snap.AnswerIndex], OptionIDs: ids, AnswerOptionID: ids[snap.AnswerIndex]}
	switch snap.Code {
	case "calc":
		op := snap.Visual.Operator
		if op == "-" {
			op = "−"
		}
		if op == "" {
			op = "+"
		}
		e.Kind = "choice"
		e.Prompt = fmt.Sprintf("%d %s %d", snap.Visual.A, op, snap.Visual.B)
		e.Counts = []int{snap.Visual.A, snap.Visual.B}
		if op == "−" {
			e.Operation = "sub"
		} else {
			e.Operation = "add"
		}
	case "story":
		e.Kind = "objects"
		e.Counts = []int{snap.Visual.LeftCount, snap.Visual.RightCount}
		e.Object = snap.Visual.Object
		if e.Object == "apple" {
			e.Object = "苹果"
		}
		if e.Object == "strawberry" {
			e.Object = "草莓"
		}
		if snap.Visual.Kind == "sub" {
			e.Operation = "sub"
			e.Prompt = fmt.Sprintf("原来有 %d 个，拿走 %d 个，还剩几个？", e.Counts[0], e.Counts[1])
		} else {
			e.Operation = "add"
			e.Prompt = fmt.Sprintf("一组 %d 个，另一组 %d 个，一共有几个？", e.Counts[0], e.Counts[1])
		}
	case "find":
		e.Kind = "audio-shape"
		e.Prompt = "听一听，选出图形"
		e.ShapeKeys = keys
		for i, key := range keys {
			if key == "" {
				return PlanExample{}, fmt.Errorf("invalid snapshot options")
			}
			if labels[i] == "" {
				e.Options[i] = shapeNames[key]
			}
		}
		e.Answer = e.Options[snap.AnswerIndex]
	case "name":
		e.Kind = "shape-name"
		e.Prompt = "看一看，这是什么图形？"
		shape := snap.Visual.Shape
		if shape == "" {
			shape = keys[snap.AnswerIndex]
		}
		e.ShapeKeys = []string{shape}
	default:
		return PlanExample{}, fmt.Errorf("unsupported snapshot code")
	}
	out := PlanExample{Example: e, AudioObjectKey: strings.TrimSpace(snap.AudioObjectKey), AudioMutable: strings.HasPrefix(strings.TrimSpace(snap.AudioObjectKey), "math/questions/")}
	if pickIndex >= 0 && pickIndex < len(e.Options) {
		out.Selected = e.Options[pickIndex]
	}
	return out, nil
}
