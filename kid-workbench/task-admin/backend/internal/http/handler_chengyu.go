package httpapi

import (
	"errors"
	"path"
	"strings"

	"github.com/conchi/study-task-admin/internal/chengyutask"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) registerChengyu(v1 *gin.RouterGroup) {
	group := v1.Group("/chengyu")
	group.Use(func(c *gin.Context) {
		if h.deps.Chengyu == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "成语任务服务未配置"})
			return
		}
		c.Next()
	})
	group.GET("/question-tasks", func(c *gin.Context) {
		v, e := h.deps.Chengyu.List()
		writeChengyu(c, gin.H{"items": v}, e)
	})
	group.POST("/question-tasks", func(c *gin.Context) {
		var in chengyutask.CreateInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "无效的请求体"})
			return
		}
		v, e := h.deps.Chengyu.Create(c.Request.Context(), in)
		writeChengyu(c, v, e)
	})
	group.GET("/question-tasks/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		v, e := h.deps.Chengyu.Get(id)
		writeChengyu(c, v, e)
	})
	group.GET("/task-media/:file", func(c *gin.Context) {
		file := c.Param("file")
		ctype, ok := mediaContentType(file)
		if !ok {
			c.Status(404)
			return
		}
		data, e := h.deps.Chengyu.Media(strings.TrimSuffix(file, path.Ext(file)))
		if e != nil {
			writeChengyu(c, nil, e)
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, ctype, data)
	})
}

func writeChengyu(c *gin.Context, v any, e error) {
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
