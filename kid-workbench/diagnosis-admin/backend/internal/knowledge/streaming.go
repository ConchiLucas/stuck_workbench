package knowledge

import (
	"gorm.io/gorm"
	"time"
)

const evidenceBatchSize = 200
const groupEvidenceLimit = 20

func evidenceSelect(skill string) string {
	return "a.id AS attempt_id,a.child_id,a.kp_id,kp.title,s.code AS subject_code,s.name AS subject_name,m.code AS module_code,m.name AS module_name,a.created_at AS occurred_at,a.is_correct,a.cost_ms,a.client_id,a.question_id," + skill + " AS skill_code"
}
func (s *Service) before(q *gorm.DB, at time.Time, id int64) *gorm.DB {
	if id == 0 {
		return q
	}
	if s.DB.Dialector.Name() == "sqlite" {
		return q.Where("(julianday(a.created_at)<julianday(?) OR (julianday(a.created_at)=julianday(?) AND a.id<?))", at, at, id)
	}
	return q.Where("(a.created_at<? OR (a.created_at=? AND a.id<?))", at, at, id)
}
func (s *Service) boundedBase(child int64, f Filter, max int64, asOf time.Time) (*gorm.DB, string) {
	q, skill, _ := s.base(child)
	q = apply(q, f, skill)
	q = q.Where("a.created_at<=?", asOf)
	if max >= 0 {
		q = q.Where("a.id<=?", max)
	}
	return q, skill
}
func (s *Service) evidenceBatch(q *gorm.DB, skill string, at time.Time, id int64, limit int) ([]Evidence, error) {
	if limit > evidenceBatchSize {
		limit = evidenceBatchSize
	}
	rows := []Evidence{}
	err := s.before(q.Session(&gorm.Session{}), at, id).Select(evidenceSelect(skill)).Order("a.created_at DESC,a.id DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if err = s.enrich(rows); err != nil {
		return nil, err
	}
	return rows, nil
}
func (s *Service) streamEvidence(q *gorm.DB, skill string, visit func([]Evidence) (bool, error)) error {
	var at time.Time
	var id int64
	for {
		rows, e := s.evidenceBatch(q, skill, at, id, evidenceBatchSize)
		if e != nil {
			return e
		}
		if len(rows) == 0 {
			return nil
		}
		more, e := visit(rows)
		if e != nil {
			return e
		}
		if !more || len(rows) < evidenceBatchSize {
			return nil
		}
		last := rows[len(rows)-1]
		at = last.OccurredAt
		id = last.AttemptID
	}
}

// instanceBase uses only exact receipt links. NULL means the old fact cannot
// establish an independent instance; SQL may still count it as a unique fact.
func (s *Service) instanceBase(child int64, f Filter, max int64, asOf time.Time) (*gorm.DB, string, string) {
	q, skill := s.boundedBase(child, f, max, asOf)
	instance := "NULL"
	if s.HasScience {
		q = q.Joins("LEFT JOIN science_attempt_receipts sr ON sr.child_id=a.child_id AND sr.client_id=a.client_id AND sr.kp_id=a.kp_id AND sr.question_id=a.question_id")
		instance = "CASE WHEN sr.item_id IS NOT NULL THEN 'item:' || CAST(sr.item_id AS TEXT) ELSE NULL END"
	}
	if s.HasPinyin {
		instance = "COALESCE('pinyin:' || pr.instance_id," + instance + ")"
	}
	if s.HasVersions {
		instance = "COALESCE('item:' || CAST(qr.plan_item_id AS TEXT)," + instance + ")"
	}
	return q, skill, instance
}
