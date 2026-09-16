package service

import (
	"strings"

	"github.com/conchi/study-learning/englishcontent"
)

type EnglishReview struct {
	Example           *englishcontent.EnglishExample `json:"example,omitempty"`
	Selected          string                            `json:"selected,omitempty"`
	ResponseKind      string                            `json:"response_kind,omitempty"`
	UnavailableReason string                            `json:"unavailable_reason,omitempty"`
}

func buildEnglishReview(raw, picks string) *EnglishReview {
	if strings.TrimSpace(raw) == "" {
		return &EnglishReview{UnavailableReason: "这条记录未保存当时的题目画面。"}
	}
	converted, err := englishcontent.PlanExampleFromSnapshot(raw, englishcontent.AttemptSelected("", picks))
	if err != nil {
		return &EnglishReview{UnavailableReason: "当时题目快照无法还原。"}
	}
	example := converted.Example
	example.SpeechURL = rewriteEnglishMedia(example.SpeechURL)
	example.Cue = rewriteEnglishMedia(example.Cue)
	for i := range example.Options {
		example.Options[i].Picture = rewriteEnglishMedia(example.Options[i].Picture)
	}
	return &EnglishReview{Example: &example, Selected: converted.Selected, ResponseKind: converted.ResponseKind}
}

func rewriteEnglishMedia(url string) string {
	replacements := [][2]string{
		{"/api/v1/english/task-media/", "/api/english/task-media/"},
		{"/api/v1/english/words/", "/api/english/words/"},
		{"/api/v1/english/sentences/", "/api/english/sentences/"},
	}
	for _, pair := range replacements {
		if strings.HasPrefix(url, pair[0]) {
			return pair[1] + strings.TrimPrefix(url, pair[0])
		}
	}
	return url
}
