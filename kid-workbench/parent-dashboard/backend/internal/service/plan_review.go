package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ReviewQuestion 是家长端的题目视图，含正确答案。
// 和 QuestionDTO 分开定义而不是加开关：孩子端接口不该有任何路径能返回答案。
type ReviewQuestion struct {
	ID          int64           `json:"id"`
	VersionID   int64           `json:"version_id,omitempty"`
	Type        string          `json:"type"`
	Stem        string          `json:"stem"`
	Options     json.RawMessage `json:"options"`
	Visual      json.RawMessage `json:"visual"`
	AnswerIndex int             `json:"answer_index"`
}

type WritingReceipt struct {
	ResponseKind      string          `json:"response_kind"`
	AnswerPayloadJSON json.RawMessage `json:"answer_payload"`
	EvaluationJSON    json.RawMessage `json:"evaluation"`
	CreatedAt         time.Time       `json:"created_at"`
}
type PlanReviewItem struct {
	LiteracyReview  *LiteracyReview  `json:"literacy_review,omitempty"`
	WritingReceipts []WritingReceipt `json:"writing_receipts,omitempty"`
	Seq             int              `json:"seq"`
	Status          string           `json:"status"` // pending|correct|wrong
	Bucket          string           `json:"bucket"` // review|shaky|learning|new
	Tries           int              `json:"tries"`
	CostMs          int              `json:"cost_ms"`
	Picks           []int            `json:"picks"` // 按作答顺序；旧数据为空
	AnsweredAt      *time.Time       `json:"answered_at"`
	KpID            int64            `json:"kp_id"`
	KpTitle         string           `json:"kp_title"`
	SubjectCode     string           `json:"subject_code"`
	SubjectName     string           `json:"subject_name"`
	Question        ReviewQuestion   `json:"question"`
}

type PlanReview struct {
	Plan  PlanDTO          `json:"plan"`
	Items []PlanReviewItem `json:"items"`
}

