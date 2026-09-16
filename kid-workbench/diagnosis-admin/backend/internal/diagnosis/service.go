package diagnosis

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/conchi/study-learning/mastery"
	"gorm.io/gorm"
)

const effectiveStatusSQL = `
	CASE WHEN ms.status = 'mastered' AND ms.due_at IS NOT NULL AND ms.due_at < CURRENT_TIMESTAMP
	     THEN 'review_due' ELSE COALESCE(ms.status, 'not_started') END`

type Service struct{ db *gorm.DB }

func NewService(gdb *gorm.DB) *Service { return &Service{db: gdb} }

type HeadlineFacts struct {
	Shaky             int
	ReviewDue         int
	LiteracyWriteWeak bool
	EnglishListenWeak bool
}

func BuildHeadline(f HeadlineFacts) string {
	var parts []string
	if f.LiteracyWriteWeak {
		parts = append(parts, "识字能认、手写偏弱")
	}
	if f.EnglishListenWeak {
		parts = append(parts, "英语听音不稳")
	}
	if f.Shaky > 0 {
		parts = append(parts, fmt.Sprintf("有 %d 个知识点需巩固", f.Shaky))
	}
	if f.ReviewDue > 0 {
		parts = append(parts, fmt.Sprintf("有 %d 个到期该复习", f.ReviewDue))
	}
	if len(parts) == 0 {
		return "目前没有突出薄弱点，可以按计划推进。"
	}
	return strings.Join(parts, "；") + "。"
}

type ChildDTO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Grade string `json:"grade"`
}

type StatusCounts struct {
	NotStarted int `json:"not_started"`
	Learning   int `json:"learning"`
	Shaky      int `json:"shaky"`
	Mastered   int `json:"mastered"`
	ReviewDue  int `json:"review_due"`
}

type SubjectHealth struct {
	Code    string       `json:"code"`
	Name    string       `json:"name"`
	Icon    string       `json:"icon"`
	Total   int          `json:"total"`
	Counts  StatusCounts `json:"counts"`
	Health  string       `json:"health"`
	Summary string       `json:"summary"`
}

type UrgentItem struct {
	KpID        int64   `json:"kp_id"`
	Title       string  `json:"title"`
	SubjectCode string  `json:"subject_code"`
	SubjectName string  `json:"subject_name"`
	ModuleName  string  `json:"module_name"`
	Status      string  `json:"status"`
	Accuracy    float64 `json:"accuracy"`
	WrongCount  int     `json:"wrong_count"`
	Reason      string  `json:"reason"`
}

type Overview struct {
	Child    ChildDTO        `json:"child"`
	Headline string          `json:"headline"`
	Subjects []SubjectHealth `json:"subjects"`
	Urgent   []UrgentItem    `json:"urgent"`
}

type SkillGap struct {
	KpID        int64  `json:"kp_id"`
	Title       string `json:"title"`
	StrongSkill string `json:"strong_skill"`
	StrongLabel string `json:"strong_label"`
	WeakSkill   string `json:"weak_skill"`
	WeakLabel   string `json:"weak_label"`
}

type ModuleHealth struct {
	Code   string       `json:"code"`
	Name   string       `json:"name"`
	Total  int          `json:"total"`
	Counts StatusCounts `json:"counts"`
}

type SubjectDiagnosis struct {
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Icon      string         `json:"icon"`
	Health    string         `json:"health"`
	Headline  string         `json:"headline"`
	Counts    StatusCounts   `json:"counts"`
	Total     int            `json:"total"`
	Modules   []ModuleHealth `json:"modules"`
	SkillGaps []SkillGap     `json:"skill_gaps"`
}

type HistoryItem struct {
	At         time.Time `json:"at"`
	IsCorrect  bool      `json:"is_correct"`
	CostMs     int       `json:"cost_ms"`
	Source     string    `json:"source"`
	SkillCode  string    `json:"skill_code,omitempty"`
	SkillLabel string    `json:"skill_label,omitempty"`
}

type SkillState struct {
	Code     string  `json:"code"`
	Label    string  `json:"label"`
	Status   string  `json:"status"`
	Accuracy float64 `json:"accuracy"`
	Attempts int     `json:"attempts"`
}

