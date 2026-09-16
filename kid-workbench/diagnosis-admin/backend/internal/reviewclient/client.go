package reviewclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Base    string
	HTTP    *http.Client
	Enabled bool
}

func New(base string, enabled bool) *Client {
	return &Client{strings.TrimRight(base, "/"), &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, enabled}
}
func (c *Client) Request(ctx context.Context, method, path, key string, body []byte) (int, []byte, error) {
	if c == nil || c.Base == "" {
		return 503, nil, fmt.Errorf("题目后台尚未配置")
	}
	if method != "GET" && !c.Enabled {
		return 503, nil, fmt.Errorf("复习建议提交暂未启用")
	}
	u, e := url.Parse(c.Base)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return 503, nil, fmt.Errorf("题目后台地址无效")
	}
	req, e := http.NewRequestWithContext(ctx, method, c.Base+path, bytes.NewReader(body))
	if e != nil {
		return 503, nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, e := c.HTTP.Do(req)
	if e != nil {
		return 503, nil, fmt.Errorf("题目后台暂不可用，请稍后重试")
	}
	defer res.Body.Close()
	data, e := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if e != nil {
		return 503, nil, e
	}
	if !json.Valid(data) {
		return 503, nil, fmt.Errorf("题目后台返回了无法读取的结果")
	}
	return res.StatusCode, data, nil
}
