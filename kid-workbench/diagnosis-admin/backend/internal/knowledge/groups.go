package knowledge

import (
	"fmt"
	"time"
)

type groupCount struct {
	KpID                                    int64
	SkillCode                               string
	WrongCount, InstanceCount, UnknownCount int
}

// Groups aggregates in SQL and returns at most twenty recent evidence references
// per group. Complete historical facts remain available through Attempts.
func (s *Service) Groups(child int64, f Filter) (Page[Candidate], error) {
	out := Page[Candidate]{Items: []Candidate{}, StateReadAt: s.Now(), Coverage: Coverage{"partial", []string{"legacy_selection_missing", "group_evidence_limited_20"}}}
	if e := normalize(&f); e != nil {
		return out, e
	}
	if e := s.child(child); e != nil {
		return out, e
	}
	c, e := s.pageCursor(child, f)
	if e != nil {
		return out, e
	}
	out.EvidenceAsOf = c.AsOf
	f.WrongOnly = true
	if f.FollowUpState == "assisted" || f.FollowUpState == "assisted_completion" {
		f.WrongOnly = false
	}
	offset := c.Offset
	for {
		q, skill, instance := s.instanceBase(child, f, c.Max, c.AsOf)
		groups := []groupCount{}
		e = q.Select("a.kp_id," + skill + " AS skill_code,SUM(CASE WHEN a.is_correct THEN 0 ELSE 1 END) AS wrong_count,COUNT(DISTINCT " + instance + ") AS instance_count,SUM(CASE WHEN " + instance + " IS NULL THEN 1 ELSE 0 END) AS unknown_count,MAX(a.created_at) AS last_wrong_at").Group("a.kp_id," + skill).Order("last_wrong_at DESC,a.kp_id,skill_code").Offset(offset).Limit(f.Limit + 1).Scan(&groups).Error
		if e != nil {
			return out, e
		}
		if len(groups) == 0 {
			break
		}
		for _, group := range groups {
			g, err := s.groupDetails(child, f, c, group)
			if err != nil {
				return out, err
			}
			if g != nil {
				if len(out.Items) == f.Limit {
					out.HasMore = true
					c.Offset = offset
					out.NextCursor = encodeCursor(c)
					return out, nil
				}
				out.Items = append(out.Items, *g)
			}
			offset++
		}
		if len(groups) < f.Limit+1 {
			break
		}
	}
	return out, nil
}
func (s *Service) groupDetails(child int64, f Filter, c cursor, g groupCount) (*Candidate, error) {
	f.KpID = g.KpID
	f.Skill = g.SkillCode
	q, skill := s.boundedBase(child, f, c.Max, c.AsOf)
	candidate := &Candidate{Key: fmt.Sprintf("%d:%s", g.KpID, g.SkillCode), KpID: g.KpID, SkillCode: g.SkillCode, WrongCount: g.WrongCount, Evidence: []CandidateEvidence{}, ReasonCode: "observed_wrong", PreferredDistractorKpIDs: []int64{}, ReviewBlockReasons: []string{"use_review_candidates"}, Mode: "mixed", RequestedCount: 3}
	filtered := f.FollowUpState != "" && f.FollowUpState != "all"
	if !filtered && g.UnknownCount == 0 {
		n := g.InstanceCount
		candidate.WrongInstanceCount = &n
	}
	observed := 0
	visit := func(rows []Evidence) (bool, error) {
		if filtered {
			if e := s.followUps(rows, c.Max, c.AsOf); e != nil {
				return false, e
			}
		}
		for _, r := range rows {
			if !matchesFollowUp(r, f.FollowUpState) {
				continue
			}
			observed++
			if observed == 1 {
				candidate.LastWrongAt = r.OccurredAt
				candidate.Title = r.Title
				candidate.SubjectCode = r.SubjectCode
				candidate.ModuleCode = r.ModuleCode
				candidate.QuestionType = r.QuestionType
				if filtered {
					candidate.LastWrongAt = r.OccurredAt
					candidate.WrongCount = 0
				}
			}
			if filtered && !r.IsCorrect {
				candidate.WrongCount++
			}
			role := "target_error"
			if r.IsCorrect {
				role = "assisted_completion"
				candidate.ReasonCode = role
			}
			if len(candidate.Evidence) < groupEvidenceLimit {
				candidate.Evidence = append(candidate.Evidence, CandidateEvidence{r.AttemptID, role, r.Source})
			} else {
				candidate.EvidenceHasMore = true
			}
		}
		return filtered, nil
	}
	if filtered {
		if e := s.streamEvidence(q, skill, visit); e != nil {
			return nil, e
		}
	} else {
		rows, e := s.evidenceBatch(q, skill, time.Time{}, 0, groupEvidenceLimit+1)
		if e != nil {
			return nil, e
		}
		if _, e = visit(rows); e != nil {
			return nil, e
		}
	}
	if observed == 0 {
		return nil, nil
	}
	candidate.ReasonText = fmt.Sprintf("%s已记录 %d 次错误", Label(candidate.SubjectCode, candidate.SkillCode), candidate.WrongCount)
	if candidate.ReasonCode == "assisted_completion" {
		candidate.ReasonText = fmt.Sprintf("%s已记录提示后完成", Label(candidate.SubjectCode, candidate.SkillCode))
	}
	return candidate, nil
}
