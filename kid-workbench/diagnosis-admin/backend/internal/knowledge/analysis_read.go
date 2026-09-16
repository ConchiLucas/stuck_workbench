package knowledge

import (
	"fmt"
	"time"
)

type analysisTotals struct {
	Wrong, Hinted, Observed int
	LastWrong               time.Time
}

// analysisEvidence selects recent instances in SQL, then retains at most twenty
// initials plus twenty recent facts and representative error/hint evidence per
// knowledge/skill. Retry counts are accumulated without retaining their payloads.
type analysisCacheValue struct {
	Rows   []Evidence
	Totals map[string]analysisTotals
}

func (s *Service) analysisEvidence(child int64, f Filter, max int64, asOf time.Time) ([]Evidence, map[string]analysisTotals, error) {
	key, e := s.factCacheKey("analysis", child, max, asOf, f)
	if e != nil {
		return nil, nil, e
	}
	var cached analysisCacheValue
	if s.cache.get(key, &cached) {
		return cached.Rows, cached.Totals, nil
	}
	rows, totals, e := s.readAnalysisEvidence(child, f, max, asOf)
	if e == nil {
		s.cache.put(key, analysisCacheValue{rows, totals})
	}
	return rows, totals, e
}
func (s *Service) readAnalysisEvidence(child int64, f Filter, max int64, asOf time.Time) ([]Evidence, map[string]analysisTotals, error) {
	start := asOf.AddDate(0, 0, -30)
	q, skill, instance := s.instanceBase(child, f, max, asOf)
	canonical := q.Where(instance + " IS NOT NULL").Select("a.id AS attempt_id,a.kp_id," + skill + " AS skill_code," + instance + " AS instance_key,a.created_at,ROW_NUMBER() OVER(PARTITION BY a.kp_id," + skill + "," + instance + " ORDER BY a.created_at,a.id) AS first_rank")
	firsts := s.DB.Table("(?) AS canonical", canonical).Where("first_rank=1 AND created_at>=?", start).Select("canonical.*,ROW_NUMBER() OVER(PARTITION BY kp_id,skill_code ORDER BY created_at DESC,attempt_id DESC) AS recent_rank")
	selected := s.DB.Table("(?) AS ranked", firsts).Where("recent_rank<=20").Select("instance_key")
	uq, usk, ui := s.instanceBase(child, f, max, asOf)
	unknown := uq.Where("a.created_at>=? AND "+ui+" IS NULL", start).Select("a.id,ROW_NUMBER() OVER(PARTITION BY a.kp_id," + usk + " ORDER BY a.created_at DESC,a.id DESC) AS recent_rank")
	unknownIDs := s.DB.Table("(?) AS unknown_facts", unknown).Where("recent_rank<=20").Select("id")
	q, skill, instance = s.instanceBase(child, f, max, asOf)
	q = q.Where("("+instance+" IN (?) OR a.id IN (?))", selected, unknownIDs)
	type sample struct {
		initial     map[string]Evidence
		recent      []Evidence
		wrong, hint *Evidence
	}
	samples := map[string]*sample{}
	totals := map[string]analysisTotals{}
	e := s.streamEvidence(q, skill, func(batch []Evidence) (bool, error) {
		for _, r := range batch {
			key := fmt.Sprintf("%d:%s", r.KpID, r.SkillCode)
			v := samples[key]
			if v == nil {
				v = &sample{initial: map[string]Evidence{}}
				samples[key] = v
			}
			if r.InstanceKey != "" {
				prev, ok := v.initial[r.InstanceKey]
				if !ok || r.OccurredAt.Before(prev.OccurredAt) || r.OccurredAt.Equal(prev.OccurredAt) && r.AttemptID < prev.AttemptID {
					v.initial[r.InstanceKey] = r
				}
			}
			if r.OccurredAt.Before(start) {
				continue
			}
			if len(v.recent) < 20 {
				v.recent = append(v.recent, r)
			}
			t := totals[key]
			t.Observed++
			if !r.IsCorrect {
				t.Wrong++
				if r.OccurredAt.After(t.LastWrong) {
					t.LastWrong = r.OccurredAt
				}
				if v.wrong == nil {
					x := r
					v.wrong = &x
				}
			}
			if r.Assistance == "hinted" && r.IsCorrect {
				t.Hinted++
				if v.hint == nil {
					x := r
					v.hint = &x
				}
			}
			totals[key] = t
		}
		return true, nil
	})
	if e != nil {
		return nil, nil, e
	}
	rows := []Evidence{}
	for _, v := range samples {
		seen := map[int64]bool{}
		add := func(r Evidence) {
			if !seen[r.AttemptID] {
				rows = append(rows, r)
				seen[r.AttemptID] = true
			}
		}
		for _, r := range v.initial {
			add(r)
		}
		for _, r := range v.recent {
			add(r)
		}
		if v.wrong != nil {
			add(*v.wrong)
		}
		if v.hint != nil {
			add(*v.hint)
		}
	}
	return rows, totals, nil
}
