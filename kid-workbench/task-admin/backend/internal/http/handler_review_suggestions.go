package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/conchi/study-task-admin/internal/reviewsuggestion"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
)

func suggestionError(c *gin.Context, e error) {
	var f *reviewsuggestion.Error
	if errors.As(e, &f) {
		c.JSON(f.Status, gin.H{"error": f.Message, "code": f.Code})
		return
	}
	if errors.Is(e, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{"error": "建议不存在", "code": "not_found"})
		return
	}
	c.JSON(500, gin.H{"error": "建议操作失败", "code": "internal_error"})
}
func strictSuggestionJSON(c *gin.Context, out any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	e := d.Decode(out)
	if e == nil {
		var extra any
		if tail := d.Decode(&extra); tail != io.EOF {
			e = errors.New("trailing JSON")
		}
	}
	if e != nil {
		var large *http.MaxBytesError
		status := 400
		if errors.As(e, &large) {
			status = 413
		}
		c.JSON(status, gin.H{"error": "请求体格式无效或超过1MB", "code": "invalid_body"})
		return false
	}
	return true
}
func (h *handlers) registerReviewSuggestions(v1 *gin.RouterGroup) {
	group := v1.Group("/children/:cid/review-suggestions")
	group.Use(func(c *gin.Context) {
		if h.deps.ReviewSuggestions == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "建议服务未配置", "code": "unavailable"})
			return
		}
		child, e := strconv.ParseInt(c.Param("cid"), 10, 64)
		if e != nil || child < 1 {
			c.AbortWithStatusJSON(400, gin.H{"error": "孩子编号无效", "code": "invalid_child"})
			return
		}
		c.Set("suggestionChild", child)
		c.Next()
	})
	childID := func(c *gin.Context) int64 { return c.MustGet("suggestionChild").(int64) }
	parseSuggestionID := func(c *gin.Context) (int64, bool) {
		id, e := strconv.ParseInt(c.Param("id"), 10, 64)
		if e != nil || id < 1 {
			c.JSON(400, gin.H{"error": "建议编号无效", "code": "invalid_id"})
			return 0, false
		}
		return id, true
	}
	group.POST("", func(c *gin.Context) {
		var in reviewsuggestion.Input
		if !strictSuggestionJSON(c, &in) {
			return
		}
		out, replay, e := h.deps.ReviewSuggestions.Save(c.Request.Context(), childID(c), c.GetHeader("Idempotency-Key"), in)
		if e != nil {
			suggestionError(c, e)
			return
		}
		status := 201
		if replay {
			status = 200
		}
		c.JSON(status, out)
	})
	group.GET("", func(c *gin.Context) {
		before, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
		limit, _ := strconv.Atoi(c.Query("limit"))
		out, e := h.deps.ReviewSuggestions.List(c.Request.Context(), childID(c), before, limit)
		if e != nil {
			suggestionError(c, e)
			return
		}
		c.JSON(200, out)
	})
	group.GET("/:id", func(c *gin.Context) {
		id, ok := parseSuggestionID(c)
		if !ok {
			return
		}
		out, e := h.deps.ReviewSuggestions.Get(c.Request.Context(), childID(c), id)
		if e != nil {
			suggestionError(c, e)
			return
		}
		c.JSON(200, out)
	})
	for _, part := range []string{"evidence", "tasks", "plans"} {
		part := part
		group.GET("/:id/"+part, func(c *gin.Context) {
			id, ok := parseSuggestionID(c)
			if !ok {
				return
			}
			after, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
			limit, _ := strconv.Atoi(c.Query("limit"))
			var out any
			var e error
			switch part {
			case "evidence":
				out, e = h.deps.ReviewSuggestions.EvidencePage(c.Request.Context(), childID(c), id, after, limit)
			case "tasks":
				out, e = h.deps.ReviewSuggestions.TasksPage(c.Request.Context(), childID(c), id, after, limit)
			case "plans":
				out, e = h.deps.ReviewSuggestions.PlansPage(c.Request.Context(), childID(c), id, after, limit)
			}
			if e != nil {
				suggestionError(c, e)
				return
			}
			c.JSON(200, out)
		})
	}
	for _, op := range []string{"generate", "retry", "cancel", "archive"} {
		op := op
		group.POST("/:id/"+op, func(c *gin.Context) {
			id, ok := parseSuggestionID(c)
			if !ok {
				return
			}
			var in reviewsuggestion.CommandInput
			if !strictSuggestionJSON(c, &in) {
				return
			}
			out, _, e := h.deps.ReviewSuggestions.Command(c.Request.Context(), childID(c), id, op, c.GetHeader("Idempotency-Key"), in)
			if e != nil {
				suggestionError(c, e)
				return
			}
			status := 200
			if op == "generate" || op == "retry" {
				status = 202
			}
			c.JSON(status, out)
		})
	}
}
