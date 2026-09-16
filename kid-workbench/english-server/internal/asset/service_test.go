package asset_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/english-server/internal/asset"
)

type reader struct{ err error }

func (r reader) Get(context.Context, string) ([]byte, error) { return []byte("ok"), r.err }
func (r reader) Ping(context.Context) error                  { return r.err }

func TestKeys(t *testing.T) {
	require.Equal(t, "english/glyphs/42.png", asset.GlyphKey(42))
	require.Equal(t, "english/senses/42.png", asset.SenseKey(42))
	require.Equal(t, "english/speech/42.mp3", asset.SpeechKey(42))
}

func TestErrorsAreMapped(t *testing.T) {
	_, err := asset.NewService(reader{err: asset.ErrObjectNotFound}).Sense(context.Background(), 42)
	require.ErrorIs(t, err, asset.ErrMissing)
	transport := errors.New("transport")
	_, err = asset.NewService(reader{err: transport}).Speech(context.Background(), 42)
	require.ErrorIs(t, err, asset.ErrUnavailable)
}
