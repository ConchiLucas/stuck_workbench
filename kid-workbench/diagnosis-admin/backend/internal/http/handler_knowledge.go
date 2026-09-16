package httpapi

import (
	"errors"
	"fmt"
	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func knowledgeJSON(c *gin.Context, v any, e error) {
	if e == nil {
		c.JSON(200, v)
		return
	}
	status := 503
	code := "service_unavailable"
	message := "学习记录暂不可读取，请稍后重试"
	var fault *knowledge.Fault
	if errors.As(e, &fault) {
		status = fault.Status
		code = fault.Code
		message = fault.Message
	} else if errors.Is(e, gorm.ErrRecordNotFound) {
		status = 404
		code = "not_found"
		message = "找不到这份学习记录"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "retryable": status == 503}})
}
func knowledgeFilter(c *gin.Context) (knowledge.Filter, error) {
	f := knowledge.Filter{Subject: c.Query("subject"), Module: c.Query("module"), Q: c.Query("q"), State: c.Query("state"), Skill: c.Query("skill"), Cursor: c.Query("cursor"), View: c.Query("view"), FollowUpState: c.Query("followUpState"), From: c.Query("from"), To: c.Query("to"), MasteredOn: c.Query("masteredOn")}
	var e error
	if v := c.Query("limit"); v != "" {
		f.Limit, e = strconv.Atoi(v)
		if e != nil || f.Limit < 1 || f.Limit > 100 {
			return f, &knowledge.Fault{Status: 400, Code: "invalid_limit", Message: "分页数量无效"}
		}
	}
	if v := c.Query("kpId"); v != "" {
		f.KpID, e = strconv.ParseInt(v, 10, 64)
		if e != nil || f.KpID < 1 {
			return f, &knowledge.Fault{Status: 400, Code: "invalid_kp", Message: "知识点编号无效"}
		}
	}
	return f, nil
}
func (h *handlers) registerKnowledge(v1 *gin.RouterGroup) {
	g := v1.Group("/children/:cid/knowledge")
	g.Use(func(c *gin.Context) {
		if h.deps.Knowledge == nil {
			knowledgeJSON(c, nil, fmt.Errorf("not ready"))
			c.Abort()
			return
		}
		if _, ok := parseID(c, "cid", "孩子编号无效"); !ok {
			c.Abort()
		}
	})
	g.GET("/modules", func(c *gin.Context) {
		cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		var count int64
		if e := h.deps.Knowledge.DB.Table("children").Where("id=?", cid).Count(&count).Error; e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		if count == 0 {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		rows := []struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			SubjectCode string `json:"subjectCode"`
		}{}
		q := h.deps.Knowledge.DB.Table("modules m").Select("m.code,m.name,s.code AS subject_code").Joins("JOIN subjects s ON s.id=m.subject_id").Where("s.code<>'game'")
		if c.Query("subject") != "" {
			q = q.Where("s.code=?", c.Query("subject"))
		}
		e := q.Order("s.order_no,m.order_no,m.id").Scan(&rows).Error
		knowledgeJSON(c, gin.H{"items": rows}, e)
	})
	g.GET("/review-stages", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		raw := strings.Split(c.Query("ids"), ",")
		ids := make([]int64, 0, len(raw))
		if len(raw) > 100 {
			knowledgeJSON(c, nil, &knowledge.Fault{Status: 400, Code: "invalid_ids", Message: "最多100个建议编号"})
			return
		}
		for _, v := range raw {
			n, e := strconv.ParseInt(v, 10, 64)
			if e != nil || n < 1 {
				knowledgeJSON(c, nil, &knowledge.Fault{Status: 400, Code: "invalid_ids", Message: "建议编号无效"})
				return
			}
			ids = append(ids, n)
		}
		v, e := h.deps.Knowledge.ReviewStages(id, ids)
		knowledgeJSON(c, v, e)
	})
	g.GET("/review-status-summary", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		v, e := h.deps.Knowledge.ReviewStatusSummary(id)
		knowledgeJSON(c, v, e)
	})
	g.GET("/calendar", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		v, e := h.deps.Knowledge.Calendar(id, c.Query("month"))
		knowledgeJSON(c, v, e)
	})
	g.GET("/summary", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		v, e := h.deps.Knowledge.Summary(id)
		knowledgeJSON(c, v, e)
	})
	g.GET("/points", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		f, e := knowledgeFilter(c)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		v, e := h.deps.Knowledge.Points(id, f)
		knowledgeJSON(c, v, e)
	})
	g.GET("/points/:kpId", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		kp, ok := parseID(c, "kpId", "知识点编号无效")
		if !ok {
			return
		}
		v, e := h.deps.Knowledge.Point(id, kp)
		knowledgeJSON(c, v, e)
	})
	g.GET("/points/:kpId/attempts", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		kp, ok := parseID(c, "kpId", "知识点编号无效")
		if !ok {
			return
		}
		f, e := knowledgeFilter(c)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		f.KpID = kp
		f.WrongOnly = c.Query("result") == "wrong"
		v, e := h.deps.Knowledge.Attempts(id, f)
		knowledgeJSON(c, v, e)
	})
	g.GET("/attempts/:attemptId", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		a, ok := parseID(c, "attemptId", "作答编号无效")
		if !ok {
			return
		}
		v, e := h.deps.Knowledge.Attempt(id, a)
		knowledgeJSON(c, v, e)
	})
	g.GET("/attempts/:attemptId/media/:mediaId", h.knowledgeMedia)
	g.GET("/wrongs", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		f, e := knowledgeFilter(c)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		if f.View == "groups" || f.View == "patterns" {
			v, e := h.deps.Knowledge.Groups(id, f)
			knowledgeJSON(c, v, e)
			return
		}
		f.WrongOnly = true
		v, e := h.deps.Knowledge.Attempts(id, f)
		knowledgeJSON(c, v, e)
	})
	g.GET("/review-candidates", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
		f, e := knowledgeFilter(c)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		v, e := h.deps.Knowledge.Candidates(id, f)
		knowledgeJSON(c, v, e)
	})
	h.registerSuggestions(g)
}
func (h *handlers) knowledgeMedia(c *gin.Context) {
	child, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	id, ok := parseID(c, "attemptId", "作答编号无效")
	if !ok {
		return
	}
	a, e := h.deps.Knowledge.Attempt(child, id)
	if e != nil {
		knowledgeJSON(c, nil, e)
		return
	}
	raw, ok := a.Media[c.Param("mediaId")]
	if !ok {
		knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
		return
	}
	kind, path, ok := strings.Cut(raw, ":")
	if !ok {
		knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
		return
	}
	base := ""
	allowed := ""
	if kind == "content" {
		base = os.Getenv("APP_CONTENT_ADMIN_URL")
		if base == "" {
			base = "http://127.0.0.1:19091"
		}
		allowed = "/api/v1/material-revisions/"
	} else if kind == "pinyin" {
		base = os.Getenv("APP_PINYIN_SERVER_URL")
		if base == "" {
			base = "http://127.0.0.1:19111"
		}
		allowed = "/api/v1/pinyin/"
	} else if kind == "math" {
		base = os.Getenv("APP_MATH_SERVER_URL")
		if base == "" {
			base = "http://127.0.0.1:19141"
		}
		allowed = "/api/v1/children/"
	} else if kind == "english" {
		parsed, e := url.Parse(path)
		if e != nil {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		if strings.HasPrefix(parsed.Path, "/api/v1/english/task-media/") {
			base = os.Getenv("APP_TASK_ADMIN_URL")
			if base == "" {
				base = "http://127.0.0.1:19201"
			}
			allowed = "/api/v1/english/task-media/"
		} else if strings.HasPrefix(parsed.Path, "/api/v1/english/words/") || strings.HasPrefix(parsed.Path, "/api/v1/english/sentences/") {
			base = os.Getenv("APP_ENGLISH_SERVER_URL")
			if base == "" {
				base = "http://127.0.0.1:19131"
			}
			allowed = "/api/v1/english/"
		} else {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		if !strings.HasPrefix(parsed.Path, allowed) || strings.Contains(parsed.Path, "..") || strings.Contains(parsed.Path, "\\") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		req, e := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(base, "/")+parsed.RequestURI(), nil)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := client.Do(req)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		data, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if e != nil || len(data) > 8<<20 {
			knowledgeJSON(c, nil, fmt.Errorf("invalid media"))
			return
		}
		typ := res.Header.Get("Content-Type")
		if !strings.HasPrefix(typ, "image/") && !strings.HasPrefix(typ, "audio/") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		c.Header("Cache-Control", "private, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, typ, data)
		return
	} else if kind == "phrase" {
		parsed, e := url.Parse(path)
		if e != nil {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		if !strings.HasPrefix(parsed.Path, "/api/v1/phrase/task-media/") || strings.Contains(parsed.Path, "..") || strings.Contains(parsed.Path, "\\") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		base = os.Getenv("APP_TASK_ADMIN_URL")
		if base == "" {
			base = "http://127.0.0.1:19201"
		}
		req, e := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(base, "/")+parsed.RequestURI(), nil)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := client.Do(req)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		data, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if e != nil || len(data) > 8<<20 {
			knowledgeJSON(c, nil, fmt.Errorf("invalid media"))
			return
		}
		typ := res.Header.Get("Content-Type")
		if !strings.HasPrefix(typ, "audio/") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		c.Header("Cache-Control", "private, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, typ, data)
		return
	} else if kind == "science" {
		parsed, e := url.Parse(path)
		if e != nil {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		if !strings.HasPrefix(parsed.Path, "/api/v1/science/task-media/") || strings.Contains(parsed.Path, "..") || strings.Contains(parsed.Path, "\\") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		base = os.Getenv("APP_SCIENCE_SERVER_URL")
		if base == "" {
			base = "http://127.0.0.1:19121"
		}
		req, e := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(base, "/")+parsed.RequestURI(), nil)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := client.Do(req)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		data, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if e != nil || len(data) > 8<<20 {
			knowledgeJSON(c, nil, fmt.Errorf("invalid media"))
			return
		}
		typ := res.Header.Get("Content-Type")
		if !strings.HasPrefix(typ, "image/") && !strings.HasPrefix(typ, "audio/") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		c.Header("Cache-Control", "private, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, typ, data)
		return
	} else if kind == "poem" {
		parsed, e := url.Parse(path)
		if e != nil {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		if !strings.HasPrefix(parsed.Path, "/api/v1/poem/task-media/") || strings.Contains(parsed.Path, "..") || strings.Contains(parsed.Path, "\\") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		base = os.Getenv("APP_POEM_SERVER_URL")
		if base == "" {
			base = "http://127.0.0.1:19161"
		}
		req, e := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(base, "/")+parsed.RequestURI(), nil)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := client.Do(req)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		data, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if e != nil || len(data) > 8<<20 {
			knowledgeJSON(c, nil, fmt.Errorf("invalid media"))
			return
		}
		typ := res.Header.Get("Content-Type")
		if !strings.HasPrefix(typ, "audio/") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		c.Header("Cache-Control", "private, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, typ, data)
		return
	} else if kind == "logic" {
		parsed, e := url.Parse(path)
		if e != nil {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		if !strings.HasPrefix(parsed.Path, "/api/v1/logic/task-media/") || strings.Contains(parsed.Path, "..") || strings.Contains(parsed.Path, "\\") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		base = os.Getenv("APP_LOGIC_SERVER_URL")
		if base == "" {
			base = "http://127.0.0.1:19191"
		}
		req, e := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(base, "/")+parsed.RequestURI(), nil)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := client.Do(req)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		data, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if e != nil || len(data) > 8<<20 {
			knowledgeJSON(c, nil, fmt.Errorf("invalid media"))
			return
		}
		typ := res.Header.Get("Content-Type")
		if !strings.HasPrefix(typ, "image/") && typ != "image/svg+xml" {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		c.Header("Cache-Control", "private, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, typ, data)
		return
	} else if kind == "chengyu" {
		parsed, e := url.Parse(path)
		if e != nil {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		if !strings.HasPrefix(parsed.Path, "/api/v1/chengyu/task-media/") || strings.Contains(parsed.Path, "..") || strings.Contains(parsed.Path, "\\") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		base = os.Getenv("APP_TASK_ADMIN_URL")
		if base == "" {
			base = "http://127.0.0.1:19201"
		}
		req, e := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(base, "/")+parsed.RequestURI(), nil)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := client.Do(req)
		if e != nil {
			knowledgeJSON(c, nil, e)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		data, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if e != nil || len(data) > 8<<20 {
			knowledgeJSON(c, nil, fmt.Errorf("invalid media"))
			return
		}
		typ := res.Header.Get("Content-Type")
		if !strings.HasPrefix(typ, "audio/") {
			knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
			return
		}
		c.Header("Cache-Control", "private, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, typ, data)
		return
	} else {
		knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
		return
	}
	parsed, e := url.Parse(path)
	if e != nil || !strings.HasPrefix(parsed.Path, allowed) || strings.Contains(parsed.Path, "..") || strings.Contains(parsed.Path, "\\") {
		knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
		return
	}
	req, e := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(base, "/")+parsed.RequestURI(), nil)
	if e != nil {
		knowledgeJSON(c, nil, e)
		return
	}
	client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, e := client.Do(req)
	if e != nil {
		knowledgeJSON(c, nil, e)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
		return
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if e != nil || len(data) > 8<<20 {
		knowledgeJSON(c, nil, fmt.Errorf("invalid media"))
		return
	}
	typ := res.Header.Get("Content-Type")
	if !strings.HasPrefix(typ, "image/") && !strings.HasPrefix(typ, "audio/") {
		knowledgeJSON(c, nil, gorm.ErrRecordNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=300")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, typ, data)
}
