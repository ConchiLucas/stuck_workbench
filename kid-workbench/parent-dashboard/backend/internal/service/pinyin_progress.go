package service

import (
	"context"
	"errors"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/conchi/study-workbench/internal/model"
	"time"
)

type PinyinTypeProgress struct {
	Mastered           int `json:"mastered"`
	AnsweredUnmastered int `json:"answered_unmastered"`
	Unattempted        int `json:"unattempted"`
	Total              int `json:"total"`
}

func addStatus(c *StatusCounts, status string, n int) {
	switch status {
	case "mastered":
		c.Mastered += n
	case "review_due":
		c.ReviewDue += n
	case "learning":
		c.Learning += n
	case "shaky":
		c.Shaky += n
	default:
		c.NotStarted += n
	}
}
func (s *DashboardService) pinyinMatrix(childID int64) (Matrix, error) {
	d := s.repo.DB()
	out := Matrix{RuleVersion: 2, CatalogAvailable: true, Modules: []MatrixModule{}, TypeProgress: map[string]PinyinTypeProgress{}}
	for _, code := range mastery.PinyinSkills {
		out.TypeProgress[code] = PinyinTypeProgress{}
	}
	if e := pinyincatalog.Sync(context.Background(), d); e != nil {
		if errors.Is(e, pinyincatalog.ErrUnavailable) {
			out.CatalogAvailable = false
		} else {
			return out, e
		}
	}
	var sub model.Subject
	if e := d.Where("code=?", "pinyin").First(&sub).Error; e != nil {
		return out, e
	}
	out.Subject = SubjectSummary{Code: sub.Code, Name: sub.Name, Icon: sub.Icon}
	var audits []learningmodel.PinyinUpgradeAudit
	if e := d.Where("child_id=? AND rule_version=2", childID).Order("applied_at").Limit(1).Find(&audits).Error; e != nil {
		return out, e
	}
	if len(audits) > 0 {
		out.RuleEffectiveAt = &audits[0].AppliedAt
	}
	type pointRow struct {
		ID                            int64
		Title, ModuleCode, ModuleName string
		Initial, Final, Syllable      string
		Tone                          int
		Attempts, Correct             int
		DueAt                         *time.Time
		LastAt                        *time.Time
	}
	var rows []pointRow
	e := d.Raw(`SELECT kp.id,kp.title,m.code AS module_code,m.name AS module_name,
 COALESCE(pl.initial_text,'') AS initial,COALESCE(pl.final_text,'') AS final,COALESCE(pl.tone,0) AS tone,COALESCE(pl.syllable_text,'') AS syllable,
 COALESCE(a.attempts,0) AS attempts,COALESCE(a.correct,0) AS correct,ms.due_at,latest.created_at AS last_at
 FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id
 LEFT JOIN pinyin_syllable_links pl ON pl.kp_id=kp.id
 LEFT JOIN mastery_states ms ON ms.child_id=? AND ms.kp_id=kp.id
 LEFT JOIN attempts latest ON latest.id=(SELECT la.id FROM attempts la WHERE la.kp_id=kp.id AND la.child_id=? AND la.source<>'parent_mark' ORDER BY la.created_at DESC,la.id DESC LIMIT 1)
 LEFT JOIN (SELECT kp_id,COUNT(*) AS attempts,SUM(CASE WHEN is_correct THEN 1 ELSE 0 END) AS correct FROM attempts WHERE child_id=? AND source <> 'parent_mark' GROUP BY kp_id) a ON a.kp_id=kp.id
 WHERE m.subject_id=? AND (m.code <> 'syllables' OR pl.enabled=true)
 ORDER BY m.order_no,kp.order_no,kp.id`, childID, childID, childID, sub.ID).Scan(&rows).Error
	if e != nil {
		return out, e
	}
	ids := []int64{}
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	skills := map[int64]map[string]model.MasterySkill{}
	if len(ids) > 0 {
		ss, e := s.repo.ListMasterySkills(d, childID, ids)
		if e != nil {
			return out, e
		}
		for _, sk := range ss {
			if skills[sk.KpID] == nil {
				skills[sk.KpID] = map[string]model.MasterySkill{}
			}
			skills[sk.KpID][sk.SkillCode] = sk
		}
	}
	now := time.Now()
	indices := map[string]int{}
	for _, r := range rows {
		i, ok := indices[r.ModuleCode]
		if !ok {
			i = len(out.Modules)
			indices[r.ModuleCode] = i
			out.Modules = append(out.Modules, MatrixModule{Code: r.ModuleCode, Name: r.ModuleName, Points: []MatrixPoint{}})
		}
		list, status := skillsFromRows(mastery.SkillsFor("pinyin", r.ModuleCode), skills[r.ID], now)
		p := MatrixPoint{ID: r.ID, Title: r.Title, Kind: "letter", Status: string(status), Skills: list, Attempts: r.Attempts, DueAt: r.DueAt, LastAt: r.LastAt, Initial: r.Initial, Final: r.Final, Tone: r.Tone, Syllable: r.Syllable}
		if r.ModuleCode == "syllables" {
			p.Kind = "syllable"
		}
		if r.Attempts > 0 {
			p.Accuracy = float64(r.Correct) / float64(r.Attempts)
		}
		m := &out.Modules[i]
		m.Total++
		if mastery.IsSkillDone(status) {
			m.Mastered++
		}
		m.Points = append(m.Points, p)
		out.Subject.Total++
		addStatus(&out.Subject.Counts, p.Status, 1)
		for _, sk := range list {
			v := out.TypeProgress[sk.Code]
			v.Total++
			if mastery.IsSkillDone(mastery.Status(sk.Status)) {
				v.Mastered++
			} else if sk.Attempts > 0 {
				v.AnsweredUnmastered++
			} else {
				v.Unattempted++
			}
			out.TypeProgress[sk.Code] = v
		}
	}
	if out.Subject.Total > 0 {
		out.Subject.Progress = float64(out.Subject.Counts.Mastered+out.Subject.Counts.ReviewDue) / float64(out.Subject.Total)
	}
	if len(ids) > 0 {
		var n int64
		if e := d.Model(&learningmodel.PinyinMasteryMilestone{}).Where("child_id=? AND rule_version=2 AND kp_id IN ? AND first_completed_at >= ?", childID, ids, now.AddDate(0, 0, -7)).Count(&n).Error; e != nil {
			return out, e
		}
		out.Subject.WeekNew = int(n)
	}
	return out, nil
}
func (s *DashboardService) pinyinDetail(childID int64, out KpDetail) (KpDetail, error) {
	d := s.repo.DB()
	out.Kind = "letter"
	out.MasteredAt = nil
	if out.ModuleCode == "syllables" {
		var link learningmodel.PinyinSyllableLink
		if e := d.Where("kp_id=?", out.KpID).First(&link).Error; e != nil {
			return out, e
		}
		out.Kind = "syllable"
		out.Initial = link.InitialText
		out.Final = link.FinalText
		out.Tone = link.Tone
		out.Syllable = link.SyllableText
	}
	rows, e := s.repo.ListMasterySkills(d, childID, []int64{out.KpID})
	if e != nil {
		return out, e
	}
	byCode := map[string]model.MasterySkill{}
	for _, r := range rows {
		byCode[r.SkillCode] = r
	}
	list, status := skillsFromRows(mastery.SkillsFor("pinyin", out.ModuleCode), byCode, time.Now())
	out.Skills = list
	out.Status = string(status)
	var milestones []learningmodel.PinyinMasteryMilestone
	if e := d.Where("child_id=? AND kp_id=? AND rule_version=2", childID, out.KpID).Find(&milestones).Error; e != nil {
		return out, e
	}
	if len(milestones) > 0 {
		out.MasteredAt = &milestones[0].FirstCompletedAt
	}
	out.History = []HistoryItem{}
	if e := d.Raw(`SELECT a.created_at AS at,a.is_correct,a.cost_ms,a.source,COALESCE(pr.skill_code,q.code,'') AS skill_code,a.id AS attempt_id,COALESCE(pr.selected_option_id,'') AS selected_option_id,COALESCE(pi.public_snapshot,'') AS question_snapshot
 FROM attempts a LEFT JOIN questions q ON q.id=a.question_id LEFT JOIN pinyin_answer_receipts pr ON pr.attempt_id=a.id LEFT JOIN pinyin_quiz_instances pi ON pi.id=pr.instance_id
 WHERE a.child_id=? AND a.kp_id=? ORDER BY a.created_at DESC,a.id DESC`, childID, out.KpID).Scan(&out.History).Error; e != nil {
		return out, e
	}
	for i := range out.History {
		item := &out.History[i]
		if item.Source == "parent_mark" {
			item.QuestionSnapshot = ""
			continue
		}
		item.PinyinReview = buildPinyinReview(item.QuestionSnapshot, item.SelectedOptionID)
		item.QuestionSnapshot = ""
	}
	out.Attempts = 0
	out.Correct = 0
	out.Accuracy = 0
	for _, h := range out.History {
		if h.Source != "parent_mark" {
			out.Attempts++
			if h.IsCorrect {
				out.Correct++
			}
		}
	}
	if out.Attempts > 0 {
		out.Accuracy = float64(out.Correct) / float64(out.Attempts)
	}
	return out, nil
}
func (s *DashboardService) correctPinyinOverview(childID int64, out *Overview) error {
	var n int64
	d := s.repo.DB()
	if e := d.Model(&model.Subject{}).Where("code=?", "pinyin").Count(&n).Error; e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	current, e := s.pinyinMatrix(childID)
	if e != nil {
		return e
	}
	var old []struct {
		Status string
		N      int
	}
	if e := d.Raw(`SELECT `+effectiveStatusSQL+` AS status,COUNT(*) AS n FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id LEFT JOIN mastery_states ms ON ms.kp_id=kp.id AND ms.child_id=? WHERE s.code='pinyin' GROUP BY 1`, childID).Scan(&old).Error; e != nil {
		return e
	}
	for _, r := range old {
		out.TotalKp -= r.N
		addStatus(&out.Counts, r.Status, -r.N)
	}
	out.TotalKp += current.Subject.Total
	c := current.Subject.Counts
	out.Counts.NotStarted += c.NotStarted
	out.Counts.Learning += c.Learning
	out.Counts.Shaky += c.Shaky
	out.Counts.Mastered += c.Mastered
	out.Counts.ReviewDue += c.ReviewDue
	return nil
}
