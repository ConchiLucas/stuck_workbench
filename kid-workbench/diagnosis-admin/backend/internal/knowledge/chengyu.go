package knowledge

import (
	"fmt"

	"github.com/conchi/study-learning/chengyucontent"
	"github.com/conchi/study-learning/mastery"
)

func decodeChengyu(r *Evidence, row mathRow) {
	r.Source = SourceRef{Kind: "chengyu_plan", PlanID: row.PlanID, ItemID: row.ItemID}
	r.PlanID = row.PlanID
	r.PlanItemID = row.ItemID
	r.InstanceKey = fmt.Sprintf("chengyu:%d", r.AttemptID)
	if row.SkillCode != "" {
		r.SkillCode = row.SkillCode
		r.QuestionType = row.SkillCode
		r.SkillLabel = Label(r.SubjectCode, row.SkillCode)
	}
	allowed := mastery.ChengyuSkills
	if row.ItemID == 0 || row.Snapshot == "" {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "unlinked_plan_item")
		return
	}
	if r.SubjectCode != "chengyu" || (len(allowed) > 0 && !hasString(allowed, r.SkillCode)) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	converted, err := chengyucontent.PlanExampleFromSnapshot(row.Snapshot, chengyucontent.AttemptSelected("", row.Picks))
	if err != nil {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	example := converted.Example
	frozen := chengyucontent.HasFrozenMedia(example)
	if example.SpeechURL != "" && frozen {
		r.Media["stem-audio"] = "chengyu:" + example.SpeechURL
	}
	if !frozen {
		example.SpeechURL = ""
	}
	opts := make([]Option, 0, len(example.Options))
	for _, o := range example.Options {
		opts = append(opts, Option{ID: o.ID, Label: o.Label})
	}
	audioID := ""
	if example.SpeechURL != "" {
		audioID = "stem-audio"
	}
	r.ChengyuExample = &example
	r.Question = &QuestionView{
		Interaction: converted.ResponseKind, TargetText: r.Title,
		Stem: Stem{Text: example.Prompt, AudioMediaID: audioID},
		Options: opts, AnswerOptionID: example.AnswerID,
	}
	r.QuestionFidelity = "instance_snapshot"
	r.MediaFidelity = "immutable"
	if !frozen {
		r.MediaFidelity = "mutable_reference"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_not_frozen")
	}
	r.ReviewBlockReasons = []string{"unsupported_subject"}
	if converted.Selected == "" {
		return
	}
	r.SelectionFidelity = "stable_option"
	r.Response = ResponseView{Kind: converted.ResponseKind, SelectedOptionID: converted.Selected, Value: converted.Selected}
	correct := converted.Selected == example.AnswerID
	if r.IsCorrect != correct {
		r.Assistance = "unknown"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
	} else {
		r.Assistance = "none"
	}
}
