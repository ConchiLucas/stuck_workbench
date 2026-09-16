package reviewsuggestion

import (
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-task-admin/internal/generation"
	"gorm.io/gorm"
	"sort"
	"strings"
	"time"
)

type verified struct {
	Assistance        string    `json:"assistance"`
	AttemptID         int64     `json:"attemptId"`
	KpID              int64     `json:"kpId"`
	QuestionType      string    `json:"questionType"`
	PlanID            int64     `json:"planId,omitempty"`
	ItemID            int64     `json:"itemId,omitempty"`
	Correct           bool      `json:"correct"`
	Assisted          bool      `json:"assisted"`
	CreatedAt         time.Time `json:"createdAt"`
	SelectedKpID      int64     `json:"selectedKpId,omitempty"`
	Semantic          string    `json:"selectedSemantic,omitempty"`
	Instance          string    `json:"instance"`
	SnapshotJSON      string    `json:"-"`
	QuestionVersionID int64     `json:"questionVersionId,omitempty"`
	ReceiptID         int64     `json:"receiptId,omitempty"`
}

func normalize(in Input) (Input, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.SubjectCode = strings.TrimSpace(in.SubjectCode)
	if in.SchemaVersion != 1 || in.AnalysisVersion != "knowledge-analysis-v1" || in.AnalysisAsOf.IsZero() || in.AnalysisAsOf.After(time.Now().Add(5*time.Minute)) {
		return in, bad("invalid_analysis", "invalid analysis version or time")
	}
	if in.Title == "" || len([]rune(in.Title)) > 80 || len(in.Targets) < 1 || len(in.Targets) > 20 {
		return in, bad("invalid_request", "title or targets out of range")
	}
	total, evidenceCount := 0, 0
	keys := map[string]bool{}
	for i := range in.Targets {
		t := &in.Targets[i]
		t.QuestionType = strings.TrimSpace(t.QuestionType)
		if t.Mode == "" {
			t.Mode = "mixed"
		}
		if t.PreferredDistractorKpIDs == nil {
			t.PreferredDistractorKpIDs = []int64{}
		}
		if t.Key != fmt.Sprintf("%d:%s", t.KpID, t.QuestionType) || keys[t.Key] || t.KpID < 1 || t.QuestionType == "" {
			return in, bad("invalid_target", "invalid or duplicate target key")
		}
		keys[t.Key] = true
		if t.Mode != "mixed" && t.Mode != "original_only" || t.RequestedCount < 1 || t.RequestedCount > 10 || len(t.Evidence) == 0 {
			return in, bad("invalid_target", "invalid target mode, count or evidence")
		}
		if t.ReasonCode != "observed_wrong" && t.ReasonCode != "repeated_skill_error" && t.ReasonCode != "repeated_confusion" && t.ReasonCode != "assisted_completion" && t.ReasonCode != "practiced_skill_gap" {
			return in, bad("invalid_reason", "unsupported reason")
		}
		if t.QuestionType == "write_char" && len(t.PreferredDistractorKpIDs) > 0 {
			return in, bad("invalid_preference", "handwriting has no distractors")
		}
		total += t.RequestedCount
		seen := map[int64]bool{}
		for _, ev := range t.Evidence {
			if ev.AttemptID < 1 || seen[ev.AttemptID] {
				return in, bad("invalid_evidence", "duplicate attempt in target")
			}
			seen[ev.AttemptID] = true
			evidenceCount++
			switch ev.Role {
			case "target_error", "target_observation", "assisted_completion", "supporting_strength":
			default:
				return in, bad("invalid_role", "invalid evidence role")
			}
			src := ev.Source
			switch src.Kind {
			case "literacy_version":
				if src.ReceiptID < 1 || src.QuestionVersionID < 1 || src.ClientID != "" || src.InstanceID != "" || src.PlanID != 0 || src.ItemID != 0 {
					return in, bad("invalid_source", "invalid literacy source")
				}
			case "pinyin_instance":
				if src.ClientID == "" || src.InstanceID == "" || src.ReceiptID != 0 || src.QuestionVersionID != 0 || src.PlanID != 0 || src.ItemID != 0 {
					return in, bad("invalid_source", "invalid pinyin source")
				}
			case "science_plan":
				if src.ClientID == "" || src.PlanID < 1 || src.ItemID < 1 || src.ReceiptID != 0 || src.QuestionVersionID != 0 || src.InstanceID != "" {
					return in, bad("invalid_source", "invalid science source")
				}
			case "legacy_attempt":
				if src.ClientID != "" || src.InstanceID != "" || src.PlanID != 0 || src.ItemID != 0 || src.ReceiptID != 0 || src.QuestionVersionID != 0 {
					return in, bad("invalid_source", "legacy source must not guess plan links")
				}
			default:
				return in, bad("invalid_source", "unsupported source discriminator")
			}
		}
		sort.Slice(t.Evidence, func(i, j int) bool { return t.Evidence[i].AttemptID < t.Evidence[j].AttemptID })
		sort.Slice(t.PreferredDistractorKpIDs, func(i, j int) bool { return t.PreferredDistractorKpIDs[i] < t.PreferredDistractorKpIDs[j] })
	}
	if total > 20 || evidenceCount > 200 {
		return in, bad("request_limit", "too many questions or evidence")
	}
	sort.Slice(in.Targets, func(i, j int) bool { return in.Targets[i].Key < in.Targets[j].Key })
	in.AnalysisAsOf = in.AnalysisAsOf.UTC()
	return in, nil
}
func verifyEvidence(tx *gorm.DB, child int64, t TargetInput, ev EvidenceInput, asof time.Time) (verified, error) {
	v := verified{AttemptID: ev.AttemptID, Assistance: "unknown"}
	var a struct {
		ID, ChildID, KpID, QuestionID int64
		IsCorrect                     bool
		Source, ClientID              string
		CreatedAt                     time.Time
	}
	if e := tx.Table("attempts").Where("id=? AND child_id=? AND kp_id=? AND source='quiz'", ev.AttemptID, child, t.KpID).Take(&a).Error; e != nil {
		return v, bad("evidence_mismatch", "attempt ownership or source mismatch")
	}
	if a.CreatedAt.After(asof) {
		return v, bad("evidence_after_analysis", "attempt occurs after analysis")
	}
	v.KpID = a.KpID
	v.Correct = a.IsCorrect
	v.CreatedAt = a.CreatedAt
	v.Instance = fmt.Sprintf("legacy:%d", a.ID)
	src := ev.Source
	switch src.Kind {
	case "literacy_version":
		var r struct {
			ID, AttemptID, ChildID, PlanID, PlanItemID, QuestionVersionID, KpID int64
			QuestionType, SelectedOptionID, EvaluationJSON, AnswerPayloadJSON   string
			IsCorrect                                                           bool
		}
		if e := tx.Table("question_attempt_receipts").Where("id=? AND attempt_id=? AND child_id=? AND kp_id=? AND question_version_id=?", src.ReceiptID, a.ID, child, t.KpID, src.QuestionVersionID).Take(&r).Error; e != nil {
			return v, bad("evidence_mismatch", "receipt does not match attempt")
		}
		var link struct {
			SnapshotJSON string
			KpID         int64
			QuestionType string
		}
		e := tx.Raw(`SELECT v.snapshot_json,v.kp_id,v.question_type FROM question_versions v JOIN question_task_revisions rev ON rev.id=v.revision_id JOIN plan_items i ON i.question_version_id=v.id JOIN study_plans p ON p.id=i.plan_id WHERE v.id=? AND i.id=? AND i.plan_id=? AND i.kp_id=? AND p.child_id=? AND p.subject_code='literacy' AND p.source_question_task_revision_id=v.revision_id AND p.source_question_task_id=rev.task_id`, src.QuestionVersionID, r.PlanItemID, r.PlanID, t.KpID, child).Scan(&link).Error
		if e != nil {
			return v, e
		}
		if link.KpID != t.KpID || link.QuestionType != r.QuestionType || r.IsCorrect != a.IsCorrect {
			return v, bad("evidence_mismatch", "receipt version or plan ownership mismatch")
		}
		v.QuestionType = r.QuestionType
		v.PlanID = r.PlanID
		v.ItemID = r.PlanItemID
		v.Instance = fmt.Sprintf("literacy:%d", r.PlanItemID)
		v.ReceiptID = r.ID
		v.QuestionVersionID = r.QuestionVersionID
		v.SnapshotJSON = link.SnapshotJSON
		var eval map[string]any
		_ = json.Unmarshal([]byte(r.EvaluationJSON), &eval)
		v.Assistance = "none"
		if r.QuestionType == "write_char" {
			if outcome, ok := eval["outcome"].(string); ok && (outcome == "passed" || outcome == "not_passed") && a.IsCorrect != (outcome == "passed") {
				return v, bad("evidence_mismatch", "writing evaluation conflicts with attempt result")
			}
			v.Assistance = "unknown"
			if eval["assistance"] == "hinted" {
				v.Assistance = "hinted"
			} else if eval["assistance"] == "none" {
				v.Assistance = "none"
			}
			var payload struct {
				HintsUsed *int `json:"hintsUsed"`
			}
			if json.Unmarshal([]byte(r.AnswerPayloadJSON), &payload) == nil && payload.HintsUsed != nil {
				if *payload.HintsUsed > 0 {
					v.Assistance = "hinted"
				} else if *payload.HintsUsed < 0 {
					v.Assistance = "unknown"
				}
			}
		}
		v.Assisted = v.Assistance == "hinted"
		var q generation.Snapshot
		if e = json.Unmarshal([]byte(link.SnapshotJSON), &q); e != nil {
			return v, bad("invalid_snapshot", "invalid source snapshot")
		}
		if q.KpID != t.KpID || q.SubjectCode != "literacy" || q.QuestionType != r.QuestionType {
			return v, bad("evidence_mismatch", "frozen snapshot target mismatch")
		}
		if err := generation.ValidateSnapshot(q); err != nil {
			return v, bad("invalid_snapshot", err.Error())
		}
		selectionFound := false
		for _, o := range q.Options {
			if o.ID == r.SelectedOptionID {
				selectionFound = true
				v.SelectedKpID = o.KpID
				v.Semantic = fmt.Sprintf("kp:%d", o.KpID)
				if o.KpID == 0 {
					b, _ := json.Marshal(o)
					v.Semantic = string(b)
				}
			}
		}
		if q.Interaction != "handwriting" && (!selectionFound || a.IsCorrect != (q.AnswerOptionID == r.SelectedOptionID)) {
			return v, bad("evidence_mismatch", "receipt selection conflicts with frozen snapshot")
		}
	case "pinyin_instance":
		var r struct {
			AttemptID, KpID                                                            int64
			SkillCode, InstanceSkill, PublicSnapshot, SelectedOptionID, AnswerOptionID string
		}
		e := tx.Raw(`SELECT r.attempt_id,i.kp_id,r.skill_code,i.public_snapshot,r.selected_option_id,i.answer_option_id,i.skill_code AS instance_skill FROM pinyin_answer_receipts r JOIN pinyin_quiz_instances i ON i.id=r.instance_id AND i.child_id=r.child_id WHERE r.child_id=? AND r.client_id=? AND r.instance_id=? AND r.attempt_id=?`, child, src.ClientID, src.InstanceID, a.ID).Scan(&r).Error
		if e != nil {
			return v, e
		}
		if r.AttemptID != a.ID || r.KpID != t.KpID || a.ClientID != src.ClientID {
			return v, bad("evidence_mismatch", "pinyin instance mismatch")
		}
		v.QuestionType = r.SkillCode
		v.Instance = "pinyin:" + src.InstanceID
		var snapshot struct {
			KpID    int64
			Options []struct{ ID, Label, SpeechText string }
		}
		valid := json.Unmarshal([]byte(r.PublicSnapshot), &snapshot) == nil && snapshot.KpID == t.KpID && r.InstanceSkill == r.SkillCode
		seen := map[string]bool{}
		selected, answer := false, false
		semantic := ""
		for _, o := range snapshot.Options {
			if o.ID == "" || seen[o.ID] {
				valid = false
			}
			seen[o.ID] = true
			answer = answer || o.ID == r.AnswerOptionID
			if o.ID == r.SelectedOptionID {
				selected = true
				semantic = strings.TrimSpace(o.Label)
				if semantic == "" {
					semantic = strings.TrimSpace(o.SpeechText)
				}
			}
		}
		// Keep the observed attempt even when its rendering cannot establish
		// an independent answer; unknown evidence cannot support a strong rule.
		if valid && selected && answer && a.IsCorrect == (r.SelectedOptionID == r.AnswerOptionID) {
			v.Assistance = "none"
			if semantic != "" {
				v.Semantic = "pinyin:" + semantic
			}
		}

	case "science_plan":
		var r struct {
			KpID, QuestionID int64
			Type             string
		}
		e := tx.Raw(`SELECT r.kp_id,r.question_id,q.code AS type FROM science_attempt_receipts r JOIN study_plans p ON p.id=r.plan_id AND p.child_id=r.child_id JOIN plan_items i ON i.id=r.item_id AND i.plan_id=p.id JOIN questions q ON q.id=r.question_id WHERE r.child_id=? AND r.client_id=? AND r.plan_id=? AND r.item_id=? AND p.subject_code='science' AND i.kp_id=r.kp_id AND i.question_id=r.question_id`, child, src.ClientID, src.PlanID, src.ItemID).Scan(&r).Error
		if e != nil {
			return v, e
		}
		if r.KpID != t.KpID || r.QuestionID != a.QuestionID || a.ClientID != src.ClientID {
			return v, bad("evidence_mismatch", "science receipt mismatch")
		}
		v.QuestionType = r.Type
		v.PlanID = src.PlanID
		v.ItemID = src.ItemID
		v.Instance = fmt.Sprintf("science:%d", src.ItemID)
	case "legacy_attempt":
		// Refuse stripping a known canonical source to bypass its verification.
		for _, table := range []string{"question_attempt_receipts", "pinyin_answer_receipts"} {
			if tx.Migrator().HasTable(table) {
				var n int64
				if e := tx.Table(table).Where("attempt_id=?", a.ID).Count(&n).Error; e != nil {
					return v, e
				}
				if n > 0 {
					return v, bad("source_required", "canonical receipt source required")
				}
			}
		}
		var q struct{ Type string }
		if e := tx.Table("questions").Select("code AS type").Where("id=? AND kp_id=?", a.QuestionID, t.KpID).Take(&q).Error; e != nil {
			return v, bad("evidence_mismatch", "legacy question missing")
		}
		v.QuestionType = q.Type
	}
	if ev.Role == "supporting_strength" {
		if !v.Correct || v.Assistance != "none" || v.QuestionType == t.QuestionType {
			return v, bad("invalid_role", "supporting strength must be correct in a different skill")
		}
	} else {
		if v.QuestionType != t.QuestionType {
			return v, bad("evidence_mismatch", "target question type mismatch")
		}
		if ev.Role == "target_error" && v.Correct {
			return v, bad("invalid_role", "target error requires a wrong answer")
		}
		if ev.Role == "assisted_completion" && (!v.Correct || !v.Assisted) {
			return v, bad("invalid_role", "assisted completion requires assisted success")
		}
		if ev.Role == "target_observation" && v.Assisted {
			return v, bad("invalid_role", "assistance is separate from unassisted observations")
		}
	}
	return v, nil
}
func validateTarget(tx *gorm.DB, child int64, in Input, t TargetInput) (Target, []Evidence, error) {
	out := Target{TargetKey: t.Key, KpID: t.KpID, QuestionType: t.QuestionType, ReasonCode: t.ReasonCode, Mode: t.Mode, RequestedCount: t.RequestedCount}
	var kp struct{ ModuleCode, SubjectCode string }
	if e := tx.Raw(`SELECT m.code AS module_code,s.code AS subject_code FROM knowledge_points k JOIN modules m ON m.id=k.module_id JOIN subjects s ON s.id=m.subject_id WHERE k.id=?`, t.KpID).Scan(&kp).Error; e != nil {
		return out, nil, e
	}
	if kp.SubjectCode != in.SubjectCode || kp.ModuleCode == "" {
		return out, nil, bad("target_subject_mismatch", "target does not belong to subject")
	}
	out.ModuleCode = kp.ModuleCode
	allowed := false
	for _, skill := range mastery.SkillsFor(in.SubjectCode, kp.ModuleCode) {
		if skill == t.QuestionType {
			allowed = true
		}
	}
	if !allowed {
		return out, nil, bad("invalid_question_type", "question type does not apply to target module")
	}
	evidences := []Evidence{}
	wrong := 0
	assisted := 0
	instances := map[string]verified{}
	confusions := map[string]map[string]bool{}
	preferred := map[int64]bool{}
	strengths := map[string]bool{}
	for _, ev := range t.Evidence {
		expectedSubject := map[string]string{"literacy_version": "literacy", "pinyin_instance": "pinyin", "science_plan": "science"}[ev.Source.Kind]
		if expectedSubject != "" && expectedSubject != in.SubjectCode {
			return out, nil, bad("source_subject_mismatch", "source discriminator does not belong to subject")
		}
		v, e := verifyEvidence(tx, child, t, ev, in.AnalysisAsOf)
		if e != nil {
			return out, nil, e
		}
		b, _ := json.Marshal(ev.Source)
		summary, _ := json.Marshal(v)
		row := Evidence{AttemptID: ev.AttemptID, Role: ev.Role, SourceKind: ev.Source.Kind, SourceJSON: string(b), EvidenceSummaryJSON: string(summary)}
		if ev.Source.Kind == "literacy_version" {
			receipt, version := ev.Source.ReceiptID, ev.Source.QuestionVersionID
			row.SourceReceiptID = &receipt
			row.SourceQuestionVersionID = &version
		}
		if v.PlanID > 0 {
			plan, item := v.PlanID, v.ItemID
			row.SourcePlanID = &plan
			row.SourcePlanItemID = &item
		}
		if ev.Source.Kind == "pinyin_instance" {
			instance := ev.Source.InstanceID
			row.SourcePinyinInstanceID = &instance
		}
		if ev.Source.Kind == "science_plan" {
			c, client := child, ev.Source.ClientID
			row.SourceChildID = &c
			row.SourceClientID = &client
		}
		evidences = append(evidences, row)
		if ev.Role == "supporting_strength" {
			strengths[v.QuestionType] = true
		}
		if ev.Role == "target_error" {
			wrong++
			if v.SelectedKpID > 0 {
				preferred[v.SelectedKpID] = true
			}
			if v.Semantic != "" {
				if confusions[v.Semantic] == nil {
					confusions[v.Semantic] = map[string]bool{}
				}
				confusions[v.Semantic][v.Instance] = true
			}
		}
		if ev.Role == "assisted_completion" {
			assisted++
		}
		if ev.Role != "supporting_strength" {
			if old, ok := instances[v.Instance]; !ok || v.CreatedAt.Before(old.CreatedAt) || v.CreatedAt.Equal(old.CreatedAt) && v.AttemptID < old.AttemptID {
				instances[v.Instance] = v
			}
		}
	}
	confusions = map[string]map[string]bool{}
	for _, v := range instances {
		if !v.Correct && v.Assistance == "none" && v.Semantic != "" && !strings.HasPrefix(v.Instance, "legacy:") {
			if confusions[v.Semantic] == nil {
				confusions[v.Semantic] = map[string]bool{}
			}
			confusions[v.Semantic][v.Instance] = true
		}
	}
	for _, id := range t.PreferredDistractorKpIDs {
		if !preferred[id] {
			return out, nil, bad("invalid_preference", "preferred distractor is not a verified selected content")
		}
	}
	switch t.ReasonCode {
	case "observed_wrong":
		if wrong < 1 {
			return out, nil, bad("insufficient_evidence", "observed_wrong requires wrong evidence")
		}
		out.ReasonText = "记录中出现过作答错误，建议再次练习。"
	case "assisted_completion":
		if assisted < 1 {
			return out, nil, bad("insufficient_evidence", "assisted completion requires assistance evidence")
		}
		out.ReasonText = "曾在提示帮助下完成，建议检查独立完成情况。"
	case "repeated_confusion":
		window, err := firstWindow(tx, child, in, t)
		if err != nil {
			return out, nil, err
		}
		submitted := map[int64]verified{}
		for _, v := range instances {
			submitted[v.AttemptID] = v
		}
		confusions = map[string]map[string]bool{}
		for _, id := range window {
			v, ok := submitted[id]
			if !ok {
				return out, nil, bad("incomplete_evidence", "confusion requires canonical first-answer observations")
			}
			if !v.Correct && v.Assistance == "none" && v.Semantic != "" {
				if confusions[v.Semantic] == nil {
					confusions[v.Semantic] = map[string]bool{}
				}
				confusions[v.Semantic][v.Instance] = true
			}
		}
		enough := false
		for _, is := range confusions {
			if len(is) >= 2 {
				enough = true
			}
		}
		if !enough {
			return out, nil, bad("insufficient_evidence", "repeated confusion requires the same semantic selection in independent instances")
		}
		out.ReasonText = "在不同题目中重复选中了同一内容，建议对照辨析。"
	case "practiced_skill_gap":
		if !tx.Migrator().HasTable("mastery_skills") {
			return out, nil, bad("insufficient_evidence", "mastery skill state is unavailable")
		}
		mastered := []string{}
		for skill := range strengths {
			var count int64
			if e := tx.Table("mastery_skills").Where("child_id=? AND kp_id=? AND skill_code=? AND status='mastered'", child, t.KpID, skill).Count(&count).Error; e != nil {
				return out, nil, e
			}
			if count > 0 {
				mastered = append(mastered, skill)
			}
		}
		if len(mastered) == 0 {
			return out, nil, bad("insufficient_evidence", "skill gap requires a mastered skill with independent correct evidence")
		}
		var shaky int64
		if e := tx.Table("mastery_skills").Where("child_id=? AND kp_id=? AND skill_code=? AND status='shaky'", child, t.KpID, t.QuestionType).Count(&shaky).Error; e != nil {
			return out, nil, e
		}
		if len(instances) == 0 {
			return out, nil, bad("insufficient_evidence", "weak skill requires actual practice")
		}
		if shaky == 0 {
			copy := t
			copy.ReasonCode = "repeated_skill_error"
			if _, _, e := validateTarget(tx, child, in, copy); e != nil {
				return out, nil, e
			}
		}
		sort.Strings(mastered)
		out.ReasonText = fmt.Sprintf("%s已有独立正确证据且当前已掌握；%s练过且需要补练。", strings.Join(mastered, "、"), t.QuestionType)
	case "repeated_skill_error":
		window, err := firstWindow(tx, child, in, t)
		if err != nil {
			return out, nil, err
		}
		submitted := map[int64]verified{}
		for _, v := range instances {
			submitted[v.AttemptID] = v
		}
		independent, errors := 0, 0
		for _, id := range window {
			v, ok := submitted[id]
			if !ok {
				return out, nil, bad("incomplete_evidence", "window requires every first-answer observation")
			}
			if v.Assistance == "none" {
				independent++
				if !v.Correct {
					errors++
				}
			}
		}
		if independent < 3 || errors < 2 {
			return out, nil, bad("insufficient_evidence", "repeated skill error requires three instances and two first errors")
		}
		out.ReasonText = fmt.Sprintf("最近30天最后%d个独立首答中有%d次错误，建议按此题型复习。", independent, errors)
	}
	return out, evidences, nil
}

