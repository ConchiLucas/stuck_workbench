package reviewsuggestion

import (
	"context"
	"encoding/json"
	"fmt"
)

type Page struct {
	Items      any    `json:"items"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
}

func pageLimit(n int) int {
	if n < 1 || n > 100 {
		return 30
	}
	return n
}
func (s *Service) EvidencePage(ctx context.Context, child, id, after int64, limit int) (Page, error) {
	out := Page{Items: []Evidence{}}
	if e := s.owned(ctx, child, id); e != nil {
		return out, e
	}
	limit = pageLimit(limit)
	rows := []Evidence{}
	e := s.DB.WithContext(ctx).Table("review_suggestion_evidence e").Select("e.*").Joins("JOIN review_suggestion_targets t ON t.id=e.target_id").Where("t.suggestion_id=? AND e.id>?", id, after).Order("e.id").Limit(limit + 1).Scan(&rows).Error
	if e != nil {
		return out, e
	}
	if len(rows) > limit {
		out.HasMore = true
		rows = rows[:limit]
		out.NextCursor = fmt.Sprint(rows[len(rows)-1].ID)
	}
	for i := range rows {
		_ = json.Unmarshal([]byte(rows[i].SourceJSON), &rows[i].Source)
		rows[i].Summary = json.RawMessage(rows[i].EvidenceSummaryJSON)
	}
	out.Items = rows
	return out, nil
}
func (s *Service) TasksPage(ctx context.Context, child, id, after int64, limit int) (Page, error) {
	out := Page{Items: []TaskLink{}}
	if e := s.owned(ctx, child, id); e != nil {
		return out, e
	}
	limit = pageLimit(limit)
	rows := []TaskLink{}
	e := s.DB.WithContext(ctx).Where("suggestion_id=? AND id>?", id, after).Order("id").Limit(limit + 1).Find(&rows).Error
	if e != nil {
		return out, e
	}
	if len(rows) > limit {
		out.HasMore = true
		rows = rows[:limit]
		out.NextCursor = fmt.Sprint(rows[len(rows)-1].ID)
	}
	for i := range rows {
		rows[i].TargetMap = json.RawMessage(rows[i].TargetMapJSON)
		if e := s.taskState(ctx, &rows[i]); e != nil {
			return out, e
		}
	}
	out.Items = rows
	return out, nil
}

type PlanLink struct {
	ActualPlanRevisionID int64  `json:"actualPlanRevisionId"`
	RevisionChanged      bool   `json:"revisionChanged"`
	AttributionStatus    string `json:"attributionStatus"`
	PlanID               int64  `json:"planId"`
	TaskID               int64  `json:"taskId"`
	GeneratedRevisionID  int64  `json:"generatedRevisionId"`
	Status               string `json:"status"`
}

func (s *Service) PlansPage(ctx context.Context, child, id, after int64, limit int) (Page, error) {
	out := Page{Items: []PlanLink{}}
	if e := s.owned(ctx, child, id); e != nil {
		return out, e
	}
	if !s.DB.Migrator().HasTable("study_plans") {
		return out, nil
	}
	limit = pageLimit(limit)
	rows := []PlanLink{}
	e := s.DB.WithContext(ctx).Table("study_plans p").Select("p.id AS plan_id,t.task_id,t.generated_revision_id,p.source_question_task_revision_id AS actual_plan_revision_id,p.status").Joins("JOIN review_suggestion_tasks t ON t.task_id=p.source_question_task_id").Where("t.suggestion_id=? AND p.child_id=? AND p.id>?", id, child, after).Order("p.id").Limit(limit + 1).Scan(&rows).Error
	if e != nil {
		return out, e
	}
	if len(rows) > limit {
		out.HasMore = true
		rows = rows[:limit]
		out.NextCursor = fmt.Sprint(rows[len(rows)-1].PlanID)
	}
	for i := range rows {
		rows[i].RevisionChanged = rows[i].ActualPlanRevisionID != rows[i].GeneratedRevisionID
		rows[i].AttributionStatus = "generated_revision"
		if rows[i].RevisionChanged {
			rows[i].AttributionStatus = "revision_changed"
		}
	}
	out.Items = rows
	return out, nil
}
