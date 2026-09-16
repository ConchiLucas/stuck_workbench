package knowledge

import "time"

type PointCounts struct {
	Complete    int `json:"complete"`
	Partial     int `json:"partial"`
	Weak        int `json:"weak"`
	Learning    int `json:"learning"`
	Unpracticed int `json:"unpracticed"`
	Unknown     int `json:"unknown"`
	ReviewDue   int `json:"reviewDue"`
}

func (c *PointCounts) add(p Point) {
	switch p.MasteryCategory {
	case "complete":
		c.Complete++
	case "partial":
		c.Partial++
	case "weak":
		c.Weak++
	case "learning":
		c.Learning++
	case "unknown":
		c.Unknown++
	default:
		c.Unpracticed++
	}
	if p.ReviewDue {
		c.ReviewDue++
	}
}

var shanghai = time.FixedZone("Asia/Shanghai", 8*60*60)

func parseDay(v string) (time.Time, error) {
	at, e := time.ParseInLocation("2006-01-02", v, shanghai)
	if e != nil || at.Format("2006-01-02") != v {
		return time.Time{}, bad("invalid_date", "日期须为有效的 YYYY-MM-DD")
	}
	return at, nil
}
func classifyPoint(p *Point, raw string, at *time.Time, now time.Time) {
	p.MasteryCategory = "unpracticed"
	done, weak, learning, unknown := 0, false, false, false
	if len(p.Skills) == 0 {
		switch p.MasteryStatus {
		case "mastered", "review_due":
			p.MasteryCategory = "complete"
		case "shaky":
			p.MasteryCategory = "weak"
		case "learning":
			p.MasteryCategory = "learning"
		case "not_started":
			if p.Stats.ObservedAttempts > 0 {
				p.MasteryCategory = "unknown"
			}
		default:
			p.MasteryCategory = "unknown"
		}
		p.ReviewDue = p.MasteryStatus == "review_due"
	} else {
		mapped := 0
		for _, sk := range p.Skills {
			mapped += sk.Stats.ObservedAttempts
		}
		for _, sk := range p.Skills {
			p.ReviewDue = p.ReviewDue || sk.MasteryStatus == "review_due"
			switch sk.MasteryStatus {
			case "mastered", "review_due":
				done++
			case "shaky":
				weak = true
			case "learning":
				learning = true
			case "not_started":
				unknown = unknown || sk.Practiced || (!sk.StateRecorded && (p.Stats.ObservedAttempts > mapped || (raw != "" && raw != "not_started" && noRecordedSkills(p.Skills))))
			default:
				unknown = true
			}
		}
		switch {
		case done == len(p.Skills):
			p.MasteryCategory = "complete"
		case done > 0:
			p.MasteryCategory = "partial"
		case weak:
			p.MasteryCategory = "weak"
		case learning:
			p.MasteryCategory = "learning"
		case unknown:
			p.MasteryCategory = "unknown"
		}
		// General rollup copies a representative skill timestamp, not full mastery.
		at = nil
		if len(p.Skills) == 1 {
			at = p.Skills[0].MasteredAt
		}
	}
	if at != nil && !at.IsZero() && !at.After(now) {
		v := at.In(shanghai).Format("2006-01-02")
		p.FirstMasteredAt = &v
	}
}
