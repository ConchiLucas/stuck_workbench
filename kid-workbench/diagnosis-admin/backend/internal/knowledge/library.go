package knowledge

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/mastery"
	"gorm.io/gorm"
	"strings"
	"time"
)

func Label(subject, code string) string {
	if subject == "pinyin" {
		if label := map[string]string{"listen": "听音选字母", "inword": "字中找拼音", "shape": "看形认读", "blend": "声韵拼读"}[code]; label != "" {
			return label
		}
	}
	if subject == "english" {
		if label := map[string]string{"listen": "听音选词", "picture": "看图选词", "build": "组句子", "type": "写单词", "read": "读一读"}[code]; label != "" {
			return label
		}
	}
	if subject == "phrase" {
		if label := map[string]string{"listen_zh": "听一听", "listen_en": "选句子", "scene": "什么时候说", "reply": "问与答"}[code]; label != "" {
			return label
		}
	}
	if subject == "chengyu" {
		if label := mastery.SkillDisplayName("chengyu", code); label != "" && label != code {
			return label
		}
	}
	if subject == "math" {
		if label := map[string]string{"calc": "算式计算", "story": "情境应用", "find": "听音找图形", "name": "看图认名称"}[code]; label != "" {
			return label
		}
	}
	if subject == "science" {
		if label := mastery.SkillDisplayName("science", code); label != "" && label != code {
			return label
		}
	}
	if subject == "poem" {
		if label := mastery.SkillDisplayName("poem", code); label != "" && label != code {
			return label
		}
	}
	if subject == "logic" {
		if label := mastery.SkillDisplayName("logic", code); label != "" && label != code {
			return label
		}
	}
	names := map[string]string{"glyph_sense": "看字选义", "sense_char": "看义选字", "write_char": "听音写字", "listen_glyph": "听音选字（历史）", "inword": "听例字选音", "shape": "辨认字形", "blend": "音节拼读", "picture": "看图选词", "build": "组句子", "type": "写单词", "read": "读一读", "calc": "算式", "story": "应用题", "find": "找图形", "name": "认图形名称", "recognize": "科普辨认"}
	if code == "listen" {
		if subject == "english" {
			return "听音选词"
		}
		return "听音选字母"
	}
	if v := names[code]; v != "" {
		return v
	}
	if code != "" {
		return "历史题型"
	}
	return "题型未记录"
}
func effective(st string, due *time.Time, now time.Time) string {
	if st == "" {
		return "not_started"
	}
	if st == "mastered" && due != nil && due.Before(now) {
		return "review_due"
	}
	return st
}
func mastered(st string) bool { return st == "mastered" || st == "review_due" }
func addStats(a *Stats, b Stats) {
	a.ObservedAttempts += b.ObservedAttempts
	a.ObservedCorrect += b.ObservedCorrect
	a.IndependentAttempts += b.IndependentAttempts
	a.IndependentCorrect += b.IndependentCorrect
	a.AssistedAttempts += b.AssistedAttempts
	a.UnknownAssistanceAttempts += b.UnknownAssistanceAttempts
	a.finish()
}

type aggregate struct {
	KpID      int64
	SkillCode string
	Stats     `gorm:"embedded"`
}

