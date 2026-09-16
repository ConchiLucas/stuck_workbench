// Package pinyincontract defines the public, answer-free quiz contract.
package pinyincontract

import "time"

type Visual struct {
	Kind     string `json:"kind"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
	Initial  string `json:"initial,omitempty"`
	Final    string `json:"final,omitempty"`
	Syllable string `json:"syllable,omitempty"`
}
type Option struct {
	ID         string `json:"id"`
	Label      string `json:"label,omitempty"`
	SpeechText string `json:"speechText,omitempty"`
	SpeechURL  string `json:"speechUrl,omitempty"`
}
type GeneratedQuestion struct {
	InstanceID string    `json:"instanceId"`
	Type       string    `json:"type"`
	TargetID   int64     `json:"targetId"`
	KpID       int64     `json:"kpId"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Stem       string    `json:"stem"`
	SpeechText string    `json:"speechText,omitempty"`
	SpeechURL  string    `json:"speechUrl,omitempty"`
	Visual     Visual    `json:"visual"`
	Options    []Option  `json:"options"`
}
type AnswerRequest struct {
	ClientID string `json:"clientId"`
	OptionID string `json:"optionId"`
	CostMs   int    `json:"costMs"`
}
type SkillResult struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}
type KnowledgeResult struct {
	KpID          int64  `json:"kpId"`
	Status        string `json:"status"`
	NewlyMastered bool   `json:"newlyMastered"`
}
type AnswerResult struct {
	InstanceID       string          `json:"instanceId"`
	AttemptID        int64           `json:"attemptId"`
	SelectedOptionID string          `json:"selectedOptionId"`
	Correct          bool            `json:"correct"`
	AnswerOptionID   string          `json:"answerOptionId"`
	Skill            SkillResult     `json:"skill"`
	Knowledge        KnowledgeResult `json:"knowledge"`
}
type InstanceSnapshot struct {
	GeneratedQuestion
	AcceptedResult *AnswerResult `json:"acceptedResult,omitempty"`
}
