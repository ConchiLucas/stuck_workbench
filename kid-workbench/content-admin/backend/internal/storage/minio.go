package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/conchi/study-content-admin/internal/configclient"
)

var ErrNotFound = errors.New("object not found")
var ErrObjectTooLarge = errors.New("object exceeds size limit")

type ObjectStore struct {
	client   *minio.Client
	bucket   string
	basePath string
}

func NewFromConfig(cfg configclient.ObjectStorageConfiguration) (*ObjectStore, error) {
	if !cfg.Configured || !cfg.Enabled {
		return nil, fmt.Errorf("MinIO 未配置或未启用，请在配置中心设置")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if v := os.Getenv("APP_MINIO_ENDPOINT"); v != "" {
		endpoint = v
	}
	accessKey, secretKey := cfg.AccessKeyID, cfg.SecretAccessKey
	bucket, basePath, useSSL := cfg.BucketName, cfg.BasePath, cfg.UseSSL
	if v := os.Getenv("APP_MINIO_ACCESS_KEY"); v != "" {
		accessKey = v
	}
	if v := os.Getenv("APP_MINIO_SECRET_KEY"); v != "" {
		secretKey = v
	}
	if v := os.Getenv("APP_MINIO_BUCKET"); v != "" {
		bucket = v
	}
	if v, ok := os.LookupEnv("APP_MINIO_BASE_PATH"); ok {
		basePath = v
	}
	if v, ok := os.LookupEnv("APP_MINIO_USE_SSL"); ok {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("APP_MINIO_USE_SSL 无效: %w", err)
		}
		useSSL = parsed
	}
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &ObjectStore{
		client:   client,
		bucket:   bucket,
		basePath: strings.Trim(basePath, "/"),
	}, nil
}

func (s *ObjectStore) fullKey(relative string) string {
	if s.basePath == "" {
		return relative
	}
	return path.Join(s.basePath, relative)
}

func (s *ObjectStore) resolveKey(relativeOrFullKey string) string {
	key := relativeOrFullKey
	if s.basePath != "" && !strings.HasPrefix(relativeOrFullKey, s.basePath+"/") {
		key = s.fullKey(relativeOrFullKey)
	}
	return key
}

// PutBytes uploads bytes and returns the full object key (not a public URL).
func (s *ObjectStore) PutBytes(ctx context.Context, relativeKey string, data []byte, contentType string) (string, error) {
	key := s.fullKey(relativeKey)
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *ObjectStore) GetBytes(ctx context.Context, relativeOrFullKey string) ([]byte, error) {
	key := s.resolveKey(relativeOrFullKey)
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

// GetBytesLimited enforces the cap while streaming, before allocating an entire object.
func (s *ObjectStore) GetBytesLimited(ctx context.Context, key string, limit int64) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, s.resolveKey(key), minio.GetObjectOptions{})
	if err != nil {
		return nil, objectReadError(err)
	}
	defer obj.Close()
	b, err := io.ReadAll(io.LimitReader(obj, limit+1))
	if err != nil {
		return nil, objectReadError(err)
	}
	if int64(len(b)) > limit {
		return nil, ErrObjectTooLarge
	}
	return b, nil
}
func objectReadError(err error) error {
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	return err
}

// PutPNG uploads bytes and returns the full object key (not a public URL).
func (s *ObjectStore) PutPNG(ctx context.Context, relativeKey string, png []byte) (string, error) {
	return s.PutBytes(ctx, relativeKey, png, "image/png")
}

func (s *ObjectStore) GetPNG(ctx context.Context, relativeOrFullKey string) ([]byte, error) {
	return s.GetBytes(ctx, relativeOrFullKey)
}

func GlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("literacy/glyphs/%d.png", kpID)
}

func SenseObjectKey(kpID int64) string {
	return fmt.Sprintf("literacy/senses/%d.png", kpID)
}

func SpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("literacy/speech/%d.mp3", kpID)
}

func (s *ObjectStore) GlyphKey(kpID int64) string {
	return s.fullKey(GlyphObjectKey(kpID))
}

