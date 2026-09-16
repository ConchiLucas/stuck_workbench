package knowledge

import (
	"fmt"
	"strconv"

	"github.com/conchi/study-learning/mathcontent"
	"github.com/conchi/study-learning/mastery"
)

func decodeMath(r *Evidence, row mathRow) {
	r.Source = SourceRef{Kind: "math_plan", PlanID: row.PlanID, ItemID: row.ItemID}
	r.PlanID = row.PlanID
	r.PlanItemID = row.ItemID
	r.InstanceKey = fmt.Sprintf("math:%d", r.AttemptID)
	if row.SkillCode != "" {
		r.SkillCode = row.SkillCode
		r.QuestionType = row.SkillCode
		r.SkillLabel = Label(r.SubjectCode, row.SkillCode)
	}
	allowed := mastery.SkillsFor("math", r.ModuleCode)
	if row.Snapshot == "" || r.SubjectCode != "math" || (len(allowed) > 0 && !hasString(allowed, r.SkillCode)) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	converted, err := mathcontent.PlanExampleFromSnapshot(row.Snapshot, mathcontent.LastPickIndex(row.Picks))
	if err != nil {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	example := converted.Example
	if example.Kind == "audio-shape" {
		if converted.AudioObjectKey == "" || row.PlanID == 0 || row.ItemID == 0 {
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
			return
		}
		path := fmt.Sprintf("/api/v1/children/%d/math/plans/%d/items/%d/audio.mp3", r.ChildID, row.PlanID, row.ItemID)
		r.Media["stem-audio"] = "math:" + path
		example.AudioURL = path
		r.AudioMutable = converted.AudioMutable
		r.MediaFidelity = "mutable_reference"
	} else {
		example.AudioURL = ""
		r.MediaFidelity = "immutable"
	}
	r.MathExample = &example
	opts := make([]Option, len(example.Options))
	for i, label := range example.Options {
		id := "o" + strconv.Itoa(i+1)
		if len(example.OptionIDs) == len(example.Options) {
			id = example.OptionIDs[i]
		}
		opts[i] = Option{ID: id, Label: label}
	}
	selectedID := ""
	if converted.Selected != "" {
		for _, o := range opts {
			if o.Label == converted.Selected {
				selectedID = o.ID
				break
			}
		}
	}
	audioID := ""
	if example.Kind == "audio-shape" {
		audioID = "stem-audio"
	}
	r.Question = &QuestionView{Interaction: "choice", TargetText: r.Title, Stem: Stem{Text: example.Prompt, AudioMediaID: audioID}, Options: opts, AnswerOptionID: example.AnswerOptionID}
	r.QuestionFidelity = "instance_snapshot"
	if selectedID != "" {
		r.SelectionFidelity = "stable_option"
		r.Response = ResponseView{Kind: "choice", SelectedOptionID: selectedID}
		if r.IsCorrect != (selectedID == example.AnswerOptionID) {
			r.Assistance = "unknown"
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
		} else {
			r.Assistance = "none"
		}
	}
	r.ReviewBlockReasons = []string{"unsupported_subject"}
}
