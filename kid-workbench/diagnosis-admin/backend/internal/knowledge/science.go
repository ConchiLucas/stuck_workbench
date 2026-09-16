package knowledge

import (
	"fmt"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/sciencecontent"
)

func decodeScience(r *Evidence, row mathRow) {
	r.Source = SourceRef{Kind: "science_plan", PlanID: row.PlanID, ItemID: row.ItemID}
	r.PlanID = row.PlanID
	r.PlanItemID = row.ItemID
	r.InstanceKey = fmt.Sprintf("science:%d", r.AttemptID)
	skill := mastery.SkillFromQuestionCode("science", row.SkillCode)
	if skill != "" {
		r.SkillCode = skill
		r.QuestionType = skill
		r.SkillLabel = Label(r.SubjectCode, skill)
	}
	if row.ItemID == 0 || row.Snapshot == "" {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "unlinked_plan_item")
		return
	}
	if r.SubjectCode != "science" || !mastery.SkillIsTracked("science", r.ModuleCode, r.SkillCode) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	converted, err := sciencecontent.PlanExampleFromSnapshot(row.Snapshot, sciencecontent.AttemptSelected("", row.Picks))
	if err != nil {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	example := converted.Example
	attachScienceMedia(r, example)
	opts := make([]Option, 0, len(example.Options))
	for _, o := range example.Options {
		opts = append(opts, Option{ID: o.ID, Label: o.Label, ImageMediaID: mediaIDForURL(r, o.ImageURL)})
	}
	r.ScienceExample = &example
	r.Question = &QuestionView{
		Interaction: converted.ResponseKind, TargetText: r.Title,
		Stem:    Stem{Text: example.Prompt, ImageMediaID: mediaIDForURL(r, example.ImageURL)},
		Options: opts, AnswerOptionID: example.AnswerID,
	}
	r.QuestionFidelity = "instance_snapshot"
	r.MediaFidelity = "immutable"
	if !sciencecontent.HasFrozenMedia(example) {
		r.MediaFidelity = "mutable_reference"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_not_frozen")
	}
	r.ReviewBlockReasons = []string{"unsupported_subject"}
	if converted.Selected == "" {
		return
	}
	r.SelectionFidelity = "stable_option"
	if converted.ResponseKind != sciencecontent.KindChoice {
		r.SelectionFidelity = "stable_mapping"
	}
	r.Response = ResponseView{Kind: converted.ResponseKind, SelectedOptionID: converted.Selected, Value: converted.Selected}
	correct := sciencecontent.IsCorrect(example, converted.Input)
	if r.IsCorrect != correct {
		r.Assistance = "unknown"
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
	} else {
		r.Assistance = "none"
	}
}

func attachScienceMedia(r *Evidence, example sciencecontent.ScienceExample) {
	add := func(id, url string) {
		if url != "" {
			r.Media[id] = "science:" + url
		}
	}
	add("stem-image", example.ImageURL)
	for i, n := range example.MatchSources {
		add(fmt.Sprintf("match-source-%d", i), n.ImageURL)
	}
	for i, n := range example.MatchTargets {
		add(fmt.Sprintf("match-target-%d", i), n.ImageURL)
	}
	for i, n := range example.SequenceItems {
		add(fmt.Sprintf("sequence-%d", i), n.ImageURL)
	}
	for i, o := range example.Options {
		add(fmt.Sprintf("option-%d-image", i), o.ImageURL)
	}
}

func mediaIDForURL(r *Evidence, url string) string {
	if url == "" {
		return ""
	}
	want := "science:" + url
	for id, raw := range r.Media {
		if raw == want {
			return id
		}
	}
	return ""
}