func (s *ObjectStore) SenseKey(kpID int64) string {
	return s.fullKey(SenseObjectKey(kpID))
}

func (s *ObjectStore) SpeechKey(kpID int64) string {
	return s.fullKey(SpeechObjectKey(kpID))
}

func PinyinSoloObjectKey(kpID int64) string {
	return fmt.Sprintf("pinyin/speech/%d/solo.mp3", kpID)
}

func PinyinWordObjectKey(kpID int64) string {
	return fmt.Sprintf("pinyin/speech/%d/word.mp3", kpID)
}

func PinyinWordExampleObjectKey(kpID int64, index int) string {
	if index <= 0 {
		return PinyinWordObjectKey(kpID)
	}
	return fmt.Sprintf("pinyin/speech/%d/word-%d.mp3", kpID, index)
}

func (s *ObjectStore) PinyinSoloKey(kpID int64) string {
	return s.fullKey(PinyinSoloObjectKey(kpID))
}

func (s *ObjectStore) PinyinWordKey(kpID int64) string {
	return s.fullKey(PinyinWordObjectKey(kpID))
}

func PinyinGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("pinyin/glyphs/%d.png", kpID)
}

func (s *ObjectStore) PinyinGlyphKey(kpID int64) string {
	return s.fullKey(PinyinGlyphObjectKey(kpID))
}

func MathGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("math/glyphs/%d.png", kpID)
}

func (s *ObjectStore) MathGlyphKey(kpID int64) string {
	return s.fullKey(MathGlyphObjectKey(kpID))
}

func MathSpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("math/speech/%d.mp3", kpID)
}

func MathQuestionSpeechObjectKey(questionID int64) string {
	return fmt.Sprintf("math/questions/%d.mp3", questionID)
}

func (s *ObjectStore) MathSpeechKey(kpID int64) string {
	return s.fullKey(MathSpeechObjectKey(kpID))
}

func EnglishGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("english/glyphs/%d.png", kpID)
}

func EnglishSenseObjectKey(kpID int64) string {
	return fmt.Sprintf("english/senses/%d.png", kpID)
}

func EnglishSpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("english/speech/%d.mp3", kpID)
}

func EnglishSentenceSpeechObjectKey(id int64) string {
	return fmt.Sprintf("english/sentences/%d.mp3", id)
}

func (s *ObjectStore) EnglishGlyphKey(kpID int64) string {
	return s.fullKey(EnglishGlyphObjectKey(kpID))
}

func (s *ObjectStore) EnglishSenseKey(kpID int64) string {
	return s.fullKey(EnglishSenseObjectKey(kpID))
}

func (s *ObjectStore) EnglishSpeechKey(kpID int64) string {
	return s.fullKey(EnglishSpeechObjectKey(kpID))
}

func (s *ObjectStore) EnglishSentenceSpeechKey(id int64) string {
	return s.fullKey(EnglishSentenceSpeechObjectKey(id))
}

func ScienceGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("science/glyphs/%d.png", kpID)
}

func ScienceGlyphVersionObjectKey(kpID int64, version int) string {
	return fmt.Sprintf("science/glyphs/%d-v%d.png", kpID, version)
}

func ScienceSenseObjectKey(kpID int64) string {
	return fmt.Sprintf("science/senses/%d.png", kpID)
}

func ScienceSenseVersionObjectKey(kpID int64, version int) string {
	return fmt.Sprintf("science/senses/%d-v%d.png", kpID, version)
}

func ScienceSpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("science/speech/%d.mp3", kpID)
}

func ScienceSpeechVersionObjectKey(kpID int64, version int) string {
	return fmt.Sprintf("science/speech/%d-v%d.mp3", kpID, version)
}

func (s *ObjectStore) ScienceGlyphKey(kpID int64) string {
	return s.fullKey(ScienceGlyphObjectKey(kpID))
}

func (s *ObjectStore) ScienceSenseKey(kpID int64) string {
	return s.fullKey(ScienceSenseObjectKey(kpID))
}

func (s *ObjectStore) ScienceSpeechKey(kpID int64) string {
	return s.fullKey(ScienceSpeechObjectKey(kpID))
}
