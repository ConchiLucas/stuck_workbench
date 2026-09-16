package knowledge

import (
	"fmt"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/poemcontent"
)

func decodePoem(r *Evidence, row mathRow) {
	r.Source = SourceRef{Kind: "poem_plan", PlanID: row.PlanID, ItemID: row.ItemID}
	r.PlanID = row.PlanID
	r.PlanItemID = row.ItemID
	r.InstanceKey = fmt.Sprintf("poem:%d", r.AttemptID)
	skill := mastery.SkillFromQuestionCode("poem", row.SkillCode)
	if skill != "" {
		r.SkillCode = skill
		r.QuestionType = skill
		r.SkillLabel = Label(r.SubjectCode, skill)
	}
	if row.ItemID == 0 || row.Snapshot == "" {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "unlinked_plan_item")
		return
	}
	if r.SubjectCode != "poem" || !mastery.SkillIsTracked("poem", r.ModuleCode, r.SkillCode) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	converted, err := poemcontent.PlanExampleFromSnapshot(row.Snapshot, poemcontent.AttemptSelected("", row.Picks))
	if err != nil {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	example := converted.Example
	attachPoemMedia(r, example)
	opts := make([]Option, 0, len(example.Options))
	for _, o := range example.Options {
		opts = append(opts, Option{ID: o.ID, Label: o.Label})
	}
	r.PoemExample = &example
	r.Question = &QuestionView{
		Interaction: converted.ResponseKind, TargetText: r.Title,
		Stem:    Stem{Text: example.Prompt, AudioMediaID: mediaIDForPoemURL(r, example.SpeechURL)},
		Options: opts, AnswerOptionID: example.AnswerID,
	}
	r.QuestionFidelity = "instance_snapshot"
	r.MediaFidelity = "immutable"
	if !poemcontent.HasFrozenMedia(example) {
		r.MediaFidelity = "mutable_reference"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_not_frozen")
	}
	r.ReviewBlockReasons = []string{"unsupported_subject"}
	if converted.Selected == "" {
		return
	}
	r.SelectionFidelity = "stable_option"
	if converted.ResponseKind == poemcontent.KindRecite {
		r.SelectionFidelity = "stable_mapping"
	}
	r.Response = ResponseView{Kind: converted.ResponseKind, SelectedOptionID: converted.Selected, Value: converted.Selected}
	correct := poemcontent.IsCorrect(example, converted.Input)
	if r.IsCorrect != correct {
		r.Assistance = "unknown"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
	} else {
		r.Assistance = "none"
	}
}

func attachPoemMedia(r *Evidence, example poemcontent.PoemExample) {
	if example.SpeechURL != "" {
		r.Media["stem-audio"] = "poem:" + example.SpeechURL
	}
}

func mediaIDForPoemURL(r *Evidence, url string) string {
	if url == "" {
		return ""
	}
	want := "poem:" + url
	for id, raw := range r.Media {
		if raw == want {
			return id
		}
	}
	return ""
}
