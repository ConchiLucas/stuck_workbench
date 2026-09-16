package service

import (
	"strings"

	"github.com/conchi/study-learning/sciencecontent"
)

type ScienceReview struct {
	Example           *sciencecontent.ScienceExample `json:"example,omitempty"`
	Selected          string                         `json:"selected,omitempty"`
	ResponseKind      string                         `json:"response_kind,omitempty"`
	Facts             []string                       `json:"facts,omitempty"`
	UnavailableReason string                         `json:"unavailable_reason,omitempty"`
}

func buildScienceReview(raw, picks string) *ScienceReview {
	if strings.TrimSpace(raw) == "" {
		return &ScienceReview{UnavailableReason: "这条记录未保存当时的题目画面。"}
	}
	converted, err := sciencecontent.PlanExampleFromSnapshot(raw, sciencecontent.AttemptSelected("", picks))
	if err != nil {
		return &ScienceReview{UnavailableReason: "当时题目快照无法还原。"}
	}
	example := converted.Example
	sciencecontent.RewriteImageURLs(&example, rewriteScienceMedia)
	facts := sciencecontent.ErrorFacts(example, converted.Input)
	return &ScienceReview{Example: &example, Selected: converted.Selected, ResponseKind: converted.ResponseKind, Facts: facts}
}

func rewriteScienceMedia(url string) string {
	const from = "/api/v1/science/task-media/"
	const to = "/api/science/task-media/"
	if strings.HasPrefix(url, from) {
		return to + strings.TrimPrefix(url, from)
	}
	return url
}
