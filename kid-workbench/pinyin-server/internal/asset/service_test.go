package asset_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/pinyin-server/internal/asset"
)

type missingReader struct{}

func (missingReader) Get(context.Context, string) ([]byte, error) {
	return nil, asset.ErrObjectNotFound
}

func TestKeys(t *testing.T) {
	require.Equal(t, "pinyin/glyphs/42.png", asset.GlyphKey(42))
	require.Equal(t, "pinyin/speech/42/solo.mp3", asset.SpeechKey(42, "solo"))
	require.Equal(t, "pinyin/speech/42/word.mp3", asset.SpeechKey(42, "word"))
	require.Equal(t, "pinyin/speech/42/word-1.mp3", asset.SpeechKey(42, "word-1"))
	require.True(t, asset.ValidSpeechKind("word-1"))
	require.False(t, asset.ValidSpeechKind("word-9"))
}

func TestMissingObjectMapsToErrMissing(t *testing.T) {
	service := asset.NewService(missingReader{})
	_, err := service.Glyph(context.Background(), 42)
	require.True(t, errors.Is(err, asset.ErrMissing))
}
