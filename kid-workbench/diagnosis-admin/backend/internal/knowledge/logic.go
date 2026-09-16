package knowledge

import (
	"fmt"

	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-learning/mastery"
)

func decodeLogic(r *Evidence, row mathRow) {
	r.Source = SourceRef{Kind: "logic_plan", PlanID: row.PlanID, ItemID: row.ItemID}
	r.PlanID = row.PlanID
	r.PlanItemID = row.ItemID
	r.InstanceKey = fmt.Sprintf("logic:%d", r.AttemptID)
	skill := mastery.SkillFromQuestionCode("logic", row.SkillCode)
	if skill != "" {
		r.SkillCode = skill
		r.QuestionType = skill
		r.SkillLabel = Label(r.SubjectCode, skill)
	}
	if row.ItemID == 0 || row.Snapshot == "" {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "unlinked_plan_item")
		return
	}
	if r.SubjectCode != "logic" || !mastery.SkillIsTracked("logic", r.ModuleCode, r.SkillCode) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	converted, err := logiccontent.PlanExampleFromSnapshot(row.Snapshot, row.Picks)
	if err != nil {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	example := converted.Example
	attachLogicMedia(r, example)
	opts := make([]Option, 0, len(example.Options))
	for _, id := range example.Options {
		o, _ := logiccontent.ObjectByID(example.Objects, id)
		opts = append(opts, Option{ID: id, Label: o.Caption, ImageMediaID: mediaIDForLogicURL(r, example.ImageURLs[id])})
	}
	r.LogicExample = &example
	r.Question = &QuestionView{
		Interaction: converted.ResponseKind, TargetText: r.Title,
		Stem:    Stem{Text: example.Prompt},
		Options: opts, AnswerOptionID: example.AnswerID,
	}
	r.QuestionFidelity = "instance_snapshot"
	r.MediaFidelity = "immutable"
	if !logiccontent.HasFrozenMedia(example) {
		r.MediaFidelity = "mutable_reference"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_not_frozen")
	}
	r.ReviewBlockReasons = []string{"unsupported_subject"}
	if converted.Selected == "" {
		return
	}
	r.SelectionFidelity = "stable_option"
	if converted.ResponseKind == "order" {
		r.SelectionFidelity = "stable_mapping"
	}
	r.Response = ResponseView{Kind: converted.ResponseKind, SelectedOptionID: converted.Selected, Value: converted.Selected}
	correct := logiccontent.IsCorrect(example, converted.Input)
	if r.IsCorrect != correct {
		r.Assistance = "unknown"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
	} else {
		r.Assistance = "none"
	}
}

func attachLogicMedia(r *Evidence, example logiccontent.LogicExample) {
	for id, url := range example.ImageURLs {
		if url != "" {
			r.Media["glyph-"+id] = "logic:" + url
		}
	}
}

func mediaIDForLogicURL(r *Evidence, url string) string {
	if url == "" {
		return ""
	}
	want := "logic:" + url
	for id, raw := range r.Media {
		if raw == want {
			return id
		}
	}
	return ""
}
