package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/gin-gonic/gin"
)

type outcomeTarget struct {
	Key          string `json:"key"`
	KpID         int64  `json:"kpId"`
	QuestionType string `json:"questionType"`
}
type outcomeLink struct {
	TaskID              int64  `json:"taskId"`
	GeneratedRevisionID int64  `json:"generatedRevisionId"`
	TaskStatus          string `json:"taskStatus"`
	PublishedRevisionID *int64 `json:"publishedRevisionId"`
	TargetMap           []struct {
		QuestionVersionID int64  `json:"questionVersionId"`
		TargetKey         string `json:"targetKey"`
		Kind              string `json:"kind"`
	} `json:"targetMap"`
}
type reviewOutcome struct {
	Key                       string  `json:"key"`
	KpID                      int64   `json:"kpId"`
	QuestionType              string  `json:"questionType"`
	Title                     string  `json:"title"`
	State                     string  `json:"state"`
	ObservedAttempts          int     `json:"observedAttempts"`
	IndependentCorrect        int     `json:"independentCorrect"`
	WrongCount                int     `json:"wrongCount"`
	AssistedAttempts          int     `json:"assistedAttempts"`
	UnknownAssistanceAttempts int     `json:"unknownAssistanceAttempts"`
	OriginalAttempts          int     `json:"originalAttempts"`
	VariantAttempts           int     `json:"variantAttempts"`
	MasteryStatus             string  `json:"masteryStatus"`
	OtherLaterAttempts        int64   `json:"otherLaterAttempts"`
	AttemptIDs                []int64 `json:"attemptIds"`
	EvidenceTruncated         bool    `json:"evidenceTruncated"`
	RevisionChangedPlans      int64   `json:"revisionChangedPlans"`
}

