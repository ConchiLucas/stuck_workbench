package math

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/conchi/study-learning/mathcontent"
)

const detailImageLimit = 8 << 20

var detailImageName = regexp.MustCompile(`^[a-f0-9]{64}\.(png|jpe?g|webp)$`)
var pngMagic = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

func (s *Service) PutDetailImage(ctx context.Context, data []byte) (string, error) {
	if s.store == nil {
		return "", fmt.Errorf("图片存储未就绪")
	}
	if len(data) == 0 || len(data) > detailImageLimit {
		return "", fmt.Errorf("图片无效或过大")
	}
	ext, contentType, ok := detectDetailImage(data)
	if !ok {
		return "", fmt.Errorf("只支持 PNG、JPEG 或 WebP")
	}
	hash := sha256.Sum256(data)
	name := hex.EncodeToString(hash[:]) + ext
	if _, err := s.store.PutBytes(ctx, "math/detail-image/"+name, data, contentType); err != nil {
		return "", err
	}
	return "/api/v1/math/detail-image/" + name, nil
}

func (s *Service) DetailImage(ctx context.Context, file string) ([]byte, error) {
	if !detailImageName.MatchString(file) || s.store == nil {
		return nil, fmt.Errorf("图片不存在")
	}
	return s.store.GetBytes(ctx, "math/detail-image/"+file)
}

func (s *Service) verifyDetailImages(ctx context.Context, d mathcontent.MathDetail) error {
	urls := make([]string, 0, 1+len(d.Example.ShapeImageURLs))
	if d.Example.ObjectImageURL != "" {
		urls = append(urls, d.Example.ObjectImageURL)
	}
	for _, href := range d.Example.ShapeImageURLs {
		if href != "" {
			urls = append(urls, href)
		}
	}
	for _, href := range urls {
		if strings.Contains(href, "/task-media/") {
			continue
		}
		file := strings.TrimPrefix(href, "/api/v1/math/detail-image/")
		data, err := s.DetailImage(ctx, file)
		if err != nil || len(data) == 0 {
			return fmt.Errorf("数量图或图形图不存在，请重新上传后发布")
		}
		sum := sha256.Sum256(data)
		if file != hex.EncodeToString(sum[:])+path.Ext(file) {
			return fmt.Errorf("图片校验失败，请重新上传")
		}
	}
	return nil
}

func detectDetailImage(data []byte) (ext, contentType string, ok bool) {
	if len(data) >= 8 && bytes.Equal(data[:8], pngMagic) {
		return ".png", "image/png", true
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return ".jpg", "image/jpeg", true
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return ".webp", "image/webp", true
	}
	return "", "", false
}

func DetailImageContentType(file string) string {
	switch strings.ToLower(path.Ext(file)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
