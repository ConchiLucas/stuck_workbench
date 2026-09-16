package asset_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/literacy-server/internal/asset"
)

type missingReader struct{}

func (missingReader) Get(context.Context, string) ([]byte, error) {
	return nil, asset.ErrObjectNotFound
}

func TestKeys(t *testing.T) {
	require.Equal(t, "literacy/glyphs/42.png", asset.GlyphKey(42))
	require.Equal(t, "literacy/senses/42.png", asset.SenseKey(42))
	require.Equal(t, "literacy/speech/42.mp3", asset.SpeechKey(42))
}

func TestMissingObjectMapsToErrMissing(t *testing.T) {
	service := asset.NewService(missingReader{})
	_, err := service.Glyph(context.Background(), 42)
	require.True(t, errors.Is(err, asset.ErrMissing))
	_, err = service.Sense(context.Background(), 42)
	require.True(t, errors.Is(err, asset.ErrMissing))
	_, err = service.Speech(context.Background(), 42)
	require.True(t, errors.Is(err, asset.ErrMissing))
}
