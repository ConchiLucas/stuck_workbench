package service

import (
	"time"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-workbench/internal/model"
)

func (s *DashboardService) mathMatrix(childID int64) (Matrix, error) {
	d := s.repo.DB()
	out := Matrix{CatalogAvailable: true, Modules: []MatrixModule{}, TypeProgress: map[string]PinyinTypeProgress{}}
	for _, code := range append(append([]string{}, mastery.MathArithmeticSkills...), mastery.MathShapeSkills...) {
		out.TypeProgress[code] = PinyinTypeProgress{}
	}
	var sub model.Subject
	if e := d.Where("code=?", "math").First(&sub).Error; e != nil {
		return out, e
	}
	out.Subject = SubjectSummary{Code: sub.Code, Name: sub.Name, Icon: sub.Icon}
	type pointRow struct {
		ID                            int64
		Title, ModuleCode, ModuleName string
		Attempts, Correct             int
		DueAt, MasteredAt             *time.Time
	}
	var rows []pointRow
	e := d.Raw(`SELECT kp.id,kp.title,m.code AS module_code,m.name AS module_name,
 COALESCE(a.attempts,0) AS attempts,COALESCE(a.correct,0) AS correct,ms.due_at,ms.mastered_at
 FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id
 LEFT JOIN mastery_states ms ON ms.child_id=? AND ms.kp_id=kp.id
 LEFT JOIN (SELECT kp_id,COUNT(*) AS attempts,SUM(CASE WHEN is_correct THEN 1 ELSE 0 END) AS correct FROM attempts WHERE child_id=? AND source <> 'parent_mark' GROUP BY kp_id) a ON a.kp_id=kp.id
 WHERE m.subject_id=?
 ORDER BY m.order_no,kp.order_no,kp.id`, childID, childID, sub.ID).Scan(&rows).Error
	if e != nil {
		return out, e
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	skills := map[int64]map[string]model.MasterySkill{}
	if len(ids) > 0 {
		ss, err := s.repo.ListMasterySkills(d, childID, ids)
		if err != nil {
			return out, err
		}
		for _, sk := range ss {
			if skills[sk.KpID] == nil {
				skills[sk.KpID] = map[string]model.MasterySkill{}
			}
			skills[sk.KpID][sk.SkillCode] = sk
		}
	}
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	indices := map[string]int{}
	for _, r := range rows {
		i, ok := indices[r.ModuleCode]
		if !ok {
			i = len(out.Modules)
			indices[r.ModuleCode] = i
			out.Modules = append(out.Modules, MatrixModule{Code: r.ModuleCode, Name: r.ModuleName, Points: []MatrixPoint{}})
		}
		list, status := skillsFromRows(mastery.SkillsFor("math", r.ModuleCode), skills[r.ID], now)
		p := MatrixPoint{ID: r.ID, Title: r.Title, Status: string(status), Skills: list, Attempts: r.Attempts, DueAt: r.DueAt}
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
		if mastery.IsSkillDone(status) && r.MasteredAt != nil && !r.MasteredAt.Before(weekAgo) {
			out.Subject.WeekNew++
		}
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
	return out, nil
}

func (s *DashboardService) correctMathOverview(childID int64, out *Overview) error {
	var n int64
	d := s.repo.DB()
	if e := d.Model(&model.Subject{}).Where("code=?", "math").Count(&n).Error; e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	current, e := s.mathMatrix(childID)
	if e != nil {
		return e
	}
	var old []struct {
		Status string
		N      int
	}
	if e := d.Raw(`SELECT `+effectiveStatusSQL+` AS status,COUNT(*) AS n FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id LEFT JOIN mastery_states ms ON ms.kp_id=kp.id AND ms.child_id=? WHERE s.code='math' GROUP BY 1`, childID).Scan(&old).Error; e != nil {
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

func (s *DashboardService) mathDetail(childID int64, out KpDetail) (KpDetail, error) {
	d := s.repo.DB()
	rows, err := s.repo.ListMasterySkills(d, childID, []int64{out.KpID})
	if err != nil {
		return out, err
	}
	byCode := map[string]model.MasterySkill{}
	for _, r := range rows {
		byCode[r.SkillCode] = r
	}
	list, status := skillsFromRows(mastery.SkillsFor("math", out.ModuleCode), byCode, time.Now())
	out.Skills = list
	out.Status = string(status)
	if !mastery.IsSkillDone(mastery.Status(out.Status)) {
		out.MasteredAt = nil
	}
	type hist struct {
		HistoryItem
		PlanID int64
		ItemID int64
	}
	var items []hist
	if err := d.Raw(`SELECT a.created_at AS at,a.is_correct,a.cost_ms,a.source,COALESCE(q.code,'') AS skill_code,a.id AS attempt_id,COALESCE(pi.picks,'') AS selected_option_id,COALESCE(CAST(pi.question_snapshot AS TEXT),'') AS question_snapshot,COALESCE(pi.plan_id,0) AS plan_id,COALESCE(pi.id,0) AS item_id
 FROM attempts a LEFT JOIN questions q ON q.id=a.question_id
 LEFT JOIN plan_items pi ON pi.id=(
  SELECT pi2.id FROM plan_items pi2 JOIN study_plans sp2 ON sp2.id=pi2.plan_id
  WHERE pi2.question_id=a.question_id AND pi2.kp_id=a.kp_id AND sp2.child_id=a.child_id AND sp2.subject_code='math'
  ORDER BY pi2.id DESC LIMIT 1)
 WHERE a.child_id=? AND a.kp_id=? ORDER BY a.created_at ASC,a.id ASC`, childID, out.KpID).Scan(&items).Error; err != nil {
		return out, err
	}
	out.History = make([]HistoryItem, 0, len(items))
	out.Attempts, out.Correct, out.Accuracy = 0, 0, 0
	for i := range items {
		h := items[i].HistoryItem
		h.SkillCode = mastery.SkillFromQuestionCode("math", h.SkillCode)
		if h.Source != "parent_mark" {
			h.MathReview = buildMathReview(h.QuestionSnapshot, h.SelectedOptionID, childID, items[i].PlanID, items[i].ItemID)
			out.Attempts++
			if h.IsCorrect {
				out.Correct++
			}
		}
		h.QuestionSnapshot = ""
		out.History = append(out.History, h)
	}
	if out.Attempts > 0 {
		out.Accuracy = float64(out.Correct) / float64(out.Attempts)
	}
	return out, nil
}
