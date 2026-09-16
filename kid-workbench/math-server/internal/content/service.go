package content

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/mathcontent"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Service struct {
	base    string
	client  *http.Client
	mu      sync.Mutex
	cached  []byte
	expires time.Time
	fetched time.Time
}

func NewService(base string) *Service {
	return &Service{base: strings.TrimRight(base, "/"), client: &http.Client{Timeout: 4 * time.Second}}
}
func (s *Service) read(ctx context.Context, path string, limit int64) ([]byte, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", s.base+path, nil)
	if e != nil {
		return nil, e
	}
	res, e := s.client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("素材服务不可用：%d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("素材响应过大")
	}
	return b, nil
}
func (s *Service) List(ctx context.Context) (mathcontent.Catalog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var out mathcontent.Catalog
	if now.Before(s.expires) && len(s.cached) > 0 {
		return out, json.Unmarshal(s.cached, &out)
	}
	raw, e := s.read(ctx, "/api/v1/math/details/published", 2<<20)
	if e == nil {
		e = json.Unmarshal(raw, &out)
		if e == nil && out.SchemaVersion != 1 {
			e = fmt.Errorf("不支持的素材版本")
		}
		ids := map[string]bool{}
		if e == nil {
			for _, d := range out.Items {
				if ids[d.ID] {
					e = fmt.Errorf("素材身份重复")
					break
				}
				ids[d.ID] = true
				if e = mathcontent.Validate(d); e != nil {
					break
				}
			}
		}
	}
	if e != nil {
		if len(s.cached) > 0 && now.Sub(s.fetched) < 24*time.Hour {
			if err := json.Unmarshal(s.cached, &out); err != nil {
				return out, err
			}
			out.Stale = true
			return out, nil
		}
		return mathcontent.Catalog{}, fmt.Errorf("素材详情暂不可用：%w", e)
	}
	if out.Items == nil {
		out.Items = []mathcontent.MathDetail{}
	}
	s.cached, _ = json.Marshal(out)
	s.expires = now.Add(time.Minute)
	s.fetched = now
	return out, nil
}

var audioFile = regexp.MustCompile(`^[a-f0-9]{64}\.mp3$`)
var imageFile = regexp.MustCompile(`^[a-f0-9]{64}\.(png|jpe?g|webp)$`)

func (s *Service) Audio(ctx context.Context, file string) ([]byte, error) {
	if !audioFile.MatchString(file) {
		return nil, fmt.Errorf("无效音频")
	}
	return s.read(ctx, "/api/v1/math/detail-audio/"+file, 10<<20)
}

func (s *Service) Image(ctx context.Context, file string) ([]byte, error) {
	if !imageFile.MatchString(file) {
		return nil, fmt.Errorf("无效图片")
	}
	return s.read(ctx, "/api/v1/math/detail-image/"+file, 8<<20)
}
