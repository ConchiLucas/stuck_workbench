package knowledge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/conchi/study-learning/englishcontent"
	"github.com/conchi/study-learning/mastery"
)

func decodeEnglish(r *Evidence, row mathRow) {
	r.Source = SourceRef{Kind: "english_plan", PlanID: row.PlanID, ItemID: row.ItemID}
	r.PlanID = row.PlanID
	r.PlanItemID = row.ItemID
	r.InstanceKey = fmt.Sprintf("english:%d", r.AttemptID)
	if row.SkillCode != "" {
		r.SkillCode = row.SkillCode
		r.QuestionType = row.SkillCode
		r.SkillLabel = Label(r.SubjectCode, row.SkillCode)
	}
	allowed := mastery.SkillsFor("english", r.ModuleCode)
	if row.ItemID == 0 || row.Snapshot == "" {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "unlinked_plan_item")
		return
	}
	if r.SubjectCode != "english" || (len(allowed) > 0 && !hasString(allowed, r.SkillCode)) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	converted, err := englishcontent.PlanExampleFromSnapshot(row.Snapshot, englishcontent.AttemptSelected("", row.Picks))
	if err != nil {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	example := converted.Example
	if example.SpeechURL != "" {
		r.Media["stem-audio"] = "english:" + example.SpeechURL
	}
	if isHTTPPath(example.Cue) {
		r.Media["stem-image"] = "english:" + example.Cue
	}
	opts := make([]Option, 0, len(example.Options))
	for i, o := range example.Options {
		imageID := ""
		if isHTTPPath(o.Picture) {
			imageID = fmt.Sprintf("option-%d-image", i)
			r.Media[imageID] = "english:" + o.Picture
		}
		opts = append(opts, Option{ID: o.ID, Label: o.Label, ImageMediaID: imageID})
	}
	audioID := ""
	if example.SpeechURL != "" {
		audioID = "stem-audio"
	}
	imageID := ""
	if isHTTPPath(example.Cue) {
		imageID = "stem-image"
	}
	r.EnglishExample = &example
	r.Question = &QuestionView{
		Interaction: converted.ResponseKind, TargetText: r.Title,
		Stem:    Stem{Text: example.Prompt, AudioMediaID: audioID, ImageMediaID: imageID},
		Options: opts, AnswerOptionID: example.AnswerID,
	}
	if converted.ResponseKind == "input" {
		r.Question.TargetText = example.Answer
	}
	if converted.ResponseKind == "order" {
		r.Question.TargetText = example.Answer
		if len(opts) == 0 {
			for i, token := range example.Bank {
				opts = append(opts, Option{ID: "bank-" + strconv.Itoa(i), Label: token})
			}
			r.Question.Options = opts
		}
	}
	r.QuestionFidelity = "instance_snapshot"
	r.MediaFidelity = "immutable"
	if !englishcontent.HasFrozenMedia(example) {
		r.MediaFidelity = "mutable_reference"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_not_frozen")
	}
	r.ReviewBlockReasons = []string{"unsupported_subject"}
	if converted.Selected == "" {
		return
	}
	r.SelectionFidelity = "stable_option"
	r.Response = ResponseView{Kind: converted.ResponseKind, SelectedOptionID: converted.Selected, Value: converted.Selected}
	correct := false
	switch converted.ResponseKind {
	case "choice":
		correct = converted.Selected == example.AnswerID
	case "input":
		correct = strings.EqualFold(strings.TrimSpace(converted.Selected), example.Answer)
	default:
		correct = converted.Selected == example.Answer
	}
	if r.IsCorrect != correct {
		r.Assistance = "unknown"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
	} else {
		r.Assistance = "none"
	}
}

func isHTTPPath(value string) bool {
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
