package practice

import "github.com/conchi/study-learning/learning"

type AnswerInput struct {
	ClientID    string `json:"clientId"`
	OptionIndex int    `json:"optionIndex"`
	CostMs      int    `json:"costMs"`
}

type PlanSummary struct {
	Status       string `json:"status"`
	TargetCount  int    `json:"targetCount"`
	DoneCount    int    `json:"doneCount"`
	CorrectCount int    `json:"correctCount"`
}

type AnswerResult struct {
	Correct     bool              `json:"correct"`
	AnswerIndex int               `json:"answerIndex"`
	CanRetry    bool              `json:"canRetry"`
	Tries       int               `json:"tries"`
	Status      string            `json:"status"`
	Mastery     learning.StateDTO `json:"mastery"`
	Plan        PlanSummary       `json:"plan"`
}
