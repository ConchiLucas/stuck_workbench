package knowledge

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Analyze(rows []Evidence, now time.Time) []Candidate {
	earliest := map[string]Evidence{}
	for _, r := range rows {
		if r.InstanceKey == "" || r.OccurredAt.After(now) {
			continue
		}
		key := fmt.Sprintf("%d:%s:%s", r.KpID, r.SkillCode, r.InstanceKey)
		prev, ok := earliest[key]
		if !ok || r.OccurredAt.Before(prev.OccurredAt) || r.OccurredAt.Equal(prev.OccurredAt) && r.AttemptID < prev.AttemptID {
			earliest[key] = r
		}
	}
	by := map[string][]Evidence{}
	for _, r := range rows {
		if r.OccurredAt.Before(now.AddDate(0, 0, -30)) || r.OccurredAt.After(now) || r.SkillCode == "" {
			continue
		}
		key := fmt.Sprintf("%d:%s", r.KpID, r.SkillCode)
		by[key] = append(by[key], r)
	}
	out := []Candidate{}
	for key, list := range by {
		sort.Slice(list, func(i, j int) bool {
			if list[i].OccurredAt.Equal(list[j].OccurredAt) {
				return list[i].AttemptID < list[j].AttemptID
			}
			return list[i].OccurredAt.Before(list[j].OccurredAt)
		})
		initial := []Evidence{}
		seen := map[string]bool{}
		for _, r := range list {
			instance := r.InstanceKey
			if instance == "" {
				continue
			}
			if seen[instance] {
				continue
			}
			seen[instance] = true
			initial = append(initial, r)
		}
		if len(initial) > 20 {
			initial = initial[len(initial)-20:]
		}

		accepted := map[string]bool{}
		for _, r := range initial {
			accepted[r.InstanceKey] = true
		}
		window := []Evidence{}
		for _, r := range list {
			if r.InstanceKey == "" || accepted[r.InstanceKey] {
				window = append(window, r)
			}
		}
		list = window
		if len(list) == 0 {
			continue
		}
		first := list[0]
		c := Candidate{Key: key, KpID: first.KpID, Title: first.Title, SubjectCode: first.SubjectCode, ModuleCode: first.ModuleCode, SkillCode: first.SkillCode, QuestionType: first.QuestionType, ReasonCode: "observed_wrong", Evidence: []CandidateEvidence{}, Mode: "mixed", RequestedCount: 3, PreferredDistractorKpIDs: []int64{}, ReviewEligible: true, ReviewBlockReasons: []string{}}
		confused := map[string]int{}
		hints := 0
		for _, r := range list {
			role := "target_observation"
			if !r.IsCorrect {
				role = "target_error"
				c.WrongCount++
				if r.OccurredAt.After(c.LastWrongAt) {
					c.LastWrongAt = r.OccurredAt
				}
			}
			if r.Assistance == "hinted" {
				hints++
				if r.IsCorrect {
					role = "assisted_completion"
				}
			}

			c.Evidence = append(c.Evidence, CandidateEvidence{r.AttemptID, role, r.Source})
			if !r.IsCorrect || r.Assistance == "hinted" {
				if !r.ReviewEligible {
					c.ReviewEligible = false
				}
				for _, reason := range r.ReviewBlockReasons {
					if !hasString(c.ReviewBlockReasons, reason) {
						c.ReviewBlockReasons = append(c.ReviewBlockReasons, reason)
					}
				}
			}
		}
		for _, r := range initial {
			if r.Assistance != "none" || earliest[fmt.Sprintf("%d:%s:%s", r.KpID, r.SkillCode, r.InstanceKey)].AttemptID != r.AttemptID {
				continue
			}
			c.IndependentInstanceCount++
			if !r.IsCorrect {
				c.WrongInitialCount++
				if r.SelectedSemanticID != "" {
					confused[r.SelectedSemanticID]++
				}
			}
		}
		if c.WrongCount == 0 && hints == 0 {
			continue
		}
		if c.WrongCount == 0 {
			c.ReasonCode = "assisted_completion"
			c.ReasonText = fmt.Sprintf("%s有 %d 次在提示后完成，可再独立试一试", Label(c.SubjectCode, c.SkillCode), hints)
		} else {
			c.ReasonText = fmt.Sprintf("%s已记录 %d 次错误，点开查看依据", Label(c.SubjectCode, c.SkillCode), c.WrongCount)
		}
		if c.IndependentInstanceCount >= 3 && c.WrongInitialCount >= 2 {
			c.ReasonCode = "repeated_skill_error"
			c.ReasonText = fmt.Sprintf("%s最近 %d 次独立练习，%d 次首答错误", Label(c.SubjectCode, c.SkillCode), c.IndependentInstanceCount, c.WrongInitialCount)
		}
		semanticKeys := []string{}
		for sem := range confused {
			semanticKeys = append(semanticKeys, sem)
		}
		sort.Strings(semanticKeys)
		for _, sem := range semanticKeys {
			if confused[sem] >= 2 {
				c.ReasonCode = "repeated_confusion"
				c.ReasonText = fmt.Sprintf("%s在 %d 次不同练习中选中同一个错误内容", Label(c.SubjectCode, c.SkillCode), confused[sem])
				if strings.HasPrefix(sem, "kp:") {
					if id, e := strconv.ParseInt(strings.TrimPrefix(sem, "kp:"), 10, 64); e == nil && id > 0 {
						c.PreferredDistractorKpIDs = append(c.PreferredDistractorKpIDs, id)
					}
				}
			}
		}
		if c.QuestionType == "" {
			c.QuestionType = c.SkillCode
		}
		out = append(out, c)
	}
	sortCandidates(out)
	return out
}
func hasString(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func (s *Service) Candidates(child int64, f Filter) (Page[Candidate], error) {
	// Review analysis retains its fixed recent-evidence scope.
	f.From, f.To, f.MasteredOn = "", "", ""
	out := Page[Candidate]{Items: []Candidate{}, StateReadAt: s.Now(), Coverage: Coverage{"partial", []string{"recent_30_days", "known_instances_only"}}}
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
	f.WrongOnly = false
	analysisFilter := f
	analysisFilter.Skill = ""
	rows, totals, e := s.analysisEvidence(child, analysisFilter, c.Max, c.AsOf)
	if e != nil {
		return out, e
	}
	// Analysis receives retries as facts. Analyze alone selects initial-instance evidence.
	if f.FollowUpState != "" {
		if e = s.followUps(rows, c.Max, c.AsOf); e != nil {
			return out, e
		}
		filtered := []Evidence{}
		for _, r := range rows {
			if matchesFollowUp(r, f.FollowUpState) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}
	items := Analyze(rows, c.AsOf)
	for i := range items {
		v := &items[i]
		t := totals[v.Key]
		v.WrongCount = t.Wrong
		v.LastWrongAt = t.LastWrong
		v.EvidenceHasMore = t.Observed > len(v.Evidence)
		if v.ReasonCode == "observed_wrong" {
			v.ReasonText = fmt.Sprintf("%s本次分析样本记录 %d 次错误", Label(v.SubjectCode, v.SkillCode), t.Wrong)
		}
		if v.ReasonCode == "assisted_completion" {
			v.ReasonText = fmt.Sprintf("%s本次分析样本有 %d 次提示后完成", Label(v.SubjectCode, v.SkillCode), t.Hinted)
		}
	}

	points, e := s.library(child, c.Max, c.AsOf)
	if e != nil {
		return out, e
	}
	for _, p := range points {
		if !p.HasMasteredAbility {
			continue
		}
		for _, sk := range p.Skills {
			if sk.MasteryStatus != "shaky" || !sk.Practiced {
				continue
			}
			strength := supportingStrength(p, sk.SkillCode, rows, c.AsOf)
			if len(strength) == 0 {
				continue
			}
			found := false
			for i := range items {
				if items[i].KpID == p.KpID && items[i].SkillCode == sk.SkillCode {
					found = true
				}
			}
			if !found {
				observations := []CandidateEvidence{}
				for _, r := range rows {
					if r.KpID == p.KpID && r.SkillCode == sk.SkillCode && !r.OccurredAt.Before(c.AsOf.AddDate(0, 0, -30)) {
						observations = append(observations, CandidateEvidence{r.AttemptID, "target_observation", r.Source})
					}
				}
				if len(observations) > 0 {
					items = append(items, Candidate{Key: fmt.Sprintf("%d:%s", p.KpID, sk.SkillCode), KpID: p.KpID, Title: p.Title, SubjectCode: p.SubjectCode, ModuleCode: p.ModuleCode, SkillCode: sk.SkillCode, QuestionType: sk.SkillCode, ReasonCode: "observed_wrong", Evidence: observations, ReviewEligible: false, ReviewBlockReasons: []string{"no_error_source"}, PreferredDistractorKpIDs: []int64{}, Mode: "mixed", RequestedCount: 3})
				}
			}
			for i := range items {
				c := &items[i]
				if c.KpID == p.KpID && c.SkillCode == sk.SkillCode && c.ReasonCode == "observed_wrong" {
					c.Evidence = append(c.Evidence, strength...)
					c.ReasonCode = "practiced_skill_gap"
					c.ReasonText = fmt.Sprintf("已有已掌握能力，%s有实际练习且仍待巩固", sk.Label)
				}
			}
		}
	}
	if f.Skill != "" {
		filtered := []Candidate{}
		for _, item := range items {
			if item.SkillCode == f.Skill {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	sortCandidates(items)

	if c.Offset > len(items) {
		return out, bad("invalid_cursor", "分页标识无效")
	}
	end := c.Offset + f.Limit
	if end > len(items) {
		end = len(items)
	}
	out.Items = items[c.Offset:end]
	out.HasMore = end < len(items)
	if out.HasMore {
		c.Offset = end
		out.NextCursor = encodeCursor(c)
	}
	return out, nil
}

func sortCandidates(out []Candidate) {
	priority := map[string]int{"repeated_confusion": 0, "repeated_skill_error": 1, "practiced_skill_gap": 2, "observed_wrong": 3, "assisted_completion": 4}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if priority[a.ReasonCode] != priority[b.ReasonCode] {
			return priority[a.ReasonCode] < priority[b.ReasonCode]
		}
		if !a.LastWrongAt.Equal(b.LastWrongAt) {
			return a.LastWrongAt.After(b.LastWrongAt)
		}
		return a.Key < b.Key
	})
}

// A mastery flag alone is not review evidence. Keep one recent, verified
// independent success from another currently mastered skill in the saved input.
func supportingStrength(p Point, weak string, rows []Evidence, asOf time.Time) []CandidateEvidence {
	strong := map[string]bool{}
	for _, sk := range p.Skills {
		if sk.SkillCode != weak && mastered(sk.MasteryStatus) {
			strong[sk.SkillCode] = true
		}
	}
	var best *Evidence
	for i := range rows {
		r := &rows[i]
		if r.KpID != p.KpID || !strong[r.SkillCode] || !r.IsCorrect || r.Assistance != "none" || r.InstanceKey == "" || r.OccurredAt.Before(asOf.AddDate(0, 0, -30)) || r.OccurredAt.After(asOf) {
			continue
		}
		if best == nil || r.OccurredAt.After(best.OccurredAt) || r.OccurredAt.Equal(best.OccurredAt) && r.AttemptID > best.AttemptID {
			best = r
		}
	}
	if best == nil {
		return nil
	}
	return []CandidateEvidence{{best.AttemptID, "supporting_strength", best.Source}}
}
