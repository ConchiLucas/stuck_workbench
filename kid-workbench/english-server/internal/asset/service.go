package asset

import (
	"context"
	"errors"
)

var (
	ErrMissing     = errors.New("asset missing")
	ErrUnavailable = errors.New("asset unavailable")
)

type Reader interface {
	Get(context.Context, string) ([]byte, error)
	Ping(context.Context) error
}
type Service struct{ reader Reader }

func NewService(reader Reader) *Service { return &Service{reader: reader} }
func (s *Service) Glyph(ctx context.Context, kpID int64) ([]byte, error) {
	return s.get(ctx, GlyphKey(kpID))
}
func (s *Service) Sense(ctx context.Context, kpID int64) ([]byte, error) {
	return s.get(ctx, SenseKey(kpID))
}
func (s *Service) Speech(ctx context.Context, kpID int64) ([]byte, error) {
	return s.get(ctx, SpeechKey(kpID))
}
func (s *Service) get(ctx context.Context, key string) ([]byte, error) {
	data, err := s.reader.Get(ctx, key)
	if errors.Is(err, ErrObjectNotFound) {
		return nil, ErrMissing
	}
	if err != nil {
		return nil, errors.Join(ErrUnavailable, err)
	}
	return data, nil
}
