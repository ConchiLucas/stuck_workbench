package knowledge

import (
	"context"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"testing"
	"time"
)

type rowMonitor struct {
	logger.Interface
	mu  sync.Mutex
	max int64
}

func (m *rowMonitor) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	_, n := fc()
	m.mu.Lock()
	if n > m.max {
		m.max = n
	}
	m.mu.Unlock()
}
func TestHistoryQueriesRemainBounded(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`WITH RECURSIVE n(x) AS (SELECT 10 UNION ALL SELECT x+1 FROM n WHERE x<1009) INSERT INTO attempts SELECT x,1,20,201,0,100,'quiz','load-'||x,'2026-09-08 12:00:00' FROM n`).Error)
	m := &rowMonitor{Interface: logger.Default}
	s := New(db)
	s.DB = s.DB.Session(&gorm.Session{Logger: m})
	s.Now = fixedNow
	p, e := s.Attempts(1, Filter{Limit: 20})
	require.NoError(t, e)
	require.Len(t, p.Items, 20)
	require.True(t, p.HasMore)
	require.LessOrEqual(t, m.max, int64(200), "no query may materialize full attempt history")
	m.max = 0
	sum, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 1003, sum.Stats.ObservedAttempts)
	require.LessOrEqual(t, m.max, int64(200))
	m.max = 0
	g, e := s.Groups(1, Filter{Subject: "english"})
	require.NoError(t, e)
	require.Len(t, g.Items, 1)
	require.Equal(t, 1002, g.Items[0].WrongCount)
	require.LessOrEqual(t, len(g.Items[0].Evidence), 20)
	require.LessOrEqual(t, m.max, int64(200))
}
