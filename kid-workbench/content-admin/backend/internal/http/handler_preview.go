package httpapi

import (
	"bytes"
	"context"
	"errors"
	"github.com/conchi/study-learning/literacycontract"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func (h *handlers) registerLiteracyPreview(v1 *gin.RouterGroup) {
	v1.GET("/question-types/literacy", func(c *gin.Context) { c.JSON(200, literacycontract.Registry()) })
	for _, path := range []string{"/generation-preview/literacy", "/generation-preview/literacy/answer"} {
		v1.POST(path, proxyLiteracyPreview("/api/v1"+path))
	}
}
func proxyLiteracyPreview(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512<<10)
		b, err := io.ReadAll(c.Request.Body)
		if err != nil {
			var limit *http.MaxBytesError
			if errors.As(err, &limit) {
				c.JSON(413, gin.H{"error": "预览请求最大512KiB"})
			} else {
				c.JSON(400, gin.H{"error": "无效预览请求"})
			}
			return
		}
		base := strings.TrimRight(os.Getenv("APP_TASK_ADMIN_URL"), "/")
		if base == "" {
			base = "http://127.0.0.1:19201"
		}
		u, err := url.Parse(base)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			c.JSON(503, gin.H{"error": "题目预览服务未配置"})
			return
		}
		u.Path = path
		u.RawPath = ""
		u.RawQuery = ""
		u.Fragment = ""
		ctx, cancel := context.WithTimeout(c.Request.Context(), 35*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(b))
		if err != nil {
			c.JSON(503, gin.H{"error": "题目预览服务不可用"})
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 35 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, err := client.Do(req)
		if err != nil {
			c.JSON(503, gin.H{"error": "题目预览服务不可用"})
			return
		}
		defer res.Body.Close()
		payload, err := io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
		if err != nil || len(payload) > 4<<20 {
			c.JSON(503, gin.H{"error": "题目预览响应不可用"})
			return
		}
		c.Header("Cache-Control", "no-store")
		c.Data(res.StatusCode, "application/json", payload)
	}
}
