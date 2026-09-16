package service

import (
	"strings"

	"github.com/conchi/study-learning/logiccontent"
)

type LogicReview struct {
	Example           *logiccontent.LogicExample `json:"example,omitempty"`
	Selected          string                       `json:"selected,omitempty"`
	ResponseKind      string                       `json:"response_kind,omitempty"`
	Facts             []string                     `json:"facts,omitempty"`
	UnavailableReason string                       `json:"unavailable_reason,omitempty"`
}

func buildLogicReview(raw, picks string) *LogicReview {
	if strings.TrimSpace(raw) == "" {
		return &LogicReview{UnavailableReason: "这条记录未保存当时的题目画面。"}
	}
	converted, err := logiccontent.PlanExampleFromSnapshot(raw, picks)
	if err != nil {
		return &LogicReview{UnavailableReason: "当时题目快照无法还原。"}
	}
	example := converted.Example
	logiccontent.RewriteImageURLs(&example, rewriteLogicMedia)
	facts := logiccontent.Facts(example, converted.Input)
	return &LogicReview{Example: &example, Selected: converted.Selected, ResponseKind: converted.ResponseKind, Facts: facts}
}

func rewriteLogicMedia(url string) string {
	const from = "/api/v1/logic/task-media/"
	const to = "/api/logic/task-media/"
	if strings.HasPrefix(url, from) {
		return to + strings.TrimPrefix(url, from)
	}
	return url
}