// Outcomes only attributes receipts linked to the exact generated task revision.
// Other subsequent practice and current mastery remain separate observations.
func (h *handlers) reviewOutcomes(c *gin.Context) {
	cid, ok := parseID(c, "cid", "孩子编号无效")
	if !ok {
		return
	}
	id, ok := parseID(c, "id", "建议编号无效")
	if !ok {
		return
	}
	path := fmt.Sprintf("/api/v1/children/%d/review-suggestions/%d", cid, id)
	status, body, err := h.deps.Reviews.Request(c.Request.Context(), http.MethodGet, path, "", nil)
	if err != nil {
		knowledgeJSON(c, nil, err)
		return
	}
	if status != 200 {
		c.Data(status, "application/json", body)
		return
	}
	var spec struct {
		Targets   []outcomeTarget `json:"targets"`
		CreatedAt time.Time       `json:"createdAt"`
	}
	if err = json.Unmarshal(body, &spec); err != nil {
		knowledgeJSON(c, nil, err)
		return
	}
	links := []outcomeLink{}
	cursor := ""
	for {
		status, body, err = h.deps.Reviews.Request(c.Request.Context(), http.MethodGet, path+"/tasks?limit=100&cursor="+cursor, "", nil)
		if err != nil {
			knowledgeJSON(c, nil, err)
			return
		}
		if status != 200 {
			c.Data(status, "application/json", body)
			return
		}
		var page struct {
			Items      []outcomeLink `json:"items"`
			HasMore    bool          `json:"hasMore"`
			NextCursor string        `json:"nextCursor"`
		}
		if err = json.Unmarshal(body, &page); err != nil {
			knowledgeJSON(c, nil, err)
			return
		}
		links = append(links, page.Items...)
		if !page.HasMore {
			break
		}
		if _, err = strconv.ParseInt(page.NextCursor, 10, 64); err != nil || page.NextCursor == cursor {
			knowledgeJSON(c, nil, fmt.Errorf("invalid upstream pagination"))
			return
		}
		cursor = page.NextCursor
	}
	db := h.deps.Knowledge.DB.WithContext(c.Request.Context())
	now := time.Now()
	out := []reviewOutcome{}
	var fence int64
	if err = db.Table("attempts").Where("child_id=?", cid).Select("COALESCE(MAX(id),0)").Scan(&fence).Error; err != nil {
		knowledgeJSON(c, nil, err)
		return
	}
	for _, target := range spec.Targets {
		item := reviewOutcome{Key: target.Key, KpID: target.KpID, QuestionType: target.QuestionType, State: "not_generated", AttemptIDs: []int64{}}
		point, e := h.deps.Knowledge.Point(cid, target.KpID)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		item.Title = point.Title
		item.MasteryStatus = point.MasteryStatus
		for _, sk := range point.Skills {
			if sk.SkillCode == target.QuestionType {
				item.MasteryStatus = sk.MasteryStatus
			}
		}
		type source struct {
			task, revision, version int64
			kind                    string
		}
		sources := []source{}
		published := false
		for _, link := range links {
			for _, m := range link.TargetMap {
				if m.TargetKey == target.Key {
					sources = append(sources, source{link.TaskID, link.GeneratedRevisionID, m.QuestionVersionID, m.Kind})
					if link.PublishedRevisionID != nil && *link.PublishedRevisionID == link.GeneratedRevisionID {
						published = true
					}
				}
			}
		}
		if len(sources) > 0 {
			item.State = "not_published"
		}
		if published {
			item.State = "not_practiced"
		}
		if db.Migrator().HasColumn("study_plans", "source_question_task_revision_id") {
			for _, link := range links {
				var changed int64
				e = db.Table("study_plans").Where("child_id=? AND source_question_task_id=? AND source_question_task_revision_id<>?", cid, link.TaskID, link.GeneratedRevisionID).Count(&changed).Error
				if e != nil {
					knowledgeJSON(c, nil, e)
					return
				}
				item.RevisionChangedPlans += changed
			}
			if item.RevisionChangedPlans > 0 {
				item.State = "revision_changed"
			}
		}

		linkedIDs := map[int64]bool{}
		firstInstances := map[string]knowledge.Evidence{}
		var latestFirst *knowledge.Evidence
		if len(sources) > 0 && db.Migrator().HasTable("question_attempt_receipts") { // each source is at most one of twenty generated questions
			for _, src := range sources {
				var after int64
				for {
					var ids []int64
					e = db.Table("attempts a").Joins("JOIN question_attempt_receipts r ON r.attempt_id=a.id AND r.child_id=a.child_id").Joins("JOIN study_plans p ON p.id=r.plan_id AND p.child_id=a.child_id").Joins("JOIN plan_items pi ON pi.id=r.plan_item_id AND pi.plan_id=p.id AND pi.question_version_id=r.question_version_id AND pi.kp_id=a.kp_id").Where("a.child_id=? AND a.kp_id=? AND a.source='quiz' AND a.id>? AND a.id<=? AND p.source_question_task_id=? AND p.source_question_task_revision_id=? AND r.question_version_id=?", cid, target.KpID, after, fence, src.task, src.revision, src.version).Order("a.id").Limit(200).Pluck("a.id", &ids).Error
					if e != nil {
						knowledgeJSON(c, nil, e)
						return
					}
					if len(ids) == 0 {
						break
					}
					rows, e := h.deps.Knowledge.AttemptsByIDs(cid, ids)
					if e != nil {
						knowledgeJSON(c, nil, e)
						return
					}
					for _, r := range rows {
						if linkedIDs[r.AttemptID] {
							continue
						}
						linkedIDs[r.AttemptID] = true
						item.ObservedAttempts++
						if !r.IsCorrect {
							item.WrongCount++
						}
						if r.Assistance == "hinted" {
							item.AssistedAttempts++
						}
						if r.Assistance == "unknown" {
							item.UnknownAssistanceAttempts++
						}
						if r.Assistance == "none" && r.IsCorrect {
							item.IndependentCorrect++
						}
						if src.kind == "original" {
							item.OriginalAttempts++
						} else if src.kind == "variant" {
							item.VariantAttempts++
						}
						if len(item.AttemptIDs) < 100 {
							item.AttemptIDs = append(item.AttemptIDs, r.AttemptID)
						} else {
							item.EvidenceTruncated = true
						}
						if r.InstanceKey != "" {
							old, exists := firstInstances[r.InstanceKey]
							if !exists || r.OccurredAt.Before(old.OccurredAt) || (r.OccurredAt.Equal(old.OccurredAt) && r.AttemptID < old.AttemptID) {
								firstInstances[r.InstanceKey] = r
							}
						}
					}
					after = ids[len(ids)-1]
					if len(ids) < 200 {
						break
					}
				}
			}
		}

		for _, first := range firstInstances {
			if latestFirst == nil || first.OccurredAt.After(latestFirst.OccurredAt) || (first.OccurredAt.Equal(latestFirst.OccurredAt) && first.AttemptID > latestFirst.AttemptID) {
				copy := first
				latestFirst = &copy
			}
		}
		if item.ObservedAttempts > 0 {
			item.State = "insufficient_evidence"
			if latestFirst != nil && latestFirst.Assistance == "none" {
				if latestFirst.IsCorrect {
					item.State = "answered_correctly"
				} else {
					item.State = "still_wrong"
				}
			}
		}

		var later int64
		e = db.Table("attempts").Where("child_id=? AND kp_id=? AND source='quiz' AND created_at>? AND id<=?", cid, target.KpID, spec.CreatedAt, fence).Count(&later).Error
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		item.OtherLaterAttempts = later - int64(len(linkedIDs))
		if item.OtherLaterAttempts < 0 {
			item.OtherLaterAttempts = 0
		}
		out = append(out, item)
	}
	c.JSON(200, gin.H{"items": out, "hasMore": false, "evidenceAsOf": now, "stateReadAt": now, "coverage": gin.H{"level": "partial", "reasonCodes": []string{"generated_revision_only", "current_mastery_not_attributed_to_review"}}})
}
