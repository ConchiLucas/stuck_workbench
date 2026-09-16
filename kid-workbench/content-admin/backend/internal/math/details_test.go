package math_test

import (
	"context"
	"strings"
	"testing"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/math"
	"github.com/conchi/study-learning/mathcontent"
	"github.com/stretchr/testify/require"
)

var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func TestDetailDraftPublicationAndConflict(t *testing.T) {
	s := setupMathDB(t)
	ctx := context.Background()
	require.NoError(t, s.InitializeDetails())
	c, e := s.Details(ctx, true)
	require.NoError(t, e)
	require.Len(t, c.Items, 12)
	d := c.Items[0]
	oldTitle := d.Title
	d.Title = "新的示例标题"
	saved, e := s.SaveDetail(ctx, d)
	require.NoError(t, e)
	require.Equal(t, 2, saved.Revision)
	_, e = s.SaveDetail(ctx, d)
	require.ErrorIs(t, e, math.ErrDetailConflict)
	published, e := s.Details(ctx, true)
	require.NoError(t, e)
	require.Equal(t, oldTitle, published.Items[0].Title)
	_, e = s.PublishDetail(ctx, d.ID, 1)
	require.ErrorIs(t, e, math.ErrDetailConflict)
	_, e = s.PublishDetail(ctx, d.ID, 2)
	require.NoError(t, e)
	require.NoError(t, s.InitializeDetails())
	published, e = s.Details(ctx, true)
	require.NoError(t, e)
	require.Equal(t, saved.Title, published.Items[0].Title)
}

func TestCannotPublishMissingImageObject(t *testing.T) {
	g, e := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, e)
	store := &questionStore{}
	s := math.NewService(g, store, nil, nil, nil)
	require.NoError(t, s.InitializeDetails())
	ctx := context.Background()
	c, e := s.Details(ctx, false)
	require.NoError(t, e)
	var d mathcontent.MathDetail
	for _, item := range c.Items {
		if item.ID == "addition-story" {
			d = item
			break
		}
	}
	require.Equal(t, "addition-story", d.ID)
	d.Example.ObjectImageURL = "/api/v1/math/detail-image/" + strings.Repeat("a", 64) + ".png"
	saved, e := s.SaveDetail(ctx, d)
	require.NoError(t, e)
	_, e = s.PublishDetail(ctx, d.ID, saved.Revision)
	require.Error(t, e)
	published, e := s.Details(ctx, true)
	require.NoError(t, e)
	for _, item := range published.Items {
		if item.ID == "addition-story" {
			require.Equal(t, 1, item.Revision)
			require.Empty(t, item.Example.ObjectImageURL)
		}
	}
}

func TestDetailImageUsesImmutableContentAddress(t *testing.T) {
	ctx := context.Background()
	g, e := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, e)
	store := &questionStore{}
	s := math.NewService(g, store, nil, nil, nil)
	require.NoError(t, s.InitializeDetails())
	url, e := s.PutDetailImage(ctx, tinyPNG)
	require.NoError(t, e)
	require.Contains(t, url, "/api/v1/math/detail-image/")
	require.True(t, strings.HasSuffix(url, ".png"))
	file := strings.TrimPrefix(url, "/api/v1/math/detail-image/")
	got, e := s.DetailImage(ctx, file)
	require.NoError(t, e)
	require.Equal(t, tinyPNG, got)
	c, e := s.Details(ctx, false)
	require.NoError(t, e)
	var d mathcontent.MathDetail
	for _, item := range c.Items {
		if item.ID == "addition-story" {
			d = item
			break
		}
	}
	d.Example.ObjectImageURL = url
	saved, e := s.SaveDetail(ctx, d)
	require.NoError(t, e)
	published, e := s.PublishDetail(ctx, saved.ID, saved.Revision)
	require.NoError(t, e)
	require.Equal(t, url, published.Example.ObjectImageURL)
}

func TestCannotPublishMissingAudioObject(t *testing.T) {
	s := setupMathDB(t)
	require.NoError(t, s.InitializeDetails())
	ctx := context.Background()
	c, e := s.Details(ctx, false)
	require.NoError(t, e)
	d := c.Items[0]
	d.Example.AudioURL = "/api/v1/math/detail-audio/" + strings.Repeat("a", 64) + ".mp3"
	saved, e := s.SaveDetail(ctx, d)
	require.NoError(t, e)
	_, e = s.PublishDetail(ctx, d.ID, saved.Revision)
	require.Error(t, e)
	published, e := s.Details(ctx, true)
	require.NoError(t, e)
	require.Equal(t, 1, published.Items[0].Revision)
}

func TestDetailAudioUsesImmutableContentAddress(t *testing.T) {
	ctx := context.Background()
	g, e := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, e)
	store := &questionStore{}
	speech := &questionSpeech{}
	s := math.NewService(g, store, nil, questionVoices{}, speech)
	require.NoError(t, s.InitializeDetails())
	d, e := s.GenerateDetailAudio(ctx, "shape-find", 1)
	require.NoError(t, e)
	require.Equal(t, 2, d.Revision)
	require.Contains(t, d.Example.AudioURL, "/api/v1/math/detail-audio/")
	_, e = s.GenerateDetailAudio(ctx, "shape-find", 1)
	require.ErrorIs(t, e, math.ErrDetailConflict)
	d.Example.Prompt = "听到：三角形"
	d.Example.Answer = "△"
	updated, e := s.SaveDetail(ctx, d)
	require.NoError(t, e)
	require.Empty(t, updated.Example.AudioURL, "changed prompt cannot keep the previous narration")
}
