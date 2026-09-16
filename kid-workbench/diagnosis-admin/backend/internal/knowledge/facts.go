package knowledge

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
	"unicode/utf8"
)

type Service struct {
	cache                                          factCache
	DB                                             *gorm.DB
	Now                                            func() time.Time
	HasVersions, HasWriting, HasPinyin, HasScience bool
}

func New(db *gorm.DB) *Service {
	return &Service{DB: db, Now: func() time.Time { return time.Now().UTC() }, HasVersions: db.Migrator().HasTable("question_attempt_receipts"), HasWriting: db.Migrator().HasColumn("question_attempt_receipts", "evaluation_json"), HasPinyin: db.Migrator().HasTable("pinyin_answer_receipts"), HasScience: db.Migrator().HasTable("science_attempt_receipts")}
}
func (s *Service) child(id int64) error {
	var n int64
	if id < 1 {
		return bad("invalid_child", "孩子编号无效")
	}
	if err := s.DB.Table("children").Where("id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (s *Service) base(child int64) (*gorm.DB, string, string) {
	q := s.DB.Table("attempts a").Joins("JOIN knowledge_points kp ON kp.id=a.kp_id").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects s ON s.id=m.subject_id").Joins("LEFT JOIN questions q ON q.id=a.question_id").Where("a.child_id=? AND a.source='quiz' AND s.code<>'game'", child)
	skill := "COALESCE(q.code,'')"
	if s.HasVersions {
		q = q.Joins("LEFT JOIN question_attempt_receipts qr ON qr.attempt_id=a.id AND qr.child_id=a.child_id")
		skill = "COALESCE(qr.skill_code,q.code,'')"
	}
	if s.HasPinyin {
		q = q.Joins("LEFT JOIN pinyin_answer_receipts pr ON pr.attempt_id=a.id AND pr.child_id=a.child_id")
		if s.HasVersions {
			skill = "COALESCE(qr.skill_code,pr.skill_code,q.code,'')"
		} else {
			skill = "COALESCE(pr.skill_code,q.code,'')"
		}
	}
	// Independence is determined by the validated source adapter, never a receipt's presence.
	return q, skill, "'unknown'"
}

func apply(q *gorm.DB, f Filter, skill string) *gorm.DB {
	if f.From != "" {
		at, _ := parseDay(f.From)
		if q.Dialector.Name() == "sqlite" {
			q = q.Where("julianday(a.created_at)>=julianday(?)", at.UTC())
		} else {
			q = q.Where("a.created_at>=?", at.UTC())
		}
	}
	if f.To != "" {
		at, _ := parseDay(f.To)
		at = at.AddDate(0, 0, 1)
		if q.Dialector.Name() == "sqlite" {
			q = q.Where("julianday(a.created_at)<julianday(?)", at.UTC())
		} else {
			q = q.Where("a.created_at<?", at.UTC())
		}
	}
	if f.Subject != "" {
		q = q.Where("s.code=?", f.Subject)
	}
	if f.Module != "" {
		q = q.Where("m.code=?", f.Module)
	}
	if f.KpID > 0 {
		q = q.Where("kp.id=?", f.KpID)
	}
	if f.Q != "" {
		q = q.Where("LOWER(kp.title) LIKE ?", "%"+strings.ToLower(f.Q)+"%")
	}
	if f.Skill != "" {
		q = q.Where(skill+"=?", f.Skill)
	}
	if f.WrongOnly {
		q = q.Where("a.is_correct=?", false)
	}
	return q
}

type cursor struct {
	Child    int64     `json:"c"`
	Hash     string    `json:"h"`
	Max      int64     `json:"m"`
	AsOf     time.Time `json:"a"`
	At       time.Time `json:"t"`
	ID       int64     `json:"i"`
	Offset   int       `json:"o"`
	Revision string    `json:"r,omitempty"`
}

func filterHash(f Filter) string {
	f.Cursor = ""
	f.Limit = 0
	b, _ := json.Marshal(f)
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
func (s *Service) pageCursor(child int64, f Filter) (cursor, error) {
	c := cursor{Child: child, Hash: filterHash(f), AsOf: s.Now()}
	if f.Cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(f.Cursor)
		if e != nil || len(b) > 4096 {
			return c, bad("invalid_cursor", "分页标识无效")
		}
		if e = json.Unmarshal(b, &c); e != nil || c.Child != child || c.Hash != filterHash(f) || c.Max < 0 || c.Offset < 0 || c.ID < 0 || c.AsOf.IsZero() {
			return c, bad("invalid_cursor", "筛选条件已改变，请重新加载")
		}
		return c, nil
	}
	e := s.DB.Table("attempts").Select("COALESCE(MAX(id),0)").Where("child_id=?", child).Scan(&c.Max).Error
	return c, e
}
func encodeCursor(c cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}
func normalize(f *Filter) error {
	for _, v := range []string{f.From, f.To, f.MasteredOn} {
		if v != "" {
			if _, e := parseDay(v); e != nil {
				return e
			}
		}
	}
	if f.From != "" && f.To != "" && f.From > f.To {
		return bad("invalid_date", "起始日期不能晚于结束日期")
	}
	f.Q = strings.TrimSpace(f.Q)
	if utf8.RuneCountInString(f.Q) > 80 {
		return bad("invalid_query", "搜索内容不能超过80字")
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit < 1 || f.Limit > 100 {
		return bad("invalid_limit", "每页数量须在1至100之间")
	}
	return nil
}

func matchesFollowUp(r Evidence, state string) bool {
	switch state {
	case "", "all":
		return true
	case "needs_practice":
		return !r.IsCorrect && !r.LaterIndependentCorrect
	case "needs_attention":
		return r.FollowUpState != "mastered_later" && r.FollowUpState != "answered_correctly_later"
	case "assisted", "assisted_completion":
		return r.Assistance == "hinted" && r.IsCorrect
	default:
		return r.FollowUpState == state
	}
}
func (s *Service) Attempts(child int64, f Filter) (Page[Evidence], error) {
	p := Page[Evidence]{Items: []Evidence{}, StateReadAt: s.Now(), Coverage: Coverage{"partial", []string{"legacy_selection_missing"}}}
	if e := normalize(&f); e != nil {
		return p, e
	}
	if e := s.child(child); e != nil {
		return p, e
	}
	c, e := s.pageCursor(child, f)
	if e != nil {
		return p, e
	}
	p.EvidenceAsOf = c.AsOf
	queryFilter := f
	if f.FollowUpState == "assisted" || f.FollowUpState == "assisted_completion" {
		queryFilter.WrongOnly = false
	}
	q, skill := s.boundedBase(child, queryFilter, c.Max, c.AsOf)
	at, id := c.At, c.ID
	for {
		limit := f.Limit + 1
		if f.FollowUpState != "" && f.FollowUpState != "all" {
			limit = evidenceBatchSize
		}
		rows, err := s.evidenceBatch(q, skill, at, id, limit)
		if err != nil {
			return p, err
		}
		if len(rows) == 0 {
			break
		}
		if err = s.followUps(rows, c.Max, c.AsOf); err != nil {
			return p, err
		}
		for _, r := range rows {
			if matchesFollowUp(r, f.FollowUpState) {
				p.Items = append(p.Items, r)
				if len(p.Items) > f.Limit {
					break
				}
			}
		}
		if len(p.Items) > f.Limit || len(rows) < limit {
			break
		}
		last := rows[len(rows)-1]
		at = last.OccurredAt
		id = last.AttemptID
	}

	if len(p.Items) > f.Limit {
		p.HasMore = true
		p.Items = p.Items[:f.Limit]
		last := p.Items[len(p.Items)-1]
		c.At = last.OccurredAt
		c.ID = last.AttemptID
		p.NextCursor = encodeCursor(c)
	}
	return p, nil
}
func (s *Service) Attempt(child, id int64) (Evidence, error) {
	var row Evidence
	q, skill, _ := s.base(child)
	e := q.Where("a.id=?", id).Select("a.id AS attempt_id,a.child_id,a.kp_id,kp.title,s.code AS subject_code,s.name AS subject_name,m.code AS module_code,m.name AS module_name,a.created_at AS occurred_at,a.is_correct,a.cost_ms,a.client_id,a.question_id," + skill + " AS skill_code").Scan(&row).Error
	if e != nil {
		return row, e
	}
	if row.AttemptID == 0 {
		return row, gorm.ErrRecordNotFound
	}
	rows := []Evidence{row}
	e = s.enrich(rows)
	if e == nil {
		e = s.followUps(rows, -1, s.Now())
	}
	return rows[0], e
}

// AttemptsByIDs resolves a bounded set of real facts with the same source/assistance
// validation as attempt detail. Missing, foreign-child and non-quiz IDs are omitted.
// It intentionally does not load follow-up history or mastery states.
func (s *Service) AttemptsByIDs(child int64, ids []int64) ([]Evidence, error) {
	rows := []Evidence{}
	if e := s.child(child); e != nil {
		return nil, e
	}
	if len(ids) == 0 {
		return rows, nil
	}
	q, skill, _ := s.base(child)
	e := q.Where("a.id IN ?", ids).Select("a.id AS attempt_id,a.child_id,a.kp_id,kp.title,s.code AS subject_code,s.name AS subject_name,m.code AS module_code,m.name AS module_name,a.created_at AS occurred_at,a.is_correct,a.cost_ms,a.client_id,a.question_id," + skill + " AS skill_code").Order("a.created_at DESC,a.id DESC").Scan(&rows).Error
	if e != nil {
		return nil, e
	}
	if e = s.enrich(rows); e != nil {
		return nil, e
	}
	return rows, nil
}
