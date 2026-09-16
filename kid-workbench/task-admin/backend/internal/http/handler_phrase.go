package httpapi

import (
	"errors"
	"path"
	"strings"

	"github.com/conchi/study-task-admin/internal/phrasetask"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) registerPhrase(v1 *gin.RouterGroup) {
	group := v1.Group("/phrase")
	group.Use(func(c *gin.Context) {
		if h.deps.Phrase == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "短句任务服务未配置"})
			return
		}
		c.Next()
	})
	group.GET("/question-tasks", func(c *gin.Context) {
		v, e := h.deps.Phrase.List()
		writePhrase(c, gin.H{"items": v}, e)
	})
	group.POST("/question-tasks", func(c *gin.Context) {
		var in phrasetask.CreateInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "无效的请求体"})
			return
		}
		v, e := h.deps.Phrase.Create(c.Request.Context(), in)
		writePhrase(c, v, e)
	})
	group.GET("/question-tasks/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		v, e := h.deps.Phrase.Get(id)
		writePhrase(c, v, e)
	})
	group.GET("/task-media/:file", func(c *gin.Context) {
		file := c.Param("file")
		ctype, ok := mediaContentType(file)
		if !ok {
			c.Status(404)
			return
		}
		data, e := h.deps.Phrase.Media(strings.TrimSuffix(file, path.Ext(file)))
		if e != nil {
			writePhrase(c, nil, e)
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, ctype, data)
	})
}

func writePhrase(c *gin.Context, v any, e error) {
	if e == nil {
		c.JSON(200, v)
		return
	}
	status := 400
	if errors.Is(e, gorm.ErrRecordNotFound) {
		status = 404
	}
	c.JSON(status, gin.H{"error": e.Error()})
}
