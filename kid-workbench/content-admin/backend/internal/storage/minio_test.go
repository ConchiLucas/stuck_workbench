package storage

import (
	"context"
	"github.com/minio/minio-go/v7"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/configclient"
)

func TestNewFromConfigHonorsContainerStorageOverrides(t *testing.T) {
	t.Setenv("APP_MINIO_ENDPOINT", "minio:9000")
	t.Setenv("APP_MINIO_BUCKET", "study-assets")
	t.Setenv("APP_MINIO_BASE_PATH", "math-content")
	store, err := NewFromConfig(configclient.ObjectStorageConfiguration{
		Configured: true, Enabled: true, Endpoint: "config.example:9000",
		AccessKeyID: "key", SecretAccessKey: "secret", BucketName: "config-bucket", BasePath: "config-prefix",
	})
	require.NoError(t, err)
	require.Equal(t, "study-assets", store.bucket)
	require.Equal(t, "math-content", store.basePath)
}

func TestLimitedObjectReadDistinguishesMissingAndUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "Sat, 12 Sep 2026 00:00:00 GMT")
		switch path.Base(r.URL.Path) {
		case "missing":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(404)
			io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
		case "denied":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(403)
			io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)
		default:
			io.WriteString(w, strings.Repeat("x", 33))
		}
	}))
	defer server.Close()
	client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{Region: "us-east-1"})
	require.NoError(t, err)
	store := &ObjectStore{client: client, bucket: "test-bucket"}
	_, err = store.GetBytesLimited(context.Background(), "large", 16)
	require.ErrorIs(t, err, ErrObjectTooLarge)
	b, err := store.GetBytesLimited(context.Background(), "fits", 33)
	require.NoError(t, err)
	require.Len(t, b, 33)
	_, err = store.GetBytesLimited(context.Background(), "missing", 33)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = store.GetBytesLimited(context.Background(), "denied", 33)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotFound)
}
