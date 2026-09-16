package knowledge

import (
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFactCacheScopesChildrenNewAttemptsAndCurrentMastery(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = fixedNow
	first, e := s.Summary(1)
	require.NoError(t, e)
	require.NoError(t, db.Exec(`INSERT INTO children(id,name) VALUES(2,'other');UPDATE mastery_skills SET status='learning' WHERE child_id=1 AND kp_id=10;INSERT INTO attempts VALUES(10,1,20,201,1,100,'quiz','new','2026-09-10 10:00:00')`).Error)
	next, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, first.Stats.ObservedAttempts+1, next.Stats.ObservedAttempts)
	require.Equal(t, 0, next.MasteredAbilityCount)
	other, e := s.Summary(2)
	require.NoError(t, e)
	require.Equal(t, 0, other.Stats.ObservedAttempts)
}
func TestFactCachePreservesHistoricalFenceAndWindowBoundary(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = fixedNow
	asof := fixedNow()
	before, e := s.aggregates(1, 3, asof)
	require.NoError(t, e)
	require.NoError(t, db.Exec(`INSERT INTO attempts VALUES(10,1,20,201,0,100,'quiz','new','2026-09-10 10:00:00')`).Error)
	after, e := s.aggregates(1, 3, asof)
	require.NoError(t, e)
	require.Equal(t, before, after)
	// Mutating a caller's result must never poison a cached copy.
	before[0].ObservedAttempts = 999
	after, e = s.aggregates(1, 3, asof)
	require.NoError(t, e)
	require.NotEqual(t, 999, after[0].ObservedAttempts)
	k1, e := s.factCacheKey("analysis", 1, 10, asof, Filter{})
	require.NoError(t, e)
	k2, e := s.factCacheKey("analysis", 1, 10, asof.Add(31*24*time.Hour), Filter{})
	require.NoError(t, e)
	require.NotEqual(t, k1, k2)
}
func TestFactCacheIsBounded(t *testing.T) {
	var c factCache
	for i := 0; i < cacheEntryLimit+5; i++ {
		c.put(string(rune('a'+i)), []int{i})
	}
	require.LessOrEqual(t, len(c.entries), cacheEntryLimit)
	var decoded []int
	require.True(t, c.get(string(rune('a'+cacheEntryLimit+4)), &decoded))
	require.Equal(t, []int{cacheEntryLimit + 4}, decoded)
}

func TestFactCacheInvalidatesWhenReceiptArrivesForExistingAttempt(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE subjects SET code='pinyin' WHERE id=2;UPDATE modules SET code='syllables' WHERE id=2;
 CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,public_snapshot TEXT,answer_option_id TEXT);
 CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT);
 INSERT INTO pinyin_quiz_instances VALUES('i',1,20,'{"kpId":20,"options":[{"id":"a"},{"id":"b"}]}','a')`).Error)
	s := New(db)
	s.Now = fixedNow
	before, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 0, before.Stats.IndependentAttempts)
	require.NoError(t, db.Exec(`INSERT INTO pinyin_answer_receipts VALUES(1,'a1','i',1,'blend','b')`).Error)
	after, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 1, after.Stats.IndependentAttempts)
}
func TestFactCacheDoesNotReuseFutureFence(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = fixedNow
	require.NoError(t, db.Exec("INSERT INTO attempts VALUES(10,1,20,201,0,100,'quiz','future',?)", fixedNow().Add(time.Minute)).Error)
	before, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 3, before.Stats.ObservedAttempts)
	s.Now = func() time.Time { return fixedNow().Add(2 * time.Minute) }
	after, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 4, after.Stats.ObservedAttempts)
}

func TestCachedFactsStillReadMasteryAtSameFence(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = fixedNow
	before, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 1, before.MasteredAbilityCount)
	page, e := s.Points(1, Filter{State: "all", Limit: 1})
	require.NoError(t, e)
	require.True(t, page.HasMore)
	require.NoError(t, db.Exec(`UPDATE mastery_skills SET status='learning' WHERE child_id=1 AND kp_id=10 AND skill_code='glyph_sense'`).Error)
	after, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 0, after.MasteredAbilityCount)
	_, e = s.Points(1, Filter{State: "all", Limit: 1, Cursor: page.NextCursor})
	require.Error(t, e)
	var fault *Fault
	require.ErrorAs(t, e, &fault)
	require.Equal(t, "list_changed", fault.Code)
}