type RecentWrong struct {
	ResponseKind      string          `json:"response_kind,omitempty"`
	AnswerPayloadJSON json.RawMessage `json:"answer_payload,omitempty"`
	EvaluationJSON    json.RawMessage `json:"evaluation,omitempty"`
	SelectedOptionID  string          `json:"selected_option_id,omitempty"`
	QuestionVersionID int64           `json:"question_version_id,omitempty"`
	Picks             string          `json:"picks"`
	Answer            string          `json:"answer"`
	AnsweredAt        *time.Time      `json:"answered_at"`
}

type KpArchive struct {
	KpID        int64         `json:"kp_id"`
	Title       string        `json:"title"`
	SubjectCode string        `json:"subject_code"`
	SubjectName string        `json:"subject_name"`
	ModuleName  string        `json:"module_name"`
	Status      string        `json:"status"`
	Attempts    int           `json:"attempts"`
	Accuracy    float64       `json:"accuracy"`
	Skills      []SkillState  `json:"skills" gorm:"-"`
	History     []HistoryItem `json:"history" gorm:"-"`
	RecentWrong []RecentWrong `json:"recent_wrong" gorm:"-"`
}

type ErrorPattern struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	KpID        int64  `json:"kp_id,omitempty"`
	KpTitle     string `json:"kp_title,omitempty"`
	SubjectCode string `json:"subject_code,omitempty"`
}

func (s *Service) Overview(childID int64) (Overview, error) {
	var out Overview
	var child ChildDTO
	if err := s.db.Table("children").Select("id, name, grade").Where("id = ?", childID).Scan(&child).Error; err != nil {
		return out, err
	}
	if child.ID == 0 {
		return out, gorm.ErrRecordNotFound
	}
	out.Child = child

	subjects, err := s.subjectHealths(childID)
	if err != nil {
		return out, err
	}
	out.Subjects = subjects

	urgent, err := s.urgentItems(childID, 8)
	if err != nil {
		return out, err
	}
	out.Urgent = urgent

	facts := HeadlineFacts{}
	for _, sub := range subjects {
		facts.Shaky += sub.Counts.Shaky
		facts.ReviewDue += sub.Counts.ReviewDue
		if sub.Code == "literacy" {
			gaps, err := s.skillGaps(childID, "literacy")
			if err != nil {
				return out, err
			}
			for _, g := range gaps {
				if g.WeakSkill == mastery.SkillWriteChar && (g.StrongSkill == mastery.SkillGlyphSense || g.StrongSkill == mastery.SkillSenseChar) {
					facts.LiteracyWriteWeak = true
				}
			}
		}
		if sub.Code == "english" {
			weak, err := s.skillStatusCount(childID, "english", mastery.SkillEnglishListen, "shaky")
			if err != nil {
				return out, err
			}
			facts.EnglishListenWeak = weak > 0
		}
	}
	out.Headline = BuildHeadline(facts)
	return out, nil
}

func (s *Service) Subject(childID int64, code string) (SubjectDiagnosis, error) {
	var out SubjectDiagnosis
	healths, err := s.subjectHealths(childID)
	if err != nil {
		return out, err
	}
	var found *SubjectHealth
	for i := range healths {
		if healths[i].Code == code {
			found = &healths[i]
			break
		}
	}
	if found == nil {
		return out, gorm.ErrRecordNotFound
	}
	out.Code = found.Code
	out.Name = found.Name
	out.Icon = found.Icon
	out.Health = found.Health
	out.Counts = found.Counts
	out.Total = found.Total

	gaps, err := s.skillGaps(childID, code)
	if err != nil {
		return out, err
	}
	out.SkillGaps = gaps
	out.Headline = found.Summary
	if len(gaps) > 0 {
		out.Headline = fmt.Sprintf("%s：%s强、%s弱", out.Name, SkillLabel(code, gaps[0].StrongSkill), SkillLabel(code, gaps[0].WeakSkill))
	}

	modules, err := s.moduleHealths(childID, code)
	if err != nil {
		return out, err
	}
	out.Modules = modules
	return out, nil
}

