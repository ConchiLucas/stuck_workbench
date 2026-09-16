package asset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type bucketStub struct {
	exists bool
	err    error
}

func (s bucketStub) BucketExists(context.Context, string) (bool, error) { return s.exists, s.err }

func TestBucketReadyRejectsMissingBucket(t *testing.T) {
	require.NoError(t, bucketReady(context.Background(), bucketStub{exists: true}, "study-assets"))
	require.ErrorIs(t, bucketReady(context.Background(), bucketStub{exists: false}, "study-assets"), ErrUnavailable)
}
