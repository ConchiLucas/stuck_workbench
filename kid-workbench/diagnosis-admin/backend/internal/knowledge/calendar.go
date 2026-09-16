package knowledge

import "time"

type CalendarSubject struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Total    int    `json:"total"`
	Wrong    int    `json:"wrong"`
	Mastered int    `json:"mastered"`
}
type CalendarDay struct {
	Date     string            `json:"date"`
	Mastered int               `json:"mastered"`
	Total    int               `json:"total"`
	Wrong    int               `json:"wrong"`
	Subjects []CalendarSubject `json:"subjects"`
}
type Calendar struct {
	Timezone                string        `json:"timezone"`
	Today                   string        `json:"today"`
	Month                   string        `json:"month"`
	Days                    []CalendarDay `json:"days"`
	Trend                   []CalendarDay `json:"trend"`
	MasteryDateUnknownCount int           `json:"masteryDateUnknownCount"`
}

func (s *Service) Calendar(child int64, month string) (Calendar, error) {
	now := s.Now()
	today := now.In(shanghai).Format("2006-01-02")
	out := Calendar{Timezone: "Asia/Shanghai", Today: today, Month: month, Days: []CalendarDay{}, Trend: []CalendarDay{}}
	start, e := time.ParseInLocation("2006-01", month, shanghai)
	if e != nil || start.Format("2006-01") != month {
		return out, bad("invalid_month", "月份须为有效的 YYYY-MM")
	}
	if e = s.child(child); e != nil {
		return out, e
	}
	pts, e := s.library(child, -1, now)
	if e != nil {
		return out, e
	}
	subjects := []CalendarSubject{}
	indexes := map[string]int{}
	for _, p := range pts {
		if _, ok := indexes[p.SubjectCode]; !ok {
			indexes[p.SubjectCode] = len(subjects)
			subjects = append(subjects, CalendarSubject{Code: p.SubjectCode, Name: p.SubjectName})
		}
	}
	days := map[string]*CalendarDay{}
	get := func(date string) *CalendarDay {
		if days[date] == nil {
			days[date] = &CalendarDay{Date: date, Subjects: append([]CalendarSubject{}, subjects...)}
		}
		return days[date]
	}
	for at := start; at.Before(start.AddDate(0, 1, 0)); at = at.AddDate(0, 0, 1) {
		get(at.Format("2006-01-02"))
	}
	todayAt, _ := parseDay(today)
	for i := 6; i >= 0; i-- {
		get(todayAt.AddDate(0, 0, -i).Format("2006-01-02"))
	}
	for _, p := range pts {
		if p.FirstMasteredAt == nil {
			if p.MasteryCategory == "complete" {
				out.MasteryDateUnknownCount++
			}
			continue
		}
		if d := days[*p.FirstMasteredAt]; d != nil {
			d.Mastered++
			d.Subjects[indexes[p.SubjectCode]].Mastered++
		}
	}
	// Scan every real fact in the requested calendar/trend ranges, without the
	// evidence endpoint's pagination limit or loading frozen question payloads.
	low := start
	if t := todayAt.AddDate(0, 0, -6); t.Before(low) {
		low = t
	}
	high := start.AddDate(0, 1, 0)
	if todayAt.AddDate(0, 0, 1).After(high) {
		high = todayAt.AddDate(0, 0, 1)
	}
	f := Filter{From: low.Format("2006-01-02"), To: high.AddDate(0, 0, -1).Format("2006-01-02")}
	q, _ := s.boundedBase(child, f, -1, now)
	rows, e := q.Select("a.created_at,s.code,a.is_correct").Rows()
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var at time.Time
		var code string
		var correct bool
		if e = rows.Scan(&at, &code, &correct); e != nil {
			return out, e
		}
		if d := days[at.In(shanghai).Format("2006-01-02")]; d != nil {
			d.Total++
			i := indexes[code]
			d.Subjects[i].Total++
			if !correct {
				d.Wrong++
				d.Subjects[i].Wrong++
			}
		}
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	for at := start; at.Before(start.AddDate(0, 1, 0)); at = at.AddDate(0, 0, 1) {
		out.Days = append(out.Days, *get(at.Format("2006-01-02")))
	}
	for i := 6; i >= 0; i-- {
		out.Trend = append(out.Trend, *get(todayAt.AddDate(0, 0, -i).Format("2006-01-02")))
	}
	return out, nil
}