func (s *Service) ErrorPatterns(childID int64) ([]ErrorPattern, error) {
	out := []ErrorPattern{}
	gaps, err := s.allSkillGaps(childID)
	if err != nil {
		return nil, err
	}
	for _, g := range gaps {
		out = append(out, ErrorPattern{
			Kind:        "skill_imbalance",
			Title:       "只会其中一种题型",
			Detail:      fmt.Sprintf("%s：%s已掌握，%s仍弱", g.Title, g.StrongLabel, g.WeakLabel),
			KpID:        g.KpID,
			KpTitle:     g.Title,
			SubjectCode: "",
		})
	}

	type wrongRow struct {
		KpID   int64
		Title  string
		Code   string
		Picks  string
		Answer string
	}
	var wrongs []wrongRow
	if err := s.db.Raw(`
		SELECT pi.kp_id, kp.title, s.code, pi.picks, pi.question_answer AS answer
		FROM plan_items pi
		JOIN study_plans sp ON sp.id = pi.plan_id
		JOIN knowledge_points kp ON kp.id = pi.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE sp.child_id = ? AND s.code <> 'game' AND pi.picks != '' AND pi.status = 'wrong' AND pi.question_id IS NOT NULL
		ORDER BY pi.answered_at DESC
		LIMIT 40`, childID).Scan(&wrongs).Error; err != nil {
		return nil, err
	}
	if s.db.Migrator().HasTable("question_attempt_receipts") {
		var rows []struct {
			KpID         int64
			Title, Code  string
			QuestionType string
			DisplayIndex int
			ResponseJSON string
		}
		if err := s.db.Raw(`SELECT qr.kp_id,kp.title,s.code,qr.question_type,qr.display_index,qr.response_json FROM question_attempt_receipts qr JOIN knowledge_points kp ON kp.id=qr.kp_id JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE qr.child_id=? AND qr.is_correct=? ORDER BY qr.created_at DESC,qr.id DESC LIMIT 40`, childID, false).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			if r.QuestionType == "write_char" {
				out = append(out, ErrorPattern{Kind: "writing_not_passed", Title: "书写尚未通过", Detail: r.Title + " 最近一次书写未通过模板匹配", KpID: r.KpID, KpTitle: r.Title})
				continue
			}
			var result struct {
				AnswerIndex int `json:"answerIndex"`
			}
			if err := json.Unmarshal([]byte(r.ResponseJSON), &result); err != nil {
				return nil, err
			}
			b, _ := json.Marshal(map[string]int{"index": result.AnswerIndex})
			wrongs = append(wrongs, wrongRow{KpID: r.KpID, Title: r.Title, Code: r.Code, Picks: strconv.Itoa(r.DisplayIndex), Answer: string(b)})
		}
	}
	seenWrong := map[string]bool{}
	for _, w := range wrongs {
		pick, okPick := lastPick(w.Picks)
		ans, okAns := answerIndex(w.Answer)
		if !okPick || !okAns || pick == ans {
			continue
		}
		key := fmt.Sprintf("%d:%d:%d", w.KpID, pick, ans)
		if seenWrong[key] {
			continue
		}
		seenWrong[key] = true
		out = append(out, ErrorPattern{
			Kind:        "observed_wrong",
			Title:       "记录过一次错选",
			Detail:      fmt.Sprintf("%s 最近选了第 %d 项，正确答案是第 %d 项", w.Title, pick+1, ans+1),
			KpID:        w.KpID,
			KpTitle:     w.Title,
			SubjectCode: w.Code,
		})
	}

	type slowRow struct {
		KpID  int64
		Title string
		Code  string
		Ms    int
	}
	var slows []slowRow
	if err := s.db.Raw(`
		SELECT a.kp_id, kp.title, s.code, a.cost_ms AS ms
		FROM attempts a
		JOIN knowledge_points kp ON kp.id = a.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE a.child_id = ? AND a.source = 'quiz' AND s.code <> 'game' AND a.is_correct = ? AND a.cost_ms >= 12000
		ORDER BY a.cost_ms DESC
		LIMIT 20`, childID, false).Scan(&slows).Error; err != nil {
		return nil, err
	}
	seenSlow := map[int64]bool{}
	for _, r := range slows {
		if seenSlow[r.KpID] {
			continue
		}
		seenSlow[r.KpID] = true
		out = append(out, ErrorPattern{
			Kind:        "slow_wrong",
			Title:       "用时较长的错误作答",
			Detail:      fmt.Sprintf("%s 用了 %.0f 秒仍答错", r.Title, float64(r.Ms)/1000),
			KpID:        r.KpID,
			KpTitle:     r.Title,
			SubjectCode: r.Code,
		})
	}
	return out, nil
}

