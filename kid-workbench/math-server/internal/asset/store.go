package asset

import (
	"context"
	"io"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/conchi/math-server/internal/config"
)

type Store struct {
	client   *minio.Client
	bucket   string
	basePath string
}

func NewStore(cfg config.ObjectStorage) (*Store, error) {
	endpoint := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(cfg.Endpoint), "http://"), "https://")
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), Secure: cfg.UseTLS,
	})
	if err != nil {
		return nil, err
	}
	return &Store{client: client, bucket: cfg.Bucket, basePath: strings.Trim(cfg.BasePath, "/")}, nil
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	if s.basePath != "" {
		key = path.Join(s.basePath, key)
	}
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, classify(err)
	}
	defer object.Close()
	data, err := io.ReadAll(object)
	return data, classify(err)
}

func (s *Store) Ready(ctx context.Context) error {
	return bucketReady(ctx, s.client, s.bucket)
}

type bucketChecker interface {
	BucketExists(context.Context, string) (bool, error)
}

func bucketReady(ctx context.Context, checker bucketChecker, bucket string) error {
	exists, err := checker.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUnavailable
	}
	return nil
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	response := minio.ToErrorResponse(err)
	if response.Code == "NoSuchKey" || response.Code == "NoSuchObject" || response.StatusCode == 404 {
		return ErrObjectNotFound
	}
	return err
}
