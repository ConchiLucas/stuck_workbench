package reviewsuggestion

import (
	"context"
	"fmt"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Explicit opt-in, isolated schema in a disposable database. Never use the formal child database.
func TestPostgresConcurrentSaveCommandsAndRecovery(t *testing.T) {
	dsn := os.Getenv("TASK_REVIEW_TEST_DSN")
	if dsn == "" {
		t.Skip("set TASK_REVIEW_TEST_DSN to a disposable kid_knowledge_test_* database")
	}
	g, e := db.OpenPostgres(dsn)
	require.NoError(t, e)
	var actual string
	require.NoError(t, g.Raw("SELECT current_database()").Scan(&actual).Error)
	require.True(t, strings.HasPrefix(actual, "kid_knowledge_test_"), "refusing writes outside disposable database")
	schema := fmt.Sprintf("review_suggestion_test_%d", time.Now().UnixNano())
	require.NoError(t, g.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() {
		require.NoError(t, g.Exec("DROP SCHEMA "+schema+" CASCADE").Error)
		sqlDB, _ := g.DB()
		_ = sqlDB.Close()
	})
	isolated, e := db.OpenPostgres(dsn + " search_path=" + schema)
	require.NoError(t, e)
	sqlDB, e := isolated.DB()
	require.NoError(t, e)
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	s := fixtureDB(t, isolated)
	ctx := context.Background()
	results := make(chan Suggestion, 4)
	errs := make(chan error, 4)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); a, _, e := s.Save(ctx, 7, "concurrent", input()); results <- a; errs <- e }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	id := int64(0)
	for a := range results {
		if id == 0 {
			id = a.ID
		}
		require.Equal(t, id, a.ID)
	}
	errs = make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, e := s.Command(ctx, 7, id, "generate", "go", CommandInput{ExpectedRowVersion: 1})
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	var n int64
	require.NoError(t, isolated.Model(&Run{}).Count(&n).Error)
	require.EqualValues(t, 1, n)
	require.NoError(t, isolated.Model(&Partition{}).Where("suggestion_id=?", id).Updates(map[string]any{"state": "running", "lease_owner": "dead-worker", "lease_until": time.Now().Add(-time.Minute)}).Error)
	require.NoError(t, s.Tick(ctx))
	a, e := s.Get(ctx, 7, id)
	require.NoError(t, e)
	require.Equal(t, "blocked", *a.GenerationStatus)
	s, _, literacy := literacyFixtureService(t, s)
	suggestion, _, e := s.Save(ctx, 7, "literacy-cross-plan", literacy)
	require.NoError(t, e)
	_, _, e = s.Command(ctx, 7, suggestion.ID, "generate", "literacy-run", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	errs = make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- s.Tick(ctx) }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	suggestion, e = s.Get(ctx, 7, suggestion.ID)
	require.NoError(t, e)
	require.Equal(t, "succeeded", *suggestion.GenerationStatus)
	require.Equal(t, 2, suggestion.TaskCount)
	links, e := s.Tasks(ctx, 7, suggestion.ID)
	require.NoError(t, e)
	require.Len(t, links, 2)
	for _, link := range links {
		require.Equal(t, "draft", link.TaskStatus)
		require.NotNil(t, link.ActiveRevisionID)
		require.Equal(t, link.GeneratedRevisionID, *link.ActiveRevisionID)
	}

}
