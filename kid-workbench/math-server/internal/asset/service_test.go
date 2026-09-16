package asset_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/math-server/internal/asset"
)

type fakeReader struct {
	data []byte
	err  error
	key  string
}

func (r *fakeReader) Get(_ context.Context, key string) ([]byte, error) {
	r.key = key
	return r.data, r.err
}

func TestQuestionAudioReadsSnapshotKey(t *testing.T) {
	reader := &fakeReader{data: []byte("mp3")}
	service := asset.NewService(reader)
	got, err := service.QuestionAudio(context.Background(), "math/questions/42.mp3")
	require.NoError(t, err)
	require.Equal(t, []byte("mp3"), got)
	require.Equal(t, "math/questions/42.mp3", reader.key)
}

func TestQuestionAudioRejectsKeysOutsideMathQuestions(t *testing.T) {
	service := asset.NewService(&fakeReader{})
	for _, key := range []string{"", "math/speech/42.mp3", "../math/questions/42.mp3", "math/questions/42.png"} {
		_, err := service.QuestionAudio(context.Background(), key)
		require.ErrorIs(t, err, asset.ErrInvalidKey)
	}
}

func TestQuestionAudioMapsMissingObjectToUnavailable(t *testing.T) {
	service := asset.NewService(&fakeReader{err: asset.ErrObjectNotFound})
	_, err := service.QuestionAudio(context.Background(), "math/questions/42.mp3")
	require.ErrorIs(t, err, asset.ErrUnavailable)
	require.False(t, errors.Is(err, asset.ErrInvalidKey))
}
