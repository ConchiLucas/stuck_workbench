package asset

import (
	"context"
	"errors"
	"regexp"
)

var (
	ErrInvalidKey     = errors.New("invalid math question audio key")
	ErrUnavailable    = errors.New("audio unavailable")
	ErrObjectNotFound = errors.New("object not found")
)

var questionAudioKey = regexp.MustCompile(`^math/questions/[1-9][0-9]*\.mp3$`)

type Reader interface {
	Get(context.Context, string) ([]byte, error)
}

type Service struct{ reader Reader }

func NewService(reader Reader) *Service { return &Service{reader: reader} }

func (s *Service) QuestionAudio(ctx context.Context, key string) ([]byte, error) {
	if !questionAudioKey.MatchString(key) {
		return nil, ErrInvalidKey
	}
	data, err := s.reader.Get(ctx, key)
	if errors.Is(err, ErrObjectNotFound) {
		return nil, ErrUnavailable
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrUnavailable
	}
	return data, nil
}