func (s *Service) KpArchive(childID, kpID int64) (KpArchive, error) {
	var out KpArchive
	err := s.db.Raw(`
		SELECT kp.id AS kp_id, kp.title, s.code AS subject_code, s.name AS subject_name, m.name AS module_name,
		       `+effectiveStatusSQL+` AS status,
		       COALESCE(ms.attempts,0) AS attempts,
		       CASE WHEN COALESCE(ms.attempts,0) > 0 THEN CAST(ms.correct AS REAL) / ms.attempts ELSE 0 END AS accuracy
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		WHERE kp.id = ?`, childID, kpID).Scan(&out).Error
	if err != nil {
		return out, err
	}
	if out.KpID == 0 {
		return out, gorm.ErrRecordNotFound
	}

	type histRow struct {
		At           time.Time
		IsCorrect    bool
		CostMs       int
		Source       string
		QuestionCode string
	}
	var hist []histRow
	receiptJoin := ""
	questionCode := "q.code"
	hasReceipts := s.db.Migrator().HasTable("question_attempt_receipts")
	if hasReceipts {
		receiptJoin = " LEFT JOIN question_attempt_receipts qr ON qr.attempt_id = a.id "
		questionCode = "COALESCE(qr.skill_code,q.code)"
	}
	if err := s.db.Raw(`
		SELECT a.created_at AS at, a.is_correct, a.cost_ms, a.source, `+questionCode+` AS question_code
		FROM attempts a
		LEFT JOIN questions q ON q.id = a.question_id
 `+receiptJoin+`
		WHERE a.child_id = ? AND a.kp_id = ? AND a.source = 'quiz'
		ORDER BY a.created_at ASC`, childID, kpID).Scan(&hist).Error; err != nil {
		return out, err
	}
	// Attempt rows are facts; mastery counters deliberately exclude assisted answers.
	out.Attempts = len(hist)
	correctFacts := 0
	skillFacts := map[string][2]int{}
	for _, r := range hist {
		if r.IsCorrect {
			correctFacts++
		}
		code := mastery.SkillFromQuestionCode(out.SubjectCode, r.QuestionCode)
		counts := skillFacts[code]
		counts[0]++
		if r.IsCorrect {
			counts[1]++
		}
		skillFacts[code] = counts
	}
	out.Accuracy = 0
	if out.Attempts > 0 {
		out.Accuracy = float64(correctFacts) / float64(out.Attempts)
	}
	for _, r := range hist {
		code := mastery.SkillFromQuestionCode(out.SubjectCode, r.QuestionCode)
		out.History = append(out.History, HistoryItem{
			At: r.At, IsCorrect: r.IsCorrect, CostMs: r.CostMs, Source: r.Source,
			SkillCode: code, SkillLabel: SkillLabel(out.SubjectCode, code),
		})
	}

	var moduleCode string
	s.db.Table("knowledge_points kp").Select("m.code").Joins("JOIN modules m ON m.id=kp.module_id").Where("kp.id=?", kpID).Scan(&moduleCode)
	skillSet := mastery.SkillsFor(out.SubjectCode, moduleCode)
	if len(skillSet) > 0 {
		type skRow struct {
			SkillCode string
			Status    string
			Attempts  int
			Correct   int
		}
		var skills []skRow
		if err := s.db.Raw(`
			SELECT skill_code, status, attempts, correct FROM mastery_skills
			WHERE child_id = ? AND kp_id = ?`, childID, kpID).Scan(&skills).Error; err != nil {
			return out, err
		}
		byCode := map[string]skRow{}
		for _, sk := range skills {
			byCode[sk.SkillCode] = sk
		}
		for _, code := range skillSet {
			st := SkillState{Code: code, Label: SkillLabel(out.SubjectCode, code), Status: "not_started"}
			if row, ok := byCode[code]; ok {
				st.Status = row.Status
				st.Attempts = row.Attempts
				if row.Attempts > 0 {
					st.Accuracy = float64(row.Correct) / float64(row.Attempts)
				}
			}
			if facts, ok := skillFacts[code]; ok {
				st.Attempts = facts[0]
				st.Accuracy = float64(facts[1]) / float64(facts[0])
			}
			out.Skills = append(out.Skills, st)
		}
	}

	var wrongs []RecentWrong
	if err := s.db.Raw(`
		SELECT picks, question_answer AS answer, answered_at
		FROM plan_items pi
		JOIN study_plans sp ON sp.id = pi.plan_id
		WHERE sp.child_id = ? AND pi.kp_id = ? AND pi.status = 'wrong' AND pi.question_id IS NOT NULL
		ORDER BY pi.answered_at DESC
		LIMIT 5`, childID, kpID).Scan(&wrongs).Error; err != nil {
		return out, err
	}
	if hasReceipts {
		var rows []struct {
			DisplayIndex                                                  int
			ResponseKind, AnswerPayloadJSON, EvaluationJSON, QuestionType string
			ResponseJSON                                                  string
			SelectedOptionID                                              string
			QuestionVersionID                                             int64
			CreatedAt                                                     time.Time
		}
		if err := s.db.Table("question_attempt_receipts").Where("child_id = ? AND kp_id = ? AND is_correct = ?", childID, kpID, false).Order("created_at DESC,id DESC").Limit(5).Find(&rows).Error; err != nil {
			return out, err
		}
		for _, r := range rows {
			if r.QuestionType == "write_char" {
				at := r.CreatedAt
				wrongs = append(wrongs, RecentWrong{ResponseKind: "handwriting", AnswerPayloadJSON: json.RawMessage(r.AnswerPayloadJSON), EvaluationJSON: json.RawMessage(r.EvaluationJSON), AnsweredAt: &at, QuestionVersionID: r.QuestionVersionID})
				continue
			}
			var result struct {
				AnswerIndex int `json:"answerIndex"`
			}
			if err := json.Unmarshal([]byte(r.ResponseJSON), &result); err != nil {
				return out, err
			}
			b, _ := json.Marshal(map[string]int{"index": result.AnswerIndex})
			at := r.CreatedAt
			wrongs = append(wrongs, RecentWrong{Picks: strconv.Itoa(r.DisplayIndex), Answer: string(b), AnsweredAt: &at, SelectedOptionID: r.SelectedOptionID, QuestionVersionID: r.QuestionVersionID})
		}
		sort.SliceStable(wrongs, func(i, j int) bool {
			if wrongs[i].AnsweredAt == nil {
				return false
			}
			if wrongs[j].AnsweredAt == nil {
				return true
			}
			return wrongs[i].AnsweredAt.After(*wrongs[j].AnsweredAt)
		})
		if len(wrongs) > 5 {
			wrongs = wrongs[:5]
		}
	}
	out.RecentWrong = wrongs
	return out, nil
}

