package service

import (
	"strings"

	"github.com/conchi/study-learning/phrasecontent"
)

type PhraseReview struct {
	Example           *phrasecontent.PhraseExample `json:"example,omitempty"`
	Selected          string                       `json:"selected,omitempty"`
	ResponseKind      string                       `json:"response_kind,omitempty"`
	UnavailableReason string                       `json:"unavailable_reason,omitempty"`
}

func buildPhraseReview(raw, picks string) *PhraseReview {
	if strings.TrimSpace(raw) == "" {
		return &PhraseReview{UnavailableReason: "这条记录未保存当时的题目画面。"}
	}
	converted, err := phrasecontent.PlanExampleFromSnapshot(raw, phrasecontent.AttemptSelected("", picks))
	if err != nil {
		return &PhraseReview{UnavailableReason: "当时题目快照无法还原。"}
	}
	example := converted.Example
	example.SpeechURL = rewritePhraseMedia(example.SpeechURL)
	return &PhraseReview{Example: &example, Selected: converted.Selected, ResponseKind: converted.ResponseKind}
}

func rewritePhraseMedia(url string) string {
	const from = "/api/v1/phrase/task-media/"
	const to = "/api/phrase/task-media/"
	if strings.HasPrefix(url, from) {
		return to + strings.TrimPrefix(url, from)
	}
	return url
}