// Review 给家长端复盘：含正确答案和孩子选过的选项。
func (s *PlanService) Review(childID, planID int64) (PlanReview, error) {
	p, err := s.loadPlan(s.repo.DB(), childID, planID)
	if err != nil {
		return PlanReview{}, err
	}

	out := PlanReview{Plan: toPlanDTO(p), Items: []PlanReviewItem{}}

	type row struct {
		Seq               int
		Status            string
		Bucket            string
		Tries             int
		CostMs            int
		Picks             string
		AnsweredAt        *time.Time
		KpID              int64
		KpTitle           string
		SubjectCode       string
		SubjectName       string
		QuestionID        int64
		QuestionVersionID int64
		OptionOrder       string
		QuestionSnapshot  string
		QuestionCode      string
		Type              string
		Stem              string
		Options           string
		Visual            string
		Answer            string
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT pi.seq, pi.status, pi.bucket, pi.tries, pi.cost_ms, pi.picks, pi.answered_at,
		       pi.kp_id, kp.title AS kp_title,
		       s.code AS subject_code, s.name AS subject_name,
		       pi.question_id, pi.question_version_id, pi.option_order,pi.question_snapshot, q.type,q.code AS question_code,
 COALESCE(NULLIF(pi.question_stem,''),q.stem) AS stem,
 COALESCE(NULLIF(pi.question_options,''),q.options) AS options,
 COALESCE(NULLIF(pi.question_visual,''),q.visual) AS visual,
 COALESCE(NULLIF(pi.question_answer,''),q.answer) AS answer
		FROM plan_items pi
		LEFT JOIN questions q    ON q.id = pi.question_id
		JOIN knowledge_points kp ON kp.id = pi.kp_id
		JOIN modules m           ON m.id = kp.module_id
		JOIN subjects s          ON s.id = m.subject_id
		WHERE pi.plan_id = ?
		ORDER BY pi.seq`, p.ID).Scan(&rows).Error; err != nil {
		return out, err
	}

	for _, r := range rows {
		var literacyReview *LiteracyReview
		if snapshotGlyph(r.QuestionSnapshot) || (r.SubjectCode == "literacy" && (r.QuestionCode == "glyph_sense" || (r.QuestionVersionID > 0 && snapshotTypeMissing(r.QuestionSnapshot)))) {
			literacyReview = buildLiteracyReview(r.QuestionVersionID, r.QuestionSnapshot, r.OptionOrder, "")
			if literacyReview.Question != nil {
				picks := parsePicks(r.Picks)
				validPicks := true
				if r.Picks != "" {
					for _, p := range strings.Split(r.Picks, ",") {
						n, e := strconv.Atoi(strings.TrimSpace(p))
						if e != nil || n < 0 || n >= len(literacyReview.Question.Options) {
							validPicks = false
							break
						}
					}
				}
				if !validPicks {
					literacyReview = unavailableReview("历史作答选项位置无效")
				} else if len(picks) > 0 {
					last := picks[len(picks)-1]
					if last < 0 || last >= len(literacyReview.Question.Options) {
						literacyReview = unavailableReview("历史作答选项位置无效")
					} else {
						literacyReview.SelectedOptionID = literacyReview.Question.Options[last].ID
					}
				}
			}
		}
		if literacyReview != nil && literacyReview.UnavailableReason != "" {
			// Preserve the older response fields only when they can be built from the
			// version snapshot. The shared player still receives an unavailable review.
			legacy := ReviewQuestion{ID: r.QuestionID, VersionID: r.QuestionVersionID, Type: "choice", Options: json.RawMessage("[]"), Visual: json.RawMessage("{}"), AnswerIndex: -1}
			if r.QuestionVersionID > 0 {
				if frozen, e := frozenHistory(r.QuestionSnapshot); e == nil {
					if options, e := orderedHistoryOptions(frozen.Options, r.OptionOrder); e == nil {
						if answer, e := parseAnswerIndex(frozen.Answer); e == nil {
							if r.OptionOrder != "" {
								for i, j := range parsePicks(r.OptionOrder) {
									if j == answer {
										answer = i
										break
									}
								}
							}
							legacy.Stem = frozen.Stem
							legacy.Options = json.RawMessage(options)
							legacy.Visual = json.RawMessage(frozen.Visual)
							legacy.AnswerIndex = answer
						}
					}
				}
			}
			out.Items = append(out.Items, PlanReviewItem{LiteracyReview: literacyReview, Seq: r.Seq, Status: r.Status, Bucket: r.Bucket, Tries: r.Tries, CostMs: r.CostMs, Picks: parsePicks(r.Picks), AnsweredAt: r.AnsweredAt, KpID: r.KpID, KpTitle: r.KpTitle, SubjectCode: r.SubjectCode, SubjectName: r.SubjectName, Question: legacy})
			continue
		}
		if r.QuestionVersionID != 0 {
			frozen, err := frozenHistory(r.QuestionSnapshot)
			if err != nil {
				return out, err
			}
			r.KpTitle = frozen.Title
			r.Type = "choice"
			if frozen.Code == "write_char" {
				r.Type = "handwriting"
			}
			r.Stem = frozen.Stem
			r.Options = frozen.Options
			r.Visual = frozen.Visual
			r.Answer = frozen.Answer
		}
		answerIndex, err := parseAnswerIndex(r.Answer)
		if r.Type == "handwriting" {
			answerIndex = -1
			err = nil
		}
		if err != nil {
			return out, err
		}
		if r.OptionOrder != "" {
			var original []json.RawMessage
			if err := json.Unmarshal([]byte(r.Options), &original); err != nil {
				return out, err
			}
			order := parsePicks(r.OptionOrder)
			if len(order) != len(original) {
				return out, fmt.Errorf("invalid option order")
			}
			reordered := make([]json.RawMessage, len(order))
			seen := map[int]bool{}
			displayAnswer := -1
			for i, j := range order {
				if j < 0 || j >= len(original) || seen[j] {
					return out, fmt.Errorf("invalid option order")
				}
				seen[j] = true
				reordered[i] = original[j]
				if j == answerIndex {
					displayAnswer = i
				}
			}
			answerIndex = displayAnswer
			b, _ := json.Marshal(reordered)
			r.Options = string(b)
		}
		var writing []WritingReceipt
		var writingRows []struct {
			ResponseKind, AnswerPayloadJSON, EvaluationJSON string
			CreatedAt                                       time.Time
		}
		if r.Type == "handwriting" {
			if err := s.repo.DB().Table("question_attempt_receipts").Where("plan_id = ? AND question_version_id = ? AND response_kind = ?", p.ID, r.QuestionVersionID, "handwriting").Order("created_at,id").Find(&writingRows).Error; err != nil {
				return out, err
			}
			for _, r := range writingRows {
				writing = append(writing, WritingReceipt{ResponseKind: r.ResponseKind, AnswerPayloadJSON: json.RawMessage(r.AnswerPayloadJSON), EvaluationJSON: json.RawMessage(r.EvaluationJSON), CreatedAt: r.CreatedAt})
			}
		}
		out.Items = append(out.Items, PlanReviewItem{LiteracyReview: literacyReview, WritingReceipts: writing,
			Seq: r.Seq, Status: r.Status, Bucket: r.Bucket, Tries: r.Tries,
			CostMs: r.CostMs, Picks: parsePicks(r.Picks), AnsweredAt: r.AnsweredAt,
			KpID: r.KpID, KpTitle: r.KpTitle,
			SubjectCode: r.SubjectCode, SubjectName: r.SubjectName,
			Question: ReviewQuestion{
				VersionID: r.QuestionVersionID, ID: r.QuestionID, Type: r.Type, Stem: r.Stem,
				Options: rawJSON(r.Options, "[]"), Visual: rawJSON(r.Visual, "{}"),
				AnswerIndex: answerIndex,
			},
		})
	}
	return out, nil
}

func parsePicks(s string) []int {
	if s == "" {
		return []int{}
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}
