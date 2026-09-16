package service

import (
	"fmt"
	"strings"

	"github.com/conchi/study-learning/mathcontent"
)

type MathReview struct {
	Example           *mathcontent.MathExample `json:"example,omitempty"`
	Selected          string                   `json:"selected,omitempty"`
	UnavailableReason string                   `json:"unavailable_reason,omitempty"`
	AudioMutable      bool                     `json:"audio_mutable,omitempty"`
}

func buildMathReview(raw, picks string, childID, planID, itemID int64) *MathReview {
	if strings.TrimSpace(raw) == "" {
		return &MathReview{UnavailableReason: "这条记录未保存当时的题目画面。"}
	}
	converted, err := mathcontent.PlanExampleFromSnapshot(raw, mathcontent.LastPickIndex(picks))
	if err != nil {
		return &MathReview{UnavailableReason: "当时题目快照无法还原。"}
	}
	example := converted.Example
	if example.Kind == "audio-shape" {
		if converted.AudioObjectKey == "" {
			return &MathReview{UnavailableReason: "听音图形缺少题目音频。"}
		}
		if childID > 0 && planID > 0 && itemID > 0 {
			example.AudioURL = fmt.Sprintf("/api/math/children/%d/plans/%d/items/%d/speech.mp3", childID, planID, itemID)
		}
	} else {
		example.AudioURL = ""
	}
	return &MathReview{Example: &example, Selected: converted.Selected, AudioMutable: converted.AudioMutable && example.Kind == "audio-shape"}
}
