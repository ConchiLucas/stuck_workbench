package asset

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

var ErrMissing = errors.New("asset missing")

type Reader interface {
	Get(context.Context, string) ([]byte, error)
}

type Service struct {
	reader Reader
}

func NewService(reader Reader) *Service { return &Service{reader: reader} }

func (s *Service) Glyph(ctx context.Context, kpID int64) ([]byte, error) {
	return s.get(ctx, GlyphKey(kpID))
}

func (s *Service) Speech(ctx context.Context, kpID int64, kind string) ([]byte, error) {
	if !ValidSpeechKind(kind) {
		return nil, fmt.Errorf("invalid speech kind %q", kind)
	}
	return s.get(ctx, SpeechKey(kpID, kind))
}

func (s *Service) get(ctx context.Context, key string) ([]byte, error) {
	data, err := s.reader.Get(ctx, key)
	if errors.Is(err, ErrObjectNotFound) {
		return nil, ErrMissing
	}
	return data, err
}

func (s *Service) SyllableSpeech(ctx context.Context, id int64, version string) ([]byte, error) {
	if id <= 0 {
		return nil, ErrMissing
	}
	key := fmt.Sprintf("pinyin/syllables/%d/speech.mp3", id)
	if version != "" {
		if !regexp.MustCompile(`^[a-f0-9]{16}$`).MatchString(version) {
			return nil, ErrMissing
		}
		key = fmt.Sprintf("pinyin/syllables/%d/speech-%s.mp3", id, version)
	}
	return s.get(ctx, key)
}
