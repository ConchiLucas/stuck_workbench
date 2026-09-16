package asset

import (
	"context"
	"errors"

	"github.com/conchi/study-science/internal/catalog"
)

var ErrMissing = errors.New("asset missing")

type Reader interface {
	Get(context.Context, string) ([]byte, error)
}

type Scope interface {
	GetItem(context.Context, int64) (catalog.Item, error)
}

type Service struct {
	reader Reader
	scope  Scope
}

func NewService(reader Reader, scope Scope) *Service { return &Service{reader: reader, scope: scope} }

func (s *Service) Glyph(ctx context.Context, kpID int64) ([]byte, int, error) {
	return s.get(ctx, kpID, GlyphKey, func(item catalog.Item) string { return item.GlyphObjectKey })
}

func (s *Service) Sense(ctx context.Context, kpID int64) ([]byte, int, error) {
	return s.get(ctx, kpID, SenseKey, func(item catalog.Item) string { return item.SenseObjectKey })
}

func (s *Service) Speech(ctx context.Context, kpID int64) ([]byte, int, error) {
	return s.get(ctx, kpID, SpeechKey, func(item catalog.Item) string { return item.SpeechObjectKey })
}

func (s *Service) get(ctx context.Context, kpID int64, legacyKey func(int64) string, objectKey func(catalog.Item) string) ([]byte, int, error) {
	item, err := s.scope.GetItem(ctx, kpID)
	if err != nil {
		return nil, 0, err
	}
	key := objectKey(item)
	if key == "" {
		key = legacyKey(kpID)
	}
	data, err := s.reader.Get(ctx, key)
	if errors.Is(err, ErrObjectNotFound) {
		return nil, 0, ErrMissing
	}
	return data, item.ContentVersion, err
}