func (s *Service) aggregates(child int64, max int64, asOf time.Time) ([]aggregate, error) {
	key, e := s.factCacheKey("aggregate", child, max, asOf, Filter{})
	if e != nil {
		return nil, e
	}
	var cached []aggregate
	if s.cache.get(key, &cached) {
		return cached, nil
	}
	rows, e := s.readAggregates(child, max, asOf)
	if e == nil {
		s.cache.put(key, rows)
	}
	return rows, e
}
func (s *Service) readAggregates(child int64, max int64, asOf time.Time) ([]aggregate, error) {
	q, skill := s.boundedBase(child, Filter{}, max, asOf)
	rows := []aggregate{}
	if e := q.Select("a.kp_id," + skill + " AS skill_code,COUNT(*) AS observed_attempts,SUM(CASE WHEN a.is_correct THEN 1 ELSE 0 END) AS observed_correct,COUNT(*) AS unknown_assistance_attempts").Group("a.kp_id," + skill).Scan(&rows).Error; e != nil {
		return nil, e
	}
	by := map[string]*aggregate{}
	for i := range rows {
		a := &rows[i]
		by[fmt.Sprintf("%d:%s", a.KpID, a.SkillCode)] = a
	}
	// Only receipt-backed facts need source JSON validation. Legacy facts stay unknown.
	receipt := ""
	if s.HasVersions {
		receipt = "qr.id IS NOT NULL"
	}
	if s.HasPinyin {
		if receipt != "" {
			receipt += " OR "
		}
		receipt += "pr.instance_id IS NOT NULL"
	}
	if receipt != "" {
		q, skill = s.boundedBase(child, Filter{}, max, asOf)
		e := s.streamEvidence(q.Where("("+receipt+")"), skill, func(batch []Evidence) (bool, error) {
			for _, r := range batch {
				a := by[fmt.Sprintf("%d:%s", r.KpID, r.SkillCode)]
				if a == nil {
					continue
				}
				switch r.Assistance {
				case "none":
					a.UnknownAssistanceAttempts--
					a.IndependentAttempts++
					if r.IsCorrect {
						a.IndependentCorrect++
					}
				case "hinted":
					a.UnknownAssistanceAttempts--
					a.AssistedAttempts++
				}
			}
			return true, nil
		})
		if e != nil {
			return nil, e
		}
	}
	return rows, nil
}
func (s *Service) library(child int64, max int64, asOf time.Time) ([]Point, error) {
	if e := s.child(child); e != nil {
		return nil, e
	}
	type raw struct {
		KpID                                                            int64
		Title, SubjectCode, SubjectName, ModuleCode, ModuleName, Status string
		DueAt, MasteredAt                                               *time.Time
		OrderNo                                                         int
	}
	rows := []raw{}
	err := s.DB.Table("knowledge_points kp").Select("kp.id AS kp_id,kp.title,s.code AS subject_code,s.name AS subject_name,m.code AS module_code,m.name AS module_name,kp.order_no,ms.status,ms.due_at,ms.mastered_at").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects s ON s.id=m.subject_id").Joins("LEFT JOIN mastery_states ms ON ms.kp_id=kp.id AND ms.child_id=?", child).Where("s.code<>'game'").Order("s.order_no,m.order_no,kp.order_no,kp.id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	aggs, err := s.aggregates(child, max, asOf)
	if err != nil {
		return nil, err
	}
	byKP := map[int64]map[string]Stats{}
	for _, a := range aggs {
		if byKP[a.KpID] == nil {
			byKP[a.KpID] = map[string]Stats{}
		}
		a.Stats.finish()
		byKP[a.KpID][a.SkillCode] = a.Stats
	}
	type state struct {
		KpID              int64
		SkillCode, Status string
		DueAt, MasteredAt *time.Time
	}
	states := []state{}
	if err = s.DB.Table("mastery_skills").Where("child_id=?", child).Scan(&states).Error; err != nil {
		return nil, err
	}
	stateMap := map[string]state{}
	for _, sk := range states {
		stateMap[fmt.Sprintf("%d:%s", sk.KpID, sk.SkillCode)] = sk
	}
	milestones := map[int64]time.Time{}
	if s.DB.Migrator().HasTable("pinyin_mastery_milestones") {
		var dates []struct {
			KpID             int64
			FirstCompletedAt time.Time
		}
		if e := s.DB.Table("pinyin_mastery_milestones").Where("child_id=? AND rule_version=2", child).Scan(&dates).Error; e != nil {
			return nil, e
		}
		for _, d := range dates {
			milestones[d.KpID] = d.FirstCompletedAt
		}
	}
	out := []Point{}
	for _, r := range rows {
		p := Point{KpID: r.KpID, Title: r.Title, SubjectCode: r.SubjectCode, SubjectName: r.SubjectName, ModuleCode: r.ModuleCode, ModuleName: r.ModuleName, OrderNo: r.OrderNo, MasteryStatus: effective(r.Status, r.DueAt, s.Now()), DueAt: r.DueAt, Skills: []Skill{}, Coverage: Coverage{"partial", []string{"historical_evidence_may_be_incomplete"}}}
		for _, a := range byKP[p.KpID] {
			addStats(&p.Stats, a)
		}
		p.WrongCount = p.Stats.ObservedAttempts - p.Stats.ObservedCorrect
		for _, code := range mastery.SkillsFor(p.SubjectCode, p.ModuleCode) {
			sk := stateMap[fmt.Sprintf("%d:%s", p.KpID, code)]
			stats := byKP[p.KpID][code]
			stats.finish()
			v := Skill{MasteredAt: sk.MasteredAt, StateRecorded: sk.Status != "", SkillCode: code, Label: Label(p.SubjectCode, code), MasteryStatus: effective(sk.Status, sk.DueAt, s.Now()), DueAt: sk.DueAt, Stats: stats, Practiced: stats.ObservedAttempts > 0, EvidenceCoverage: "partial"}
			if stats.ObservedAttempts == 0 {
				v.EvidenceCoverage = "missing"
			}
			p.Skills = append(p.Skills, v)
			if mastered(v.MasteryStatus) {
				p.HasMasteredAbility = true
			}
		}
		if len(p.Skills) == 0 {
			p.HasMasteredAbility = mastered(p.MasteryStatus)
		}
		classifyPoint(&p, r.Status, r.MasteredAt, s.Now())
		if at, ok := milestones[p.KpID]; ok && p.SubjectCode == "pinyin" && !at.IsZero() && !at.After(s.Now()) {
			v := at.In(shanghai).Format("2006-01-02")
			p.FirstMasteredAt = &v
		}
		out = append(out, p)
	}
	return out, nil
}
func (s *Service) Summary(child int64) (Summary, error) {
	out := Summary{Subjects: []Subject{}, Coverage: Coverage{"partial", []string{"only_recorded_practice"}}}
	if e := s.DB.Table("children").Where("id=?", child).Scan(&out.Child).Error; e != nil {
		return out, e
	}
	pts, e := s.library(child, -1, s.Now())
	if e != nil {
		return out, e
	}
	idx := map[string]int{}
	for _, p := range pts {
		i, ok := idx[p.SubjectCode]
		if !ok {
			i = len(out.Subjects)
			idx[p.SubjectCode] = i
			out.Subjects = append(out.Subjects, Subject{Code: p.SubjectCode, Name: p.SubjectName, Coverage: Coverage{"partial", []string{"only_recorded_practice"}}})
		}
		sub := &out.Subjects[i]
		sub.Total++
		out.TotalCount++
		sub.PointCounts.add(p)
		out.PointCounts.add(p)
		sub.AttemptsCount += p.Stats.ObservedAttempts
		if p.Stats.ObservedAttempts > 0 {
			sub.PracticedCount++
			if len(p.Skills) == 0 {
				sub.UnmappedPointCount++
			}
			mappedAttempts := 0
			for _, sk := range p.Skills {
				mappedAttempts += sk.Stats.ObservedAttempts
			}
			unattributedPractice := p.Stats.ObservedAttempts > mappedAttempts
			for _, sk := range p.Skills {
				switch sk.MasteryStatus {
				case "mastered":
					sub.Abilities.Mastered++
				case "learning":
					sub.Abilities.Learning++
				case "shaky", "review_due":
					sub.Abilities.Shaky++
				case "not_started":
					// Unattributed historical attempts may belong to any missing
					// skill. Preserve explicit states, but do not invent "unpracticed".
					legacyStateOnly := p.MasteryStatus != "not_started" && noRecordedSkills(p.Skills)
					if sk.Practiced || (!sk.StateRecorded && (unattributedPractice || legacyStateOnly)) {
						sub.Abilities.Unknown++
					} else {
						sub.Abilities.Unpracticed++
					}
				default:
					sub.Abilities.Unknown++
				}
			}
			out.PracticedCount++
		}
		if p.HasMasteredAbility {
			sub.MasteredAbilityCount++
			out.MasteredAbilityCount++
		}
		if p.WrongCount > 0 {
			sub.WrongPointCount++
			out.WrongPointCount++
		}
		sub.WrongCount += p.WrongCount
		out.WrongCount += p.WrongCount
		addStats(&out.Stats, p.Stats)
	}
	return out, nil
}
func (s *Service) Points(child int64, f Filter) (Page[Point], error) {
	out := Page[Point]{Items: []Point{}, StateReadAt: s.Now(), Coverage: Coverage{"partial", []string{"only_recorded_practice"}}}
	if e := normalize(&f); e != nil {
		return out, e
	}
	c, e := s.pageCursor(child, f)
	if e != nil {
		return out, e
	}
	pts, e := s.library(child, c.Max, c.AsOf)
	if e != nil {
		return out, e
	}
	out.EvidenceAsOf = c.AsOf
	// Directory and effective mastery identity prevent silent membership changes between pages.
	var revision []string
	for _, p := range pts {
		date := ""
		if p.FirstMasteredAt != nil {
			date = *p.FirstMasteredAt
		}
		revision = append(revision, fmt.Sprintf("%s:%t:%s", p.MasteryCategory, p.ReviewDue, date))
		revision = append(revision, fmt.Sprintf("%d:%d:%s:%s:%s", p.KpID, p.OrderNo, p.ModuleCode, p.SubjectCode, p.MasteryStatus))
		for _, sk := range p.Skills {
			revision = append(revision, sk.SkillCode+":"+sk.MasteryStatus)
		}
	}
	b, _ := json.Marshal(revision)
	hash := fmt.Sprintf("%x", sha256.Sum256(b))
	if c.Revision != "" && c.Revision != hash {
		return out, &Fault{409, "list_changed", "学习记录已更新，请刷新列表"}
	}
	c.Revision = hash
	selected := []Point{}
	for _, p := range pts {
		if f.Subject != "" && p.SubjectCode != f.Subject || f.Module != "" && p.ModuleCode != f.Module || f.Q != "" && !containsFold(p.Title, f.Q) {
			continue
		}
		if f.Skill != "" {
			found := false
			for _, sk := range p.Skills {
				found = found || sk.SkillCode == f.Skill
			}
			if !found {
				continue
			}
		}
		switch f.State {
		case "mastered":
			if !p.HasMasteredAbility {
				continue
			}
		case "complete", "partial", "weak", "learning", "unknown", "unpracticed":
			if p.MasteryCategory != f.State {
				continue
			}
		case "review_due":
			if !p.ReviewDue {
				continue
			}
		case "all":
		default:
			if p.Stats.ObservedAttempts == 0 && f.MasteredOn == "" {
				continue
			}
		}
		if f.MasteredOn != "" && (p.FirstMasteredAt == nil || *p.FirstMasteredAt != f.MasteredOn) {
			continue
		}
		selected = append(selected, p)
	}
	if c.Offset > len(selected) {
		return out, bad("invalid_cursor", "分页标识无效")
	}
	end := c.Offset + f.Limit
	if end > len(selected) {
		end = len(selected)
	}
	out.Items = selected[c.Offset:end]
	out.HasMore = end < len(selected)
	if out.HasMore {
		c.Offset = end
		out.NextCursor = encodeCursor(c)
	}
	return out, nil
}
func (s *Service) Point(child, kp int64) (Point, error) {
	pts, e := s.library(child, -1, s.Now())
	if e != nil {
		return Point{}, e
	}
	for _, p := range pts {
		if p.KpID == kp {
			return p, nil
		}
	}
	return Point{}, gorm.ErrRecordNotFound
}
func containsFold(a, b string) bool { return strings.Contains(strings.ToLower(a), strings.ToLower(b)) }

func noRecordedSkills(skills []Skill) bool {
	for _, sk := range skills {
		if sk.StateRecorded {
			return false
		}
	}
	return true
}
