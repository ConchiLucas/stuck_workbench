package knowledge

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
	"time"
)

const cacheEntryLimit = 8
const cacheValueLimit = 4 << 20
const cacheLifetime = 30 * time.Second

type factCacheEntry struct {
	data     []byte
	inserted time.Time
}
type factCache struct {
	mu      sync.Mutex
	entries map[string]factCacheEntry
}

func (c *factCache) get(key string, out any) bool {
	c.mu.Lock()
	entry, ok := c.entries[key]
	if ok && time.Since(entry.inserted) > cacheLifetime {
		delete(c.entries, key)
		ok = false
	}
	c.mu.Unlock()
	return ok && gob.NewDecoder(bytes.NewReader(entry.data)).Decode(out) == nil
}
func (c *factCache) put(key string, value any) {
	var b bytes.Buffer
	if gob.NewEncoder(&b).Encode(value) != nil || b.Len() > cacheValueLimit {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]factCacheEntry{}
	}
	if len(c.entries) >= cacheEntryLimit {
		oldest := ""
		var at time.Time
		for k, v := range c.entries {
			if oldest == "" || v.inserted.Before(at) {
				oldest = k
				at = v.inserted
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[key] = factCacheEntry{append([]byte(nil), b.Bytes()...), time.Now()}
}

// Attempts and their canonical snapshots are append-only evidence. Reuse their
// derived values only for the same child and eligible fact set, for at most 30s.
// Mastery, catalog membership, and follow-up state are never cached. A count of
// the sliding window detects boundary crossings without rounding evidenceAsOf.
// Receipt counts also detect a canonical receipt arriving after its attempt.
func (s *Service) factCacheKey(kind string, child, max int64, asOf time.Time, f Filter) (string, error) {
	q := s.DB.Table("attempts a").Where("a.child_id=? AND a.created_at<=?", child, asOf)
	if max >= 0 {
		q = q.Where("a.id<=?", max)
	}
	selectSQL := "COUNT(*) AS count,COALESCE(MAX(a.id),0) AS max_id,SUM(CASE WHEN a.created_at>=? THEN 1 ELSE 0 END) AS recent"
	if s.HasVersions {
		q = q.Joins("LEFT JOIN question_attempt_receipts cache_qr ON cache_qr.attempt_id=a.id AND cache_qr.child_id=a.child_id")
		q = q.Joins("LEFT JOIN study_plans cache_plan ON cache_plan.id=cache_qr.plan_id AND cache_plan.child_id=a.child_id")
		selectSQL += ",COUNT(cache_qr.id) AS versions,SUM(CASE WHEN cache_plan.status='done' THEN 1 ELSE 0 END) AS completed_plans"
	}
	if s.HasPinyin {
		q = q.Joins("LEFT JOIN pinyin_answer_receipts cache_pr ON cache_pr.attempt_id=a.id AND cache_pr.child_id=a.child_id")
		selectSQL += ",COUNT(cache_pr.instance_id) AS pinyin"
	}
	if s.HasScience {
		q = q.Joins("LEFT JOIN science_attempt_receipts cache_sr ON cache_sr.child_id=a.child_id AND cache_sr.client_id=a.client_id")
		selectSQL += ",COUNT(cache_sr.item_id) AS science"
	}
	var signature struct{ Count, MaxID, Recent, Versions, Pinyin, Science, CompletedPlans int64 }
	if e := q.Select(selectSQL, asOf.AddDate(0, 0, -30)).Scan(&signature).Error; e != nil {
		return "", e
	}
	return fmt.Sprintf("%s:%d:%d:%+v:%s", kind, child, max, signature, filterHash(f)), nil
}
