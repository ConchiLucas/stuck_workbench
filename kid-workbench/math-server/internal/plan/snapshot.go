package plan

import (
	"encoding/json"
	"fmt"
	"strings"
)

var validShapes = map[string]bool{
	"circle": true, "square": true, "rect": true, "triangle": true,
	"oval": true, "trapezoid": true, "rhombus": true, "star": true,
}

func BuildSnapshot(candidate Candidate) (QuestionSnapshot, error) {
	var options []json.RawMessage
	if err := json.Unmarshal([]byte(candidate.Options), &options); err != nil || len(options) != 4 {
		return QuestionSnapshot{}, fmt.Errorf("question %d must have four options", candidate.QuestionID)
	}
	var answer struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(candidate.Answer), &answer); err != nil || answer.Index < 0 || answer.Index >= len(options) {
		return QuestionSnapshot{}, fmt.Errorf("question %d has invalid answer", candidate.QuestionID)
	}
	wantKey := fmt.Sprintf("math/questions/%d.mp3", candidate.QuestionID)
	if strings.TrimSpace(candidate.MediaURL) != wantKey {
		return QuestionSnapshot{}, fmt.Errorf("question %d has unpublished audio", candidate.QuestionID)
	}
	visual, err := normalizeVisual(candidate)
	if err != nil {
		return QuestionSnapshot{}, err
	}
	return QuestionSnapshot{
		QuestionID: candidate.QuestionID, Code: candidate.Code, Stem: candidate.Stem,
		Options: json.RawMessage(candidate.Options), AnswerIndex: answer.Index,
		Visual: visual, AudioObjectKey: candidate.MediaURL,
	}, nil
}

func normalizeVisual(candidate Candidate) (Visual, error) {
	var raw struct {
		Kind  string `json:"kind"`
		A     int    `json:"a"`
		B     int    `json:"b"`
		Emoji string `json:"emoji"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal([]byte(candidate.Visual), &raw); err != nil {
		return Visual{}, fmt.Errorf("question %d has invalid visual", candidate.QuestionID)
	}
	switch candidate.Code {
	case "calc":
		operator := "+"
		if candidate.ModuleCode == "sub10" && raw.Kind == "sub" {
			operator = "-"
		} else if candidate.ModuleCode != "add10" || raw.Kind != "add" {
			return Visual{}, fmt.Errorf("question %d has mismatched calc visual", candidate.QuestionID)
		}
		return Visual{Kind: "equation", A: raw.A, B: raw.B, Operator: operator}, nil
	case "story":
		if (candidate.ModuleCode == "add10" && raw.Kind != "add") ||
			(candidate.ModuleCode == "sub10" && raw.Kind != "sub") {
			return Visual{}, fmt.Errorf("question %d has mismatched story visual", candidate.QuestionID)
		}
		object := map[string]string{"🍎": "apple", "🍓": "strawberry"}[raw.Emoji]
		if object == "" {
			return Visual{}, fmt.Errorf("question %d has unsupported count object", candidate.QuestionID)
		}
		return Visual{Kind: raw.Kind, LeftCount: raw.A, RightCount: raw.B, Object: object}, nil
	case "find":
		if candidate.ModuleCode != "shape" {
			return Visual{}, fmt.Errorf("question %d has invalid find module", candidate.QuestionID)
		}
		return Visual{Kind: "none"}, nil
	case "name":
		if candidate.ModuleCode != "shape" || raw.Kind != "shape" || !validShapes[raw.Text] {
			return Visual{}, fmt.Errorf("question %d has invalid shape visual", candidate.QuestionID)
		}
		return Visual{Kind: "shape", Shape: raw.Text}, nil
	default:
		return Visual{}, fmt.Errorf("question %d has unsupported code %q", candidate.QuestionID, candidate.Code)
	}
}

func PublicQuestion(snapshot QuestionSnapshot, audioURL string) QuestionDTO {
	return QuestionDTO{
		QuestionID: snapshot.QuestionID, Code: snapshot.Code, Stem: snapshot.Stem,
		Options: snapshot.Options, Visual: snapshot.Visual, AudioURL: audioURL,
	}
}