func (s *Service) subjectHealths(childID int64) ([]SubjectHealth, error) {
	type row struct {
		Code   string
		Name   string
		Icon   string
		Status string
		N      int
		Total  int
	}
	var rows []row
	if err := s.db.Raw(`
		SELECT s.code, s.name, s.icon, `+effectiveStatusSQL+` AS status, COUNT(kp.id) AS n,
		       (SELECT COUNT(1) FROM knowledge_points k2 JOIN modules m2 ON m2.id = k2.module_id WHERE m2.subject_id = s.id) AS total
		FROM subjects s
		JOIN modules m ON m.subject_id = s.id
		JOIN knowledge_points kp ON kp.module_id = m.id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		GROUP BY s.id, s.code, s.name, s.icon, s.order_no, `+effectiveStatusSQL+`
		ORDER BY s.order_no`, childID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byCode := map[string]*SubjectHealth{}
	var order []string
	for _, r := range rows {
		if r.Code == "game" {
			continue
		}
		h, ok := byCode[r.Code]
		if !ok {
			h = &SubjectHealth{Code: r.Code, Name: r.Name, Icon: r.Icon, Total: r.Total}
			byCode[r.Code] = h
			order = append(order, r.Code)
		}
		addCount(&h.Counts, r.Status, r.N)
	}
	out := make([]SubjectHealth, 0, len(order))
	for _, code := range order {
		h := *byCode[code]
		started := h.Counts.Learning + h.Counts.Shaky + h.Counts.Mastered + h.Counts.ReviewDue
		h.Counts.NotStarted = h.Total - started
		if h.Counts.NotStarted < 0 {
			h.Counts.NotStarted = 0
		}
		h.Health = healthOf(h.Counts)
		h.Summary = healthSummary(h.Name, h.Counts)
		out = append(out, h)
	}
	return out, nil
}

func (s *Service) moduleHealths(childID int64, subjectCode string) ([]ModuleHealth, error) {
	type row struct {
		Code   string
		Name   string
		Status string
		N      int
		Total  int
	}
	var rows []row
	if err := s.db.Raw(`
		SELECT m.code, m.name, `+effectiveStatusSQL+` AS status, COUNT(kp.id) AS n,
		       (SELECT COUNT(1) FROM knowledge_points k2 WHERE k2.module_id = m.id) AS total
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		JOIN knowledge_points kp ON kp.module_id = m.id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		WHERE s.code = ?
		GROUP BY m.id, m.code, m.name, m.order_no, `+effectiveStatusSQL+`
		ORDER BY m.order_no`, childID, subjectCode).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byCode := map[string]*ModuleHealth{}
	var order []string
	for _, r := range rows {
		h, ok := byCode[r.Code]
		if !ok {
			h = &ModuleHealth{Code: r.Code, Name: r.Name, Total: r.Total}
			byCode[r.Code] = h
			order = append(order, r.Code)
		}
		addCount(&h.Counts, r.Status, r.N)
	}
	out := make([]ModuleHealth, 0, len(order))
	for _, code := range order {
		h := *byCode[code]
		started := h.Counts.Learning + h.Counts.Shaky + h.Counts.Mastered + h.Counts.ReviewDue
		h.Counts.NotStarted = h.Total - started
		if h.Counts.NotStarted < 0 {
			h.Counts.NotStarted = 0
		}
		out = append(out, h)
	}
	return out, nil
}

func (s *Service) urgentItems(childID int64, limit int) ([]UrgentItem, error) {
	var out []UrgentItem
	err := s.db.Raw(`
		SELECT ms.kp_id, kp.title, s.code AS subject_code, s.name AS subject_name, m.name AS module_name,
		       `+effectiveStatusSQL+` AS status,
		       CASE WHEN ms.attempts > 0 THEN CAST(ms.correct AS REAL) / ms.attempts ELSE 0 END AS accuracy,
		       (SELECT COUNT(*) FROM attempts a WHERE a.child_id=ms.child_id AND a.kp_id=ms.kp_id AND a.source='quiz' AND a.is_correct=false) AS wrong_count
		FROM mastery_states ms
		JOIN knowledge_points kp ON kp.id = ms.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE ms.child_id = ?
		  AND s.code <> 'game'
		  AND (`+effectiveStatusSQL+`) IN ('shaky','review_due')
		ORDER BY CASE WHEN (`+effectiveStatusSQL+`) = 'shaky' THEN 0 ELSE 1 END,
		         accuracy ASC, wrong_count DESC
		LIMIT ?`, childID, limit).Scan(&out).Error
	for i := range out {
		if out[i].Status == "shaky" {
			out[i].Reason = "当前待巩固，可查看具体作答"
		} else {
			out[i].Reason = "曾经掌握，该复习了"
		}
	}
	return out, err
}

func (s *Service) skillGaps(childID int64, subjectCode string) ([]SkillGap, error) {
	return s.skillGapsWhere(childID, "s.code = ?", subjectCode)
}

func (s *Service) allSkillGaps(childID int64) ([]SkillGap, error) {
	return s.skillGapsWhere(childID, "1 = 1")
}

func (s *Service) skillGapsWhere(childID int64, extra string, args ...any) ([]SkillGap, error) {
	type row struct {
		KpID      int64
		Title     string
		Code      string
		SkillCode string
		Status    string
		Attempts  int
	}
	query := `
		SELECT kp.id AS kp_id, kp.title, s.code, sk.skill_code, sk.status, sk.attempts
		FROM mastery_skills sk
		JOIN knowledge_points kp ON kp.id = sk.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE sk.child_id = ? AND s.code <> 'game' AND ` + extra
	args = append([]any{childID}, args...)
	var rows []row
	if err := s.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byKp := map[int64][]row{}
	for _, r := range rows {
		byKp[r.KpID] = append(byKp[r.KpID], r)
	}
	var out []SkillGap
	for _, list := range byKp {
		var strong, weak *row
		for i := range list {
			st := list[i].Status
			if st == "mastered" || st == "review_due" {
				strong = &list[i]
			}
			if list[i].Attempts > 0 && (st == "shaky" || st == "learning") {
				if weak == nil || st == "shaky" {
					weak = &list[i]
				}
			}
		}
		if strong == nil || weak == nil || strong.SkillCode == weak.SkillCode {
			continue
		}
		if weak.Status == "learning" && strong.Status == "review_due" {
			continue
		}
		if weak.Status != "shaky" && weak.Status != "not_started" {
			// keep learning as weak only if strong is mastered
			if weak.Status != "learning" {
				continue
			}
		}
		out = append(out, SkillGap{
			KpID:        strong.KpID,
			Title:       strong.Title,
			StrongSkill: strong.SkillCode,
			StrongLabel: SkillLabel(strong.Code, strong.SkillCode),
			WeakSkill:   weak.SkillCode,
			WeakLabel:   SkillLabel(weak.Code, weak.SkillCode),
		})
	}
	return out, nil
}

func (s *Service) skillStatusCount(childID int64, subjectCode, skillCode, status string) (int, error) {
	var n int
	err := s.db.Raw(`
		SELECT COUNT(1) FROM mastery_skills sk
		JOIN knowledge_points kp ON kp.id = sk.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE sk.child_id = ? AND s.code = ? AND sk.skill_code = ? AND sk.status = ?`,
		childID, subjectCode, skillCode, status).Scan(&n).Error
	return n, err
}

func addCount(c *StatusCounts, status string, n int) {
	switch status {
	case "learning":
		c.Learning += n
	case "shaky":
		c.Shaky += n
	case "mastered":
		c.Mastered += n
	case "review_due":
		c.ReviewDue += n
	case "not_started":
		c.NotStarted += n
	}
}

func healthOf(c StatusCounts) string {
	if c.Shaky >= 2 {
		return "weak"
	}
	if c.Shaky > 0 || c.ReviewDue > 0 || (c.Learning > 0 && c.Mastered == 0) {
		return "watch"
	}
	return "ok"
}

func healthSummary(name string, c StatusCounts) string {
	switch healthOf(c) {
	case "weak":
		return name + "有多处需巩固"
	case "watch":
		return name + "有待稳住的知识点"
	default:
		return name + "目前比较稳"
	}
}

func SkillLabel(subject, code string) string {
	switch code {
	case mastery.SkillGlyphSense:
		return "看字选义"
	case mastery.SkillSenseChar:
		return "看义选字"
	case mastery.SkillWriteChar:
		return "手写"
	case mastery.SkillPinyinInWord:
		return "听例字"
	case mastery.SkillPinyinListen:
		if subject == "english" {
			return "听单词"
		}
		return "听单音"
	case mastery.SkillEnglishPicture:
		return "看图选词"
	case mastery.SkillMathCalc:
		return "算式"
	case mastery.SkillMathStory:
		return "应用题"
	case mastery.SkillMathFind:
		return "找图形"
	case mastery.SkillMathName:
		return "认名称"
	case mastery.SkillScienceRecognize:
		return "认知识卡"
	default:
		return code
	}
}

func lastPick(picks string) (int, bool) {
	parts := strings.Split(picks, ",")
	last := strings.TrimSpace(parts[len(parts)-1])
	if last == "" {
		return 0, false
	}
	n, err := strconv.Atoi(last)
	return n, err == nil
}

func answerIndex(raw string) (int, bool) {
	var m struct {
		Index *int `json:"index"`
	}
	if json.Unmarshal([]byte(raw), &m) != nil || m.Index == nil {
		return 0, false
	}
	return *m.Index, true
}
