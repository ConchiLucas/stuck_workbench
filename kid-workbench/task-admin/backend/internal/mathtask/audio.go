package mathtask

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

var audioName = regexp.MustCompile(`^[a-f0-9]{64}\.mp3$`)

// Audio proxies only immutable material media for older packages that still
// reference detail-audio. New packages serve frozen bytes from Media().
func (s *Service) Audio(ctx context.Context, name string) ([]byte, error) {
	if !audioName.MatchString(name) {
		return nil, fmt.Errorf("无效音频文件")
	}
	if s.db != nil {
		if data, err := s.Media(name[:64]); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.contentURL+"/api/v1/math/detail-audio/"+name, nil)
	if err != nil {
		return nil, ErrMaterials
	}
	res, err := s.client.Do(req)
	if err != nil {
		return nil, ErrMaterials
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, ErrMaterials
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (10<<20)+1))
	if err != nil || len(data) > 10<<20 || len(data) == 0 {
		return nil, ErrMaterials
	}
	return data, nil
}
