package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/conchi/study-task-admin/internal/taskgen"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

func genError(c *gin.Context, err error) {
	status := 422
	code := "invalid_request"
	var ge *taskgen.Error
	if errors.As(err, &ge) {
		status = ge.Status
		code = ge.Code
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		status = 404
		code = "not_found"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": err.Error()}})
}
func (h *handlers) generated(c *gin.Context) bool {
	if h.deps.Generation == nil {
		return false
	}
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id < 1 {
		return false
	}
	var row struct{ SourceMode string }
	e = h.deps.Generation.DB.Table("question_tasks").Select("source_mode").Where("id = ?", id).Take(&row).Error
	return e == nil && row.SourceMode == "material_template"
}
func (h *handlers) createGenerated(c *gin.Context) bool {
	if h.deps.Generation == nil {
		return false
	}
	raw, e := io.ReadAll(io.LimitReader(c.Request.Body, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 {
		c.JSON(400, gin.H{"error": "请求过大或无法读取"})
		return true
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	var body struct {
		Title string           `json:"title"`
		Spec  *generation.Spec `json:"spec"`
	}
	if json.Unmarshal(raw, &body) != nil || body.Spec == nil {
		return false
	}
	t, e := h.deps.Generation.Create(c.Request.Context(), body.Title, *body.Spec)
	if e != nil {
		genError(c, e)
	} else {
		c.JSON(201, t)
	}
	return true
}
func (h *handlers) listGenerated(c *gin.Context) {
	ts, e := h.deps.Generation.List(c.Request.Context(), c.Query("subject"), c.Query("status"), c.Query("kind"))
	if e != nil {
		genError(c, e)
		return
	}
	if c.Query("page") == "" {
		c.JSON(200, ts)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	start := (page - 1) * size
	if start > len(ts) {
		start = len(ts)
	}
	end := start + size
	if end > len(ts) {
		end = len(ts)
	}
	c.JSON(200, gin.H{"items": ts[start:end], "total": len(ts), "page": page, "pageSize": size})
}
func (h *handlers) generatedAction(c *gin.Context, action string) {
	s := h.deps.Generation
	if s == nil {
		c.JSON(503, gin.H{"error": "出题服务未配置"})
		return
	}
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id < 1 {
		c.JSON(400, gin.H{"error": "无效任务编号"})
		return
	}
	ctx := c.Request.Context()
	if action == "get" {
		t, e := s.Get(ctx, id)
		if e != nil {
			genError(c, e)
		} else {
			c.JSON(200, t)
		}
		return
	}
	if action == "revision" {
		rid, e := strconv.ParseInt(c.Param("revisionId"), 10, 64)
		if e != nil {
			c.JSON(400, gin.H{"error": "无效版本"})
			return
		}
		r, e := s.Revision(ctx, id, rid)
		if e != nil {
			genError(c, e)
		} else {
			c.JSON(200, r)
		}
		return
	}
	if action == "delete" {
		if e = s.Delete(ctx, id); e != nil {
			genError(c, e)
		} else {
			c.Status(204)
		}
		return
	}
	var body struct {
		Expected int64           `json:"expectedRowVersion"`
		Preview  bool            `json:"previewConfirmed"`
		Title    string          `json:"title"`
		Spec     generation.Spec `json:"spec"`
		Order    []int           `json:"order"`
	}
	if e = c.ShouldBindJSON(&body); e != nil {
		c.JSON(400, gin.H{"error": "请求体无效"})
		return
	}
	var t taskgen.Task
	switch action {
	case "generate", "replace":
		seq := 0
		if action == "replace" {
			seq, e = strconv.Atoi(c.Param("seq"))
			if e != nil || seq < 1 {
				c.JSON(400, gin.H{"error": "题号无效"})
				return
			}
		}
		t, e = s.Generate(ctx, id, body.Expected, c.GetHeader("Idempotency-Key"), seq)
	case "publish":
		t, e = s.Publish(ctx, id, body.Expected, body.Preview)
	case "unpublish":
		t, e = s.Transition(ctx, id, body.Expected, "draft")
	case "archive":
		t, e = s.Transition(ctx, id, body.Expected, "archived")
	case "update":
		t, e = s.Update(ctx, id, body.Expected, body.Title, body.Spec)
	case "order":
		t, e = s.Reorder(ctx, id, body.Expected, body.Order)
	default:
		e = fmt.Errorf("不支持的操作")
	}
	if e != nil {
		genError(c, e)
	} else {
		c.JSON(200, t)
	}
}
func (h *handlers) materialList(c *gin.Context) {
	if h.deps.Generation == nil || h.deps.Generation.Materials == nil {
		c.JSON(503, gin.H{"error": "素材服务未配置"})
		return
	}
	rows, e := h.deps.Generation.Materials.List(c.Request.Context(), c.Query("moduleCode"))
	if e != nil {
		genError(c, e)
		return
	}
	c.JSON(200, gin.H{"subjectCode": "literacy", "moduleCode": c.Query("moduleCode"), "items": rows})
}
func (h *handlers) materialMedia(c *gin.Context) {
	m, ok := h.deps.Generation.Materials.(*taskgen.MaterialClient)
	if !ok {
		c.Status(503)
		return
	}
	kind := c.Param("kind")
	if kind != "glyph" && kind != "sense" && kind != "speech" {
		c.Status(404)
		return
	}
	path := "/api/v1/material-revisions/" + url.PathEscape(c.Param("revisionId")) + "/media/" + kind
	req, e := http.NewRequestWithContext(c.Request.Context(), "GET", m.Base+path, nil)
	if e != nil {
		c.Status(400)
		return
	}
	res, e := m.Client.Do(req)
	if e != nil {
		c.Status(503)
		return
	}
	defer res.Body.Close()
	for _, k := range []string{"Content-Type", "ETag", "Cache-Control"} {
		if v := res.Header.Get(k); v != "" {
			c.Header(k, v)
		}
	}
	c.Status(res.StatusCode)
	_, _ = io.Copy(c.Writer, io.LimitReader(res.Body, 21<<20))
}
func (h *handlers) registerGeneration(v1 *gin.RouterGroup) {
	h.registerPreview(v1)
	v1.POST("/question-tasks/:id/import", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		task, e := h.deps.Generation.ImportLegacy(c.Request.Context(), id, c.GetHeader("Idempotency-Key"))
		if e != nil {
			genError(c, e)
		} else {
			c.JSON(201, task)
		}
	})
	for path, action := range map[string]string{"/question-tasks/:id/generate": "generate", "/question-tasks/:id/items/:seq/replace": "replace", "/question-tasks/:id/archive": "archive"} {
		a := action
		v1.POST(path, func(c *gin.Context) { h.generatedAction(c, a) })
	}
	v1.PATCH("/question-tasks/:id", func(c *gin.Context) { h.generatedAction(c, "update") })
	v1.PUT("/question-tasks/:id/order", func(c *gin.Context) { h.generatedAction(c, "order") })
	v1.GET("/question-tasks/:id/revisions/:revisionId", func(c *gin.Context) { h.generatedAction(c, "revision") })
	v1.GET("/generation-materials/literacy", h.materialList)
	v1.GET("/material-revisions/:revisionId/media/:kind", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		h.materialMedia(c)
	})
}
