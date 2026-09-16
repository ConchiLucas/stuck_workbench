package math_test

import (
	"context"
	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/math"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func TestPostgresDetailPublishing(t *testing.T) {
	name := os.Getenv("MATH_TEST_DB")
	if name == "" {
		t.Skip("requires isolated kid_math_verify_* database")
	}
	require.True(t, strings.HasPrefix(name, "kid_math_verify_"))
	require.Empty(t, os.Getenv("APP_DSN"))
	t.Setenv("APP_DB_NAME", name)
	d, e := db.OpenPostgres(db.DSNFromEnv())
	require.NoError(t, e)
	var actual string
	require.NoError(t, d.Raw("SELECT current_database()").Scan(&actual).Error)
	require.Equal(t, name, actual)
	s := math.NewService(d, nil, nil, nil, nil)
	require.NoError(t, s.InitializeDetails())
	ctx := context.Background()
	c, e := s.Details(ctx, true)
	require.NoError(t, e)
	require.Len(t, c.Items, 12)
	item := c.Items[0]
	item.Title = "PostgreSQL发布验证"
	saved, e := s.SaveDetail(ctx, item)
	require.NoError(t, e)
	_, e = s.SaveDetail(ctx, item)
	require.ErrorIs(t, e, math.ErrDetailConflict)
	pub, e := s.PublishDetail(ctx, item.ID, saved.Revision)
	require.NoError(t, e)
	require.Equal(t, saved.Revision, pub.PublishedRevision)
	require.NoError(t, s.InitializeDetails())
	c, e = s.Details(ctx, true)
	require.NoError(t, e)
	require.Equal(t, pub.Title, c.Items[0].Title)
}
