package service

import (
	"strings"

	"github.com/conchi/study-learning/chengyucontent"
)

type ChengyuReview struct {
	Example           *chengyucontent.ChengyuExample `json:"example,omitempty"`
	Selected          string                         `json:"selected,omitempty"`
	ResponseKind      string                         `json:"response_kind,omitempty"`
	UnavailableReason string                         `json:"unavailable_reason,omitempty"`
}

func buildChengyuReview(raw, picks string) *ChengyuReview {
	if strings.TrimSpace(raw) == "" {
		return &ChengyuReview{UnavailableReason: "这条记录未保存当时的题目画面。"}
	}
	converted, err := chengyucontent.PlanExampleFromSnapshot(raw, chengyucontent.AttemptSelected("", picks))
	if err != nil {
		return &ChengyuReview{UnavailableReason: "当时题目快照无法还原。"}
	}
	example := converted.Example
	example.SpeechURL = rewriteChengyuMedia(example.SpeechURL)
	review := &ChengyuReview{Example: &example, Selected: converted.Selected, ResponseKind: converted.ResponseKind}
	if converted.Example.Kind == "meaning" && example.SpeechURL == "" {
		review.UnavailableReason = "当时的读音未冻结，无法可靠还原音频；题目和作答仍可查看。"
	}
	return review
}

func rewriteChengyuMedia(url string) string {
	const from = "/api/v1/chengyu/task-media/"
	const to = "/api/chengyu/task-media/"
	if strings.HasPrefix(url, from) {
		return to + strings.TrimPrefix(url, from)
	}
	return ""
}
