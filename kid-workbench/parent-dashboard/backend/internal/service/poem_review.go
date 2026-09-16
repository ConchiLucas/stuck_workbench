package service

import (
	"strings"

	"github.com/conchi/study-learning/poemcontent"
)

type PoemReview struct {
	Example           *poemcontent.PoemExample `json:"example,omitempty"`
	Selected          string                    `json:"selected,omitempty"`
	ResponseKind      string                     `json:"response_kind,omitempty"`
	Facts             []string                    `json:"facts,omitempty"`
	UnavailableReason string                     `json:"unavailable_reason,omitempty"`
}

func buildPoemReview(raw, picks string) *PoemReview {
	if strings.TrimSpace(raw) == "" {
		return &PoemReview{UnavailableReason: "这条记录未保存当时的题目画面。"}
	}
	converted, err := poemcontent.PlanExampleFromSnapshot(raw, poemcontent.AttemptSelected("", picks))
	if err != nil {
		return &PoemReview{UnavailableReason: "当时题目快照无法还原。"}
	}
	example := converted.Example
	poemcontent.RewriteMediaURLs(&example, rewritePoemMedia)
	facts := poemcontent.ErrorFacts(example, converted.Input)
	return &PoemReview{Example: &example, Selected: converted.Selected, ResponseKind: converted.ResponseKind, Facts: facts}
}

func rewritePoemMedia(url string) string {
	const from = "/api/v1/poem/task-media/"
	const to = "/api/poem/task-media/"
	if strings.HasPrefix(url, from) {
		return to + strings.TrimPrefix(url, from)
	}
	return url
}