// firstWindow reconstructs canonical instances, never guesses a legacy plan link.
func firstWindow(tx *gorm.DB, child int64, in Input, t TargetInput) ([]int64, error) {
	table, join, instance, skill := "", "", "", ""
	switch in.SubjectCode {
	case "literacy":
		table = "question_attempt_receipts"
		join = "JOIN question_attempt_receipts r ON r.attempt_id=a.id AND r.child_id=a.child_id"
		instance = "CAST(r.plan_item_id AS TEXT)"
		skill = "r.question_type"
	case "pinyin":
		table = "pinyin_answer_receipts"
		join = "JOIN pinyin_answer_receipts r ON r.attempt_id=a.id AND r.child_id=a.child_id"
		instance = "r.instance_id"
		skill = "r.skill_code"
	case "science":
		table = "science_attempt_receipts"
		join = "JOIN science_attempt_receipts r ON r.child_id=a.child_id AND r.client_id=a.client_id AND r.question_id=a.question_id JOIN questions q ON q.id=r.question_id"
		instance = "CAST(r.item_id AS TEXT)"
		skill = "q.code"
	default:
		return nil, bad("insufficient_evidence", "cannot reconstruct independent first answers")
	}
	if !tx.Migrator().HasTable(table) {
		return nil, bad("insufficient_evidence", "cannot reconstruct independent first answers")
	}
	var rows []struct {
		AttemptID int64
		Instance  string
		CreatedAt time.Time
	}
	e := tx.Raw("SELECT attempt_id,instance,created_at FROM (SELECT a.id AS attempt_id,"+instance+" AS instance,a.created_at,ROW_NUMBER() OVER(PARTITION BY "+instance+" ORDER BY a.created_at,a.id) AS rn FROM attempts a "+join+" WHERE a.child_id=? AND a.kp_id=? AND a.source='quiz' AND "+skill+"=? AND a.created_at<=?) canonical WHERE rn=1 AND created_at>=? ORDER BY created_at DESC,attempt_id DESC LIMIT 20", child, t.KpID, t.QuestionType, in.AnalysisAsOf, in.AnalysisAsOf.AddDate(0, 0, -30)).Scan(&rows).Error
	if e != nil {
		return nil, e
	}
	out := []int64{}
	for _, r := range rows {
		out = append(out, r.AttemptID)
	}
	return out, nil
}
