package knowledge

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This opt-in test refuses the live study_workbench database and creates/drops
// only its own uniquely named schema in an explicitly named isolated test DB.
func TestPostgres100kKnowledgeReadBounds(t *testing.T) {
	dsn := os.Getenv("KNOWLEDGE_TEST_DSN")
	if dsn == "" {
		t.Skip("set KNOWLEDGE_TEST_DSN to an isolated kid_knowledge_test database")
	}
	cfg, e := pgx.ParseConfig(dsn)
	require.NoError(t, e)
	require.True(t, strings.HasPrefix(cfg.Database, "kid_knowledge_test"), "refusing non-isolated database")
	admin, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	schema := fmt.Sprintf("knowledge_perf_%d", time.Now().UnixNano())
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() {
		require.NoError(t, admin.Exec("DROP SCHEMA "+schema+" CASCADE").Error)
		db, _ := admin.DB()
		db.Close()
	})
	scoped := dsn + " search_path=" + schema
	if strings.Contains(dsn, "://") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		scoped = dsn + sep + "search_path=" + schema
	}
	db, e := gorm.Open(postgres.Open(scoped), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	sqlDB, e := db.DB()
	require.NoError(t, e)
	sqlDB.SetMaxOpenConns(2)
	t.Cleanup(func() { sqlDB.Close() })
	ddl := strings.Split(testdb.SQL, "INSERT INTO children")[0]
	ddl = strings.ReplaceAll(ddl, "DATETIME", "TIMESTAMPTZ")
	ddl = strings.ReplaceAll(ddl, "is_correct INTEGER", "is_correct BOOLEAN")
	require.NoError(t, db.Exec(ddl).Error)
	require.NoError(t, db.Exec(`INSERT INTO children(id,name) VALUES(1,'isolated perf child');
 INSERT INTO subjects(id,code,name,order_no) VALUES(1,'pinyin','pinyin',1);INSERT INTO modules(id,subject_id,code,name,order_no) VALUES(1,1,'syllables','syllables',1);
 INSERT INTO knowledge_points(id,module_id,code,title,order_no) SELECT i,1,'kp-'||i,'syllable-'||i,i FROM generate_series(1,100)i;
 CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,public_snapshot TEXT,answer_option_id TEXT);
 CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER UNIQUE,skill_code TEXT,selected_option_id TEXT);
 INSERT INTO attempts(id,child_id,kp_id,is_correct,cost_ms,source,client_id,created_at) SELECT i,1,1+((i-1)%100),i%3<>0,100,'quiz','perf-'||i,TIMESTAMPTZ '2026-09-12 11:59:59+00' - i * INTERVAL '1 second' FROM generate_series(1,100000)i;
 INSERT INTO pinyin_quiz_instances SELECT 'instance-'||id,child_id,kp_id,json_build_object('kpId',kp_id,'stem','choose','options',json_build_array(json_build_object('id','a','label','a'),json_build_object('id','b','label','b')))::text,'a' FROM attempts;
 INSERT INTO pinyin_answer_receipts SELECT child_id,client_id,'instance-'||id,id,'blend',CASE WHEN is_correct THEN 'a' ELSE 'b' END FROM attempts;
 CREATE INDEX ON attempts(child_id,created_at DESC,id DESC);CREATE INDEX ON attempts(child_id,kp_id,created_at DESC,id DESC);CREATE INDEX ON pinyin_answer_receipts(child_id,attempt_id);ANALYZE;`).Error)
	s := New(db)
	s.Now = fixedNow
	m := &rowMonitor{Interface: logger.Default}
	s.DB = s.DB.Session(&gorm.Session{Logger: m})
	measure := func(name string, run func()) {
		t.Helper()
		m.max = 0
		started := time.Now()
		run()
		elapsed := time.Since(started)
		t.Logf("%s: %s; max returned SQL rows=%d", name, elapsed, m.max)
		require.LessOrEqual(t, m.max, int64(200))
		require.Less(t, elapsed, 15*time.Second)
	}
	var page Page[Evidence]
	measure("Attempts first page", func() {
		var err error
		page, err = s.Attempts(1, Filter{Limit: 20})
		require.NoError(t, err)
		require.Len(t, page.Items, 20)
		require.True(t, page.HasMore)
	})
	measure("Attempts next page", func() {
		next, err := s.Attempts(1, Filter{Limit: 20, Cursor: page.NextCursor})
		require.NoError(t, err)
		require.Len(t, next.Items, 20)
		require.NotEqual(t, page.Items[0].AttemptID, next.Items[0].AttemptID)
	})
	measure("Summary 100k validated receipts", func() {
		sum, err := s.Summary(1)
		require.NoError(t, err)
		require.Equal(t, 100000, sum.Stats.ObservedAttempts)
		require.Equal(t, 100000, sum.Stats.IndependentAttempts)
		require.Equal(t, 33333, sum.WrongCount)
	})
	measure("Groups", func() {
		g, err := s.Groups(1, Filter{Limit: 20})
		require.NoError(t, err)
		require.Len(t, g.Items, 20)
		require.True(t, g.HasMore)
		for _, v := range g.Items {
			require.LessOrEqual(t, len(v.Evidence), 20)
			require.True(t, v.EvidenceHasMore)
		}
	})
	measure("Candidates", func() {
		g, err := s.Candidates(1, Filter{Limit: 20})
		require.NoError(t, err)
		require.Len(t, g.Items, 20)
		require.True(t, g.HasMore)
		for _, v := range g.Items {
			require.LessOrEqual(t, len(v.Evidence), 42)
			require.LessOrEqual(t, v.IndependentInstanceCount, 20)
		}
	})
	wallStart := time.Now()
	s.Now = func() time.Time { return fixedNow().Add(time.Since(wallStart)) }
	hot := func(name string, limit time.Duration, run func()) {
		t.Helper()
		run()
		samples := make([]time.Duration, 20)
		for i := range samples {
			started := time.Now()
			run()
			samples[i] = time.Since(started)
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		p95 := samples[18]
		t.Logf("%s: 20 hot calls p95=%s max=%s", name, p95, samples[19])
		require.LessOrEqual(t, p95, limit)
	}
	hot("Summary", 500*time.Millisecond, func() {
		sum, err := s.Summary(1)
		require.NoError(t, err)
		require.Equal(t, 100000, sum.Stats.IndependentAttempts)
	})
	hot("Candidates", 2*time.Second, func() {
		page, err := s.Candidates(1, Filter{Limit: 20})
		require.NoError(t, err)
		require.Len(t, page.Items, 20)
	})

}
