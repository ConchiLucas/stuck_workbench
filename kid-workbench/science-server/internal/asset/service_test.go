package asset_test

import (
	"context"
	"testing"

	"github.com/conchi/study-science/internal/asset"
	"github.com/conchi/study-science/internal/catalog"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type reader map[string][]byte

func (r reader) Get(_ context.Context, key string) ([]byte, error) {
	if data, ok := r[key]; ok {
		return data, nil
	}
	return nil, asset.ErrObjectNotFound
}

type scope bool

func (s scope) GetItem(context.Context, int64) (catalog.Item, error) {
	if !s {
		return catalog.Item{}, gorm.ErrRecordNotFound
	}
	return catalog.Item{KpID: 42, ContentVersion: 3, SenseObjectKey: asset.SenseVersionKey(42, 3)}, nil
}

func TestScienceAssetKeys(t *testing.T) {
	require.Equal(t, "science/senses/42.png", asset.SenseKey(42))
	require.Equal(t, "science/glyphs/42.png", asset.GlyphKey(42))
	require.Equal(t, "science/speech/42.mp3", asset.SpeechKey(42))
	require.Equal(t, "science/senses/42-v3.png", asset.SenseVersionKey(42, 3))
}

func TestAssetsRequirePublishedScienceItem(t *testing.T) {
	svc := asset.NewService(reader{asset.SenseVersionKey(42, 3): []byte("png")}, scope(false))
	_, _, err := svc.Sense(context.Background(), 42)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestPublishedAssetReadsItsContentVersion(t *testing.T) {
	svc := asset.NewService(reader{asset.SenseVersionKey(42, 3): []byte("v3")}, scope(true))
	data, version, err := svc.Sense(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, []byte("v3"), data)
	require.Equal(t, 3, version)
}
