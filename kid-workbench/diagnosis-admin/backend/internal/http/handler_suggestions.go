package httpapi

import (
	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/gin-gonic/gin"
	"io"
	"net/url"
	"strconv"
	"strings"
)

func (h *handlers) registerSuggestions(g *gin.RouterGroup) {
	g.GET("/review-suggestions/:id/outcomes", h.reviewOutcomes)
	forward := func(c *gin.Context) {
		cid, ok := parseID(c, "cid", "孩子编号无效")
		if !ok {
			return
		}
		suffix := ""
		if c.Param("id") != "" {
			id, ok := parseID(c, "id", "建议编号无效")
			if !ok {
				return
			}
			suffix = "/" + strconv.FormatInt(id, 10)
		}
		parts := strings.Split(c.FullPath(), "/")
		end := parts[len(parts)-1]
		if end != "review-suggestions" && end != ":id" {
			suffix += "/" + end
		}
		path := "/api/v1/children/" + strconv.FormatInt(cid, 10) + "/review-suggestions" + suffix
		if c.Request.Method == "GET" {
			query := url.Values{}
			for _, k := range []string{"cursor", "limit", "lifecycle"} {
				if v := c.Query(k); v != "" {
					query.Set(k, v)
				}
			}
			if len(query) > 0 {
				path += "?" + query.Encode()
			}
		}
		body, e := io.ReadAll(io.LimitReader(c.Request.Body, (1<<20)+1))
		if e != nil || len(body) > 1<<20 {
			knowledgeJSON(c, nil, &knowledge.Fault{Status: 400, Code: "invalid_body", Message: "请求内容过大或无法读取"})
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if c.Request.Method != "GET" && (key == "" || len(key) > 120) {
			knowledgeJSON(c, nil, &knowledge.Fault{Status: 400, Code: "idempotency_required", Message: "缺少有效请求标识，请重试"})
			return
		}
		status, data, e := h.deps.Reviews.Request(c.Request.Context(), c.Request.Method, path, key, body)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		c.Data(status, "application/json", data)
	}
	for _, path := range []string{"/review-suggestions", "/review-suggestions/:id", "/review-suggestions/:id/evidence", "/review-suggestions/:id/tasks", "/review-suggestions/:id/plans"} {
		g.GET(path, forward)
	}
	for _, path := range []string{"/review-suggestions", "/review-suggestions/:id/generate", "/review-suggestions/:id/retry", "/review-suggestions/:id/cancel", "/review-suggestions/:id/archive"} {
		g.POST(path, forward)
	}
}
