package service

import (
	"encoding/json"
	"time"

	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
)

func (s *DashboardService) logicDetail(childID int64, out KpDetail) (KpDetail, error) {
	d := s.repo.DB()
	rows, err := s.repo.ListMasterySkills(d, childID, []int64{out.KpID})
	if err != nil {
		return out, err
	}
	byCode := map[string]model.MasterySkill{}
	for _, r := range rows {
		byCode[r.SkillCode] = r
	}
	list, _ := skillsFromRows(mastery.LogicSkills, byCode, time.Now())
	out.Skills = list
	if !mastery.IsSkillDone(mastery.Status(out.Status)) {
		out.MasteredAt = nil
	}
	type hist struct {
		HistoryItem
		PlanID int64
		ItemID int64
	}
	var items []hist
	join := "LEFT JOIN plan_items pi ON 1=0"
	if d.Migrator().HasColumn("attempts", "plan_item_id") {
		join = "LEFT JOIN plan_items pi ON pi.id=a.plan_item_id"
	}
	selectedExpr := "COALESCE(pi.picks,'')"
	if d.Migrator().HasColumn("attempts", "selected") {
		selectedExpr = "COALESCE(NULLIF(a.selected,''), CASE WHEN COALESCE(pi.picks,'') LIKE '[%' THEN '' ELSE COALESCE(pi.picks,'') END, '')"
	}
	if err := d.Raw(`SELECT a.created_at AS at,a.is_correct,a.cost_ms,a.source,COALESCE(q.code,'') AS skill_code,a.id AS attempt_id,`+selectedExpr+` AS selected_option_id,COALESCE(CAST(pi.question_snapshot AS TEXT),'') AS question_snapshot,COALESCE(pi.plan_id,0) AS plan_id,COALESCE(pi.id,0) AS item_id
 FROM attempts a LEFT JOIN questions q ON q.id=a.question_id
 `+join+`
 WHERE a.child_id=? AND a.kp_id=? ORDER BY a.created_at ASC,a.id ASC`, childID, out.KpID).Scan(&items).Error; err != nil {
		return out, err
	}
	out.History = make([]HistoryItem, 0, len(items))
	out.Attempts, out.Correct, out.Accuracy = 0, 0, 0
	for i := range items {
		h := items[i].HistoryItem
		h.SkillCode = mastery.SkillFromQuestionCode("logic", h.SkillCode)
		if h.Source != "parent_mark" {
			if items[i].ItemID == 0 {
				h.LogicReview = &LogicReview{UnavailableReason: "这条记录没有关联到当时那一题，无法还原作答画面。"}
			} else {
				h.LogicReview = buildLogicReview(h.QuestionSnapshot, h.SelectedOptionID)
				var snap logiccontent.HistorySnapshot
				if json.Unmarshal([]byte(h.QuestionSnapshot), &snap) == nil && h.LogicReview.Example != nil {
					if err := logiccontent.VerifyMedia(d, logiccontent.SnapshotToExample(snap)); err != nil {
						h.LogicReview.UnavailableReason = err.Error() + "，无法可靠还原当时图片；作答记录仍保留。"
					}
				}
			}
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

func logicKindFromPayload(payload, module string) string {
	if mastery.SkillFromQuestionCode("logic", module) != "" {
		return module
	}
	var parsed struct {
		Kind    string `json:"kind"`
		Example struct {
			Kind string `json:"kind"`
		} `json:"example"`
	}
	_ = json.Unmarshal([]byte(payload), &parsed)
	if mastery.SkillFromQuestionCode("logic", parsed.Example.Kind) != "" {
		return parsed.Example.Kind
	}
	if mastery.SkillFromQuestionCode("logic", parsed.Kind) != "" {
		return parsed.Kind
	}
	return ""
}
