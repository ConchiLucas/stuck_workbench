package knowledge

import "fmt"

// ReviewStatusSummary counts open suggestions, rather than tasks or attempts.
// Answered means at least one attributable quiz attempt, not completion/mastery.
type ReviewStatusSummary struct {
	Pending  int `json:"pending"`
	Draft    int `json:"draft"`
	Awaiting int `json:"awaiting"`
	Answered int `json:"answered"`
	Total    int `json:"total"`
}

func (s *Service) ReviewStatusSummary(child int64) (ReviewStatusSummary, error) {
	out := ReviewStatusSummary{}
	if err := s.child(child); err != nil {
		return out, err
	}
	// Task-admin owns these schemas. Never migrate or write them from here. Missing
	// tables are different from a fully installed system with no open suggestions.
	if e := s.reviewSchemaAvailable(); e != nil {
		return out, e
	}
	// A historical child plan remains proof of publication even when the task's
	// published revision has since changed. A plan from another revision does not.
	err := s.DB.Raw(`SELECT
 COALESCE(SUM(CASE WHEN stage=0 THEN 1 ELSE 0 END),0) AS pending,
 COALESCE(SUM(CASE WHEN stage=1 THEN 1 ELSE 0 END),0) AS draft,
 COALESCE(SUM(CASE WHEN stage=2 THEN 1 ELSE 0 END),0) AS awaiting,
 COALESCE(SUM(CASE WHEN stage=3 THEN 1 ELSE 0 END),0) AS answered,
 COUNT(*) AS total
 FROM (
`+fmt.Sprintf(reviewStagesSQL, "")+`
 ) stages`, child).Scan(&out).Error
	if err != nil {
		return ReviewStatusSummary{}, err
	}
	return out, nil
}

const reviewStagesSQL = `  SELECT rs.id, MAX(CASE
   WHEN EXISTS (
    SELECT 1 FROM study_plans p
    JOIN plan_items i ON i.plan_id=p.id
    JOIN question_versions v ON v.id=i.question_version_id AND v.revision_id=link.generated_revision_id AND v.kp_id=i.kp_id
    JOIN question_attempt_receipts r ON r.plan_id=p.id AND r.plan_item_id=i.id AND r.question_version_id=v.id AND r.kp_id=v.kp_id AND r.question_type=v.question_type AND r.child_id=p.child_id
    JOIN attempts a ON a.id=r.attempt_id AND a.child_id=r.child_id AND a.kp_id=r.kp_id AND a.source='quiz'
    WHERE p.child_id=rs.child_id AND p.source_question_task_id=t.id AND p.source_question_task_revision_id=rev.id
   ) THEN 3
   WHEN t.published_revision_id=rev.id OR EXISTS (
    SELECT 1 FROM study_plans p WHERE p.child_id=rs.child_id AND p.source_question_task_id=t.id AND p.source_question_task_revision_id=rev.id
   ) THEN 2
   WHEN rev.id IS NOT NULL THEN 1
   ELSE 0 END) AS stage
  FROM review_suggestions rs
  LEFT JOIN review_suggestion_tasks link ON link.suggestion_id=rs.id
  LEFT JOIN question_tasks t ON t.id=link.task_id AND (t.target_child_id IS NULL OR t.target_child_id=rs.child_id)
  LEFT JOIN question_task_revisions rev ON rev.id=link.generated_revision_id AND rev.task_id=t.id
  WHERE rs.child_id=? AND rs.lifecycle='open' %s
  GROUP BY rs.id`

type ReviewStage struct {
	ID    int64  `json:"id"`
	Stage string `json:"stage"`
}
type ReviewStages struct {
	Items []ReviewStage `json:"items"`
}

func (s *Service) ReviewStages(child int64, ids []int64) (ReviewStages, error) {
	out := ReviewStages{Items: []ReviewStage{}}
	if len(ids) == 0 || len(ids) > 100 {
		return out, bad("invalid_ids", "请提供1至100个建议编号")
	}
	for _, id := range ids {
		if id < 1 {
			return out, bad("invalid_ids", "建议编号无效")
		}
	}
	if e := s.child(child); e != nil {
		return out, e
	}
	if e := s.reviewSchemaAvailable(); e != nil {
		return out, e
	}
	rows := []struct {
		ID    int64
		Stage int
	}{}
	if e := s.DB.Raw(fmt.Sprintf(reviewStagesSQL, "AND rs.id IN ?")+" ORDER BY rs.id", child, ids).Scan(&rows).Error; e != nil {
		return out, e
	}
	labels := []string{"pending", "draft", "awaiting", "answered"}
	for _, r := range rows {
		out.Items = append(out.Items, ReviewStage{r.ID, labels[r.Stage]})
	}
	return out, nil
}

func (s *Service) reviewSchemaAvailable() error {
	for _, table := range []string{"review_suggestions", "review_suggestion_tasks", "question_tasks", "question_task_revisions", "question_versions", "question_attempt_receipts", "study_plans", "plan_items", "attempts"} {
		if !s.DB.Migrator().HasTable(table) {
			return &Fault{503, "review_status_unavailable", "复习状态数据尚不可用"}
		}
	}
	return nil
}
