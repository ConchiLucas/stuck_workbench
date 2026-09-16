package httpapi

import (
	"errors"

	"github.com/conchi/study-task-admin/internal/logictask"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) registerLogic(v1 *gin.RouterGroup) {
	group := v1.Group("/logic")
	group.Use(func(c *gin.Context) {
		if h.deps.Logic == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "逻辑任务服务未配置"})
			return
		}
		c.Next()
	})
	group.GET("/question-tasks", func(c *gin.Context) {
		v, e := h.deps.Logic.List()
		writeLogic(c, gin.H{"items": v}, e)
	})
	group.POST("/question-tasks", func(c *gin.Context) {
		var in logictask.CreateInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "无效的请求体"})
			return
		}
		v, e := h.deps.Logic.Create(c.Request.Context(), in)
		writeLogic(c, v, e)
	})
	group.GET("/question-tasks/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		v, e := h.deps.Logic.Get(id)
		writeLogic(c, v, e)
	})
	group.GET("/task-media/:file", func(c *gin.Context) {
		file := c.Param("file")
		ctype, ok := mediaContentType(file)
		if !ok {
			c.Status(404)
			return
		}
		data, got, e := h.deps.Logic.Media(file)
		if e != nil {
			writeLogic(c, nil, e)
			return
		}
		if got != "" {
			ctype = got
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, ctype, data)
	})
}

func writeLogic(c *gin.Context, v any, e error) {
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
