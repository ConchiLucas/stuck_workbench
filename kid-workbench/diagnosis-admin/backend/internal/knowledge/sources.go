package knowledge

import (
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/chengyucontent"
	"github.com/conchi/study-learning/englishcontent"
	"github.com/conchi/study-learning/literacycontract"
	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/phrasecontent"
	"github.com/conchi/study-learning/poemcontent"
	"github.com/conchi/study-learning/sciencecontent"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type versionRow struct {
	AttemptID, ID, PlanID, PlanItemID, QuestionVersionID                                                                                  int64
	SelectedOptionID, QuestionType, SkillCode, SnapshotJSON, AnswerPayloadJSON, EvaluationJSON, EvaluatorVersion, PlanStatus, OptionOrder string
}
type mathRow struct {
	AttemptID, PlanID, ItemID  int64
	SkillCode, Snapshot, Picks string
}
type pinyinRow struct {
	AttemptID                                                                         int64
	InstanceID, ClientID, SelectedOptionID, PublicSnapshot, AnswerOptionID, SkillCode string
}
type legacyQuestion struct {
	ID                          int64
	Stem, Options, Answer, Code string
}
type mediaRef struct{ RevisionID, Kind, SHA256 string }
type versionSnapshot struct {
	SchemaVersion                                                                         int
	SubjectCode, QuestionType, SkillCode, TargetText, Prompt, Interaction, AnswerOptionID string
	KpID                                                                                  int64
	Stem                                                                                  struct {
		Text         string
		Image, Audio *mediaRef
	}
	Options []struct {
		ID, Text     string
		KpID         int64
		Image, Audio *mediaRef
	}
}

func (s *Service) enrich(rows []Evidence) error {
	if len(rows) == 0 {
		return nil
	}
	ids := []int64{}
	qids := []int64{}
	for _, r := range rows {
		ids = append(ids, r.AttemptID)
		if r.QuestionID != nil {
			qids = append(qids, *r.QuestionID)
		}
	}
	versions := map[int64]versionRow{}
	if s.HasVersions && s.DB.Migrator().HasTable("question_versions") {
		v := []versionRow{}
		extra := ""
		if s.HasWriting {
			extra = ",qr.answer_payload_json,qr.evaluation_json,qr.evaluator_version"
		}
		order := "'' AS option_order"
		if s.DB.Migrator().HasColumn("plan_items", "option_order") {
			order = "pi.option_order"
		}
		e := s.DB.Table("question_attempt_receipts qr").Select("qr.attempt_id,qr.id,qr.plan_id,qr.plan_item_id,qr.question_version_id,qr.selected_option_id,qr.question_type,qr.skill_code,v.snapshot_json,sp.status AS plan_status,"+order+extra).Joins("JOIN question_versions v ON v.id=qr.question_version_id AND v.kp_id=qr.kp_id").Joins("JOIN plan_items pi ON pi.id=qr.plan_item_id AND pi.plan_id=qr.plan_id AND pi.kp_id=qr.kp_id AND pi.question_version_id=qr.question_version_id").Joins("JOIN study_plans sp ON sp.id=qr.plan_id AND sp.child_id=qr.child_id").Where("qr.attempt_id IN ? AND qr.child_id=?", ids, rows[0].ChildID).Scan(&v).Error
		if e != nil {
			return e
		}
		for _, r := range v {
			versions[r.AttemptID] = r
		}
	}
	pyn := map[int64]pinyinRow{}
	if s.HasPinyin {
		v := []pinyinRow{}
		e := s.DB.Table("pinyin_answer_receipts pr").Select("pr.attempt_id,pr.instance_id,pr.client_id,pr.selected_option_id,pr.skill_code,qi.public_snapshot,qi.answer_option_id").Joins("JOIN pinyin_quiz_instances qi ON qi.id=pr.instance_id AND qi.child_id=pr.child_id").Where("pr.attempt_id IN ? AND pr.child_id=?", ids, rows[0].ChildID).Scan(&v).Error
		if e != nil {
			return e
		}
		for _, r := range v {
			pyn[r.AttemptID] = r
		}
	}
	maths := map[int64]mathRow{}
	english := map[int64]mathRow{}
	if s.DB.Migrator().HasTable("plan_items") && s.DB.Migrator().HasTable("study_plans") && s.DB.Migrator().HasColumn("plan_items", "question_snapshot") {
		v := []mathRow{}
		e := s.DB.Raw(`SELECT a.id AS attempt_id, COALESCE(q.code,'') AS skill_code, COALESCE(CAST(pi.question_snapshot AS TEXT),'') AS snapshot, COALESCE(pi.picks,'') AS picks, COALESCE(pi.plan_id,0) AS plan_id, COALESCE(pi.id,0) AS item_id
 FROM attempts a LEFT JOIN questions q ON q.id=a.question_id
 LEFT JOIN plan_items pi ON pi.id=(
  SELECT pi2.id FROM plan_items pi2 JOIN study_plans sp2 ON sp2.id=pi2.plan_id
  WHERE pi2.question_id=a.question_id AND pi2.kp_id=a.kp_id AND sp2.child_id=a.child_id AND sp2.subject_code='math'
  ORDER BY pi2.id DESC LIMIT 1)
 WHERE a.id IN ? AND a.child_id=?`, ids, rows[0].ChildID).Scan(&v).Error
		if e != nil {
			return e
		}
		for _, r := range v {
			if r.Snapshot != "" || r.SkillCode != "" {
				maths[r.AttemptID] = r
			}
		}
		v = nil
		if s.DB.Migrator().HasColumn("attempts", "plan_item_id") {
			selectedExpr := "COALESCE(pi.picks,'')"
			if s.DB.Migrator().HasColumn("attempts", "selected") {
				selectedExpr = "COALESCE(NULLIF(a.selected,''), CASE WHEN COALESCE(pi.picks,'') LIKE '[%' THEN '' ELSE COALESCE(pi.picks,'') END, '')"
			}
			e = s.DB.Raw(`SELECT a.id AS attempt_id, COALESCE(q.code,'') AS skill_code, COALESCE(CAST(pi.question_snapshot AS TEXT),'') AS snapshot, `+selectedExpr+` AS picks, COALESCE(pi.plan_id,0) AS plan_id, COALESCE(pi.id,0) AS item_id
 FROM attempts a LEFT JOIN questions q ON q.id=a.question_id
 LEFT JOIN plan_items pi ON pi.id=a.plan_item_id
 WHERE a.id IN ? AND a.child_id=?`, ids, rows[0].ChildID).Scan(&v).Error
			if e != nil {
				return e
			}
			for _, r := range v {
				if r.Snapshot != "" || r.SkillCode != "" {
					english[r.AttemptID] = r
				}
			}
		}
	}
	questions := map[int64]legacyQuestion{}
	if len(qids) > 0 {
		v := []legacyQuestion{}
		if e := s.DB.Table("questions").Where("id IN ?", qids).Scan(&v).Error; e != nil {
			return e
		}
		for _, r := range v {
			questions[r.ID] = r
		}
	}
	for i := range rows {
		r := &rows[i]
		r.Source = SourceRef{Kind: "legacy_attempt"}
		r.QuestionType = r.SkillCode
		r.SkillLabel = Label(r.SubjectCode, r.SkillCode)
		r.Assistance = "unknown"
		r.QuestionFidelity = "missing"
		r.SelectionFidelity = "missing"
		r.MediaFidelity = "missing"
		r.Response = ResponseView{Kind: "unknown"}
		r.ReviewBlockReasons = []string{"evidence_incomplete"}
		r.EvidenceReasonCodes = []string{}
		r.Media = map[string]string{}
		r.FollowUpState = "no_later_practice"
		if v, ok := versions[r.AttemptID]; ok {
			decodeVersion(r, v)
		} else if p, ok := pyn[r.AttemptID]; ok {
			decodePinyin(r, p)
		} else if m, ok := maths[r.AttemptID]; ok && r.SubjectCode == "math" {
			decodeMath(r, m)
		} else if r.SubjectCode == "english" {
			decodeEnglish(r, english[r.AttemptID])
			if r.EnglishExample != nil && r.MediaFidelity == "immutable" {
				if err := englishcontent.VerifyMedia(s.DB, *r.EnglishExample); err != nil {
					r.MediaFidelity = "missing"
					r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_missing_or_corrupt")
					r.Media = map[string]string{}
				}
			}
			if r.MediaFidelity == "mutable_reference" {
				r.Media = map[string]string{}
			}
		} else if r.SubjectCode == "phrase" {
			decodePhrase(r, english[r.AttemptID])
			if r.PhraseExample != nil && r.MediaFidelity == "immutable" {
				if err := phrasecontent.VerifyMedia(s.DB, *r.PhraseExample); err != nil {
					r.MediaFidelity = "missing"
					r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_missing_or_corrupt")
					r.Media = map[string]string{}
				}
			}
			if r.MediaFidelity == "mutable_reference" {
				r.Media = map[string]string{}
			}
		} else if r.SubjectCode == "chengyu" {
			decodeChengyu(r, english[r.AttemptID])
			if r.ChengyuExample != nil && r.MediaFidelity == "immutable" {
				if err := chengyucontent.VerifyMedia(s.DB, *r.ChengyuExample); err != nil {
					r.MediaFidelity = "missing"
					r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_missing_or_corrupt")
					r.Media = map[string]string{}
				}
			}
			if r.MediaFidelity == "mutable_reference" {
				r.Media = map[string]string{}
			}
		} else if r.SubjectCode == "science" {
			if s.DB.Migrator().HasColumn("attempts", "plan_item_id") {
				row := english[r.AttemptID]
				if strings.TrimSpace(row.Snapshot) != "" || row.ItemID == 0 {
					decodeScience(r, row)
					if r.ScienceExample != nil && r.MediaFidelity == "immutable" {
						if err := sciencecontent.VerifyMedia(s.DB, *r.ScienceExample); err != nil {
							r.MediaFidelity = "missing"
							r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_missing_or_corrupt")
							r.Media = map[string]string{}
						}
					}
					if r.MediaFidelity == "mutable_reference" {
						r.Media = map[string]string{}
					}
				}
			}
		} else if r.SubjectCode == "poem" {
			if s.DB.Migrator().HasColumn("attempts", "plan_item_id") {
				row := english[r.AttemptID]
				if strings.TrimSpace(row.Snapshot) != "" || row.ItemID == 0 {
					decodePoem(r, row)
					if r.PoemExample != nil && r.MediaFidelity == "immutable" {
						if err := poemcontent.VerifyMedia(s.DB, *r.PoemExample); err != nil {
							r.MediaFidelity = "missing"
							r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_missing_or_corrupt")
							r.Media = map[string]string{}
						}
					}
					if r.MediaFidelity == "mutable_reference" {
						r.Media = map[string]string{}
					}
				}
			}
		} else if r.SubjectCode == "logic" {
			if s.DB.Migrator().HasColumn("attempts", "plan_item_id") {
				row := english[r.AttemptID]
				if strings.TrimSpace(row.Snapshot) != "" || row.ItemID == 0 {
					decodeLogic(r, row)
					if r.LogicExample != nil && r.MediaFidelity == "immutable" {
						if err := logiccontent.VerifyMedia(s.DB, *r.LogicExample); err != nil {
							r.MediaFidelity = "missing"
							r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "media_missing_or_corrupt")
							r.Media = map[string]string{}
						}
					}
					if r.MediaFidelity == "mutable_reference" {
						r.Media = map[string]string{}
					}
				}
			}
		} else if r.QuestionID != nil {
			if q, ok := questions[*r.QuestionID]; ok {
				decodeLegacy(r, q)
			}
		}
	}
	if s.HasScience {
		if e := s.science(rows); e != nil {
			return e
		}
	}
	return nil
}
func media(r *Evidence, id string, ref *mediaRef) string {
	if ref == nil || ref.RevisionID == "" || ref.Kind == "" {
		return ""
	}
	r.Media[id] = "content:/api/v1/material-revisions/" + url.PathEscape(ref.RevisionID) + "/media/" + url.PathEscape(ref.Kind)
	return id
}
func decodeVersion(r *Evidence, v versionRow) {
	r.Source = SourceRef{Kind: "literacy_version", ReceiptID: v.ID, QuestionVersionID: v.QuestionVersionID}
	r.PlanID = v.PlanID
	r.PlanItemID = v.PlanItemID
	r.InstanceKey = fmt.Sprintf("item:%d", v.PlanItemID)
	r.QuestionType = v.QuestionType
	r.SkillCode = v.SkillCode
	r.SkillLabel = Label(r.SubjectCode, v.SkillCode)
	var snap versionSnapshot
	if json.Unmarshal([]byte(v.SnapshotJSON), &snap) != nil || snap.SubjectCode != "literacy" || r.SubjectCode != "literacy" || !literacycontract.Supports(v.QuestionType) || snap.SkillCode != v.SkillCode || v.SkillCode != v.QuestionType || snap.KpID != r.KpID || snap.QuestionType != v.QuestionType || (snap.SchemaVersion != 1 && snap.SchemaVersion != 2) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	q := &QuestionView{Interaction: snap.Interaction, TargetText: snap.TargetText, Stem: Stem{Text: snap.Prompt}, Options: []Option{}, AnswerOptionID: snap.AnswerOptionID}
	if q.Interaction == "" {
		q.Interaction = "choice"
	}
	if snap.Stem.Text != "" {
		q.Stem.Text = snap.Stem.Text
	}
	q.Stem.ImageMediaID = media(r, "stem-image", snap.Stem.Image)
	q.Stem.AudioMediaID = media(r, "stem-audio", snap.Stem.Audio)
	for i, o := range snap.Options {
		q.Options = append(q.Options, Option{ID: o.ID, Label: o.Text, SemanticID: fmt.Sprintf("kp:%d", o.KpID), ImageMediaID: media(r, fmt.Sprintf("option-%d-image", i), o.Image), AudioMediaID: media(r, fmt.Sprintf("option-%d-audio", i), o.Audio)})
		if o.ID == v.SelectedOptionID && o.KpID > 0 {
			r.SelectedSemanticID = fmt.Sprintf("kp:%d", o.KpID)
		}
	}
	r.Question = q
	r.QuestionFidelity = "frozen_version"
	r.MediaFidelity = "immutable"
	r.Assistance = "unknown"
	if v.QuestionType == "write_char" {
		var payload struct {
			Strokes   json.RawMessage
			HintsUsed *int
		}
		var eval struct{ Assistance, Outcome string }
		if json.Unmarshal([]byte(v.AnswerPayloadJSON), &payload) != nil || payload.HintsUsed == nil || len(payload.Strokes) == 0 {
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "writing_payload_missing")
			r.Assistance = "unknown"
			return
		}
		var evaluation json.RawMessage
		if json.Unmarshal([]byte(v.EvaluationJSON), &eval) == nil {
			evaluation = json.RawMessage(v.EvaluationJSON)
			if eval.Assistance == "none" {
				r.Assistance = "none"
			}
			if eval.Assistance == "hinted" {
				r.Assistance = "hinted"
			}
		}
		r.Response = ResponseView{Kind: "handwriting", Strokes: payload.Strokes, HintsUsed: payload.HintsUsed, Evaluation: evaluation, EvaluatorVersion: v.EvaluatorVersion}
		if *payload.HintsUsed > 0 {
			r.Assistance = "hinted"
		}
		if *payload.HintsUsed < 0 {
			r.Assistance = "unknown"
		}
		if (eval.Outcome == "passed" || eval.Outcome == "not_passed") && r.IsCorrect != (eval.Outcome == "passed") {
			r.Assistance = "unknown"
			r.ReviewEligible = false
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
			r.ReviewBlockReasons = append(r.ReviewBlockReasons, "result_conflict")
			return
		}
	} else {
		selected, answer := false, false
		seen := map[string]bool{}
		for _, o := range q.Options {
			if o.ID == "" || seen[o.ID] {
				r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_options")
				return
			}
			seen[o.ID] = true
			selected = selected || o.ID == v.SelectedOptionID
			answer = answer || o.ID == q.AnswerOptionID
		}
		if !selected || !answer {
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "selection_missing")
			return
		}
		r.Assistance = "none"
		r.Response = ResponseView{Kind: "choice", SelectedOptionID: v.SelectedOptionID}
		r.SelectionFidelity = "stable_option"
		if v.OptionOrder != "" {
			if ordered, ok := orderedOptions(q.Options, v.OptionOrder); ok {
				q.Options = ordered
			} else {
				r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "display_order_missing")
			}
		}
		if r.IsCorrect != (v.SelectedOptionID == q.AnswerOptionID) {
			r.Assistance = "unknown"
			r.SelectedSemanticID = ""
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
			r.ReviewBlockReasons = []string{"result_conflict"}
			r.ReviewEligible = false
			return
		}
	}
	r.ReviewBlockReasons = []string{}
	if v.PlanStatus != "done" {
		r.ReviewBlockReasons = append(r.ReviewBlockReasons, "source_plan_incomplete")
	}
	r.ReviewEligible = len(r.ReviewBlockReasons) == 0
}
func orderedOptions(opts []Option, raw string) ([]Option, bool) {
	parts := strings.Split(raw, ",")
	if len(parts) != len(opts) {
		return opts, false
	}
	out := []Option{}
	seen := map[int]bool{}
	for _, part := range parts {
		i, e := strconv.Atoi(strings.TrimSpace(part))
		if e != nil || i < 0 || i >= len(opts) || seen[i] {
			return opts, false
		}
		seen[i] = true
		out = append(out, opts[i])
	}
	return out, true
}
func decodePinyin(r *Evidence, p pinyinRow) {
	r.Source = SourceRef{Kind: "pinyin_instance", InstanceID: p.InstanceID, ClientID: p.ClientID}
	r.InstanceKey = "pinyin:" + p.InstanceID
	r.SkillCode = p.SkillCode
	r.SkillLabel = Label(r.SubjectCode, p.SkillCode)
	r.QuestionType = p.SkillCode
	var snap struct {
		KpID            int64
		Stem, SpeechURL string
		Visual          json.RawMessage
		Options         []struct{ ID, Label, SpeechText, SpeechURL string }
	}
	if json.Unmarshal([]byte(p.PublicSnapshot), &snap) != nil || snap.KpID != r.KpID || r.SubjectCode != "pinyin" || !hasString(mastery.SkillsFor(r.SubjectCode, r.ModuleCode), p.SkillCode) {
		r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_snapshot")
		return
	}
	q := &QuestionView{Interaction: "choice", TargetText: r.Title, Stem: Stem{Text: snap.Stem}, AnswerOptionID: p.AnswerOptionID, Options: []Option{}}
	if snap.SpeechURL != "" {
		r.Media["stem-audio"] = "pinyin:" + snap.SpeechURL
		q.Stem.AudioMediaID = "stem-audio"
	}
	var visual struct{ Kind, Text, ImageURL, Initial, Final, Syllable string }
	_ = json.Unmarshal(snap.Visual, &visual)
	q.Visual = snap.Visual
	if visual.ImageURL != "" {
		r.Media["stem-image"] = "pinyin:" + visual.ImageURL
		q.Stem.ImageMediaID = "stem-image"
	}

	selected, answer := false, false
	seen := map[string]bool{}
	for i, o := range snap.Options {
		if o.ID == "" || seen[o.ID] {
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "invalid_options")
			return
		}
		seen[o.ID] = true
		v := Option{ID: o.ID, Label: o.Label}
		if v.Label == "" {
			v.Label = o.SpeechText
		}
		if o.SpeechURL != "" {
			id := fmt.Sprintf("option-%d-audio", i)
			r.Media[id] = "pinyin:" + o.SpeechURL
			v.AudioMediaID = id
		}
		q.Options = append(q.Options, v)
		selected = selected || o.ID == p.SelectedOptionID
		answer = answer || o.ID == p.AnswerOptionID
	}
	r.Question = q
	r.QuestionFidelity = "instance_snapshot"
	r.MediaFidelity = "mutable_reference"
	if selected && answer {
		r.SelectionFidelity = "stable_option"
		r.Response = ResponseView{Kind: "choice", SelectedOptionID: p.SelectedOptionID}
		if r.IsCorrect == (p.SelectedOptionID == p.AnswerOptionID) {
			r.Assistance = "none"
		} else {
			r.EvidenceReasonCodes = append(r.EvidenceReasonCodes, "result_conflict")
		}
	}
	r.ReviewBlockReasons = []string{"unsupported_subject"}
}
func decodeLegacy(r *Evidence, q legacyQuestion) {
	view := &QuestionView{Interaction: "reference", TargetText: r.Title, Stem: Stem{Text: q.Stem}, Options: []Option{}}
	var opts []struct{ Label, Text, Emoji string }
	if json.Unmarshal([]byte(q.Options), &opts) == nil {
		for i, o := range opts {
			label := o.Label
			if label == "" {
				label = o.Text
			}
			if label == "" {
				label = o.Emoji
			}
			view.Options = append(view.Options, Option{ID: fmt.Sprintf("reference-%d", i), Label: label})
		}
	}
	r.Question = view
	r.QuestionFidelity = "current_reference"
	r.EvidenceReasonCodes = []string{"historical_selection_missing"}
}
func (s *Service) science(rows []Evidence) error {
	keys := []string{}
	for _, r := range rows {
		if r.SubjectCode == "science" && r.Source.Kind == "legacy_attempt" {
			keys = append(keys, r.ClientID)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	type item struct {
		ClientID                                                                      string
		KpID, QuestionID, PlanID, ItemID                                              int64
		ResultJSON, QuestionStem, QuestionOptions, QuestionAnswer, Picks, OptionOrder string
	}
	items := []item{}
	order := "'' AS option_order"
	if s.DB.Migrator().HasColumn("plan_items", "option_order") {
		order = "pi.option_order"
	}
	e := s.DB.Table("science_attempt_receipts sr").Select("sr.client_id,sr.kp_id,sr.question_id,sr.plan_id,sr.item_id,sr.result_json,pi.question_stem,pi.question_options,pi.question_answer,pi.picks,"+order).Joins("JOIN plan_items pi ON pi.id=sr.item_id AND pi.plan_id=sr.plan_id AND pi.kp_id=sr.kp_id").Joins("JOIN study_plans sp ON sp.id=sr.plan_id AND sp.child_id=sr.child_id").Where("sr.child_id=? AND sr.client_id IN ?", rows[0].ChildID, keys).Scan(&items).Error
	if e != nil {
		return e
	}
	by := map[string]item{}
	for _, v := range items {
		by[v.ClientID] = v
	}
	for i := range rows {
		r := &rows[i]
		v, ok := by[r.ClientID]
		if !ok || r.KpID != v.KpID || r.QuestionID == nil || *r.QuestionID != v.QuestionID {
			continue
		}
		r.Source = SourceRef{Kind: "science_plan", ClientID: r.ClientID, PlanID: v.PlanID, ItemID: v.ItemID}
		r.PlanID = v.PlanID
		r.PlanItemID = v.ItemID
		r.InstanceKey = fmt.Sprintf("item:%d", v.ItemID)
		r.QuestionFidelity = "plan_snapshot_unverified"
		r.ReviewBlockReasons = []string{"unsupported_subject"}
		decodeLegacy(r, legacyQuestion{Stem: v.QuestionStem, Options: v.QuestionOptions})
		r.QuestionFidelity = "plan_snapshot_unverified"
		var result struct{ Tries int }
		var answer struct{ Index *int }
		picks := strings.Split(v.Picks, ",")
		ordered, validOrder := orderedOptions(r.Question.Options, v.OptionOrder)
		if json.Unmarshal([]byte(v.ResultJSON), &result) == nil && json.Unmarshal([]byte(v.QuestionAnswer), &answer) == nil && validOrder && result.Tries > 0 && result.Tries <= len(picks) && answer.Index != nil && *answer.Index >= 0 && *answer.Index < len(r.Question.Options) {
			pick, err := strconv.Atoi(strings.TrimSpace(picks[result.Tries-1]))
			if err == nil && pick >= 0 && pick < len(ordered) {
				r.Question.AnswerOptionID = r.Question.Options[*answer.Index].ID
				r.Question.Options = ordered
				r.Response = ResponseView{Kind: "choice", SelectedOptionID: ordered[pick].ID}
				r.SelectionFidelity = "verified_order"
				r.EvidenceReasonCodes = []string{"historical_snapshot_unverified"}
			}
		}
	}
	return nil
}
func (s *Service) followUps(rows []Evidence, max int64, asOf time.Time) error {
	if len(rows) == 0 {
		return nil
	}
	q, skill, instance := s.instanceBase(rows[0].ChildID, Filter{}, max, asOf)
	keys := map[string]bool{}
	ids := []int64{}
	clauses := []string{}
	args := []any{}
	for _, r := range rows {
		key := fmt.Sprintf("%d:%s", r.KpID, r.SkillCode)
		if keys[key] {
			continue
		}
		keys[key] = true
		ids = append(ids, r.KpID)
		clauses = append(clauses, "(a.kp_id=? AND "+skill+"=?)")
		args = append(args, r.KpID, r.SkillCode)
	}
	q = q.Where("("+strings.Join(clauses, " OR ")+")", args...)
	// First establish lifetime instance initials, then select only the latest
	// initial for each page key. Retries never become later independent practice.
	initials := q.Select(evidenceSelect(skill) + ",ROW_NUMBER() OVER(PARTITION BY a.kp_id," + skill + ",COALESCE(" + instance + ",'unknown:' || CAST(a.id AS TEXT)) ORDER BY a.created_at,a.id) AS initial_rank")
	ranked := s.DB.Table("(?) AS initials", initials).Where("initial_rank=1").Select("initials.*,ROW_NUMBER() OVER(PARTITION BY kp_id,skill_code ORDER BY occurred_at DESC,attempt_id DESC) AS latest_rank")
	latest := []Evidence{}
	if e := s.DB.Table("(?) AS latest", ranked).Where("latest_rank=1").Limit(evidenceBatchSize).Scan(&latest).Error; e != nil {
		return e
	}
	if e := s.enrich(latest); e != nil {
		return e
	}
	// Resolve any later independently verified first attempt, separately from the
	// latest follow-up display. Unknown or hinted success cannot close an error.
	correctQuery, correctSkill := s.boundedBase(rows[0].ChildID, Filter{}, max, asOf)
	correctQuery = correctQuery.Where("a.id IN (?)", s.DB.Table("(?) AS initial_correct", initials).Select("attempt_id").Where("initial_rank=1 AND is_correct=?", true))
	if e := s.streamEvidence(correctQuery, correctSkill, func(batch []Evidence) (bool, error) {
		for _, a := range batch {
			if a.Assistance != "none" || a.InstanceKey == "" {
				continue
			}
			for i := range rows {
				r := &rows[i]
				if a.KpID == r.KpID && a.SkillCode == r.SkillCode && a.InstanceKey != r.InstanceKey && (a.OccurredAt.After(r.OccurredAt) || (a.OccurredAt.Equal(r.OccurredAt) && a.AttemptID > r.AttemptID)) {
					r.LaterIndependentCorrect = true
				}
			}
		}
		return true, nil
	}); e != nil {
		return e
	}
	type state struct {
		KpID              int64
		SkillCode, Status string
		DueAt             *time.Time
	}
	states := []state{}
	if e := s.DB.Table("mastery_skills").Where("child_id=? AND kp_id IN ?", rows[0].ChildID, ids).Scan(&states).Error; e != nil {
		return e
	}
	stateMap := map[string]string{}
	for _, st := range states {
		stateMap[fmt.Sprintf("%d:%s", st.KpID, st.SkillCode)] = effective(st.Status, st.DueAt, asOf)
	}
	for i := range rows {
		r := &rows[i]
		r.FollowUpState = "no_later_practice"
		r.LaterAttempts = 0
		for _, a := range latest {
			if a.KpID != r.KpID || a.SkillCode != r.SkillCode || a.InstanceKey != "" && a.InstanceKey == r.InstanceKey {
				continue
			}
			if a.OccurredAt.Before(r.OccurredAt) || a.OccurredAt.Equal(r.OccurredAt) && a.AttemptID <= r.AttemptID {
				continue
			}
			r.LaterAttempts = 1
			switch {
			case a.InstanceKey == "" || a.Assistance == "unknown":
				r.FollowUpState = "insufficient_evidence"
			case !a.IsCorrect:
				r.FollowUpState = "still_wrong"
			case a.Assistance != "none":
				r.FollowUpState = "insufficient_evidence"
			case mastered(stateMap[fmt.Sprintf("%d:%s", r.KpID, r.SkillCode)]):
				r.FollowUpState = "mastered_later"
			default:
				r.FollowUpState = "answered_correctly_later"
			}
		}
	}
	return nil
}
