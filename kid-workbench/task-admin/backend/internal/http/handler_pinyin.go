package httpapi

import (
	"errors"
	"github.com/conchi/study-task-admin/internal/pinyintask"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strings"
)

func (h *handlers) registerPinyin(v1 *gin.RouterGroup) {
	group := v1.Group("/pinyin")
	group.Use(func(c *gin.Context) {
		if h.deps.Pinyin == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "拼音任务服务未配置"})
			return
		}
		c.Next()
	})
	group.GET("/question-tasks", func(c *gin.Context) { v, e := h.deps.Pinyin.List(); writePinyin(c, gin.H{"items": v}, e) })
	group.POST("/question-tasks", func(c *gin.Context) {
		var in pinyintask.CreateInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "无效的请求体"})
			return
		}
		v, e := h.deps.Pinyin.Create(c.Request.Context(), in)
		writePinyin(c, v, e)
	})
	group.GET("/question-tasks/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		v, e := h.deps.Pinyin.Get(id)
		writePinyin(c, v, e)
	})
	group.GET("/task-media/:file", func(c *gin.Context) {
		file := c.Param("file")
		if !strings.HasSuffix(file, ".mp3") {
			c.Status(404)
			return
		}
		data, e := h.deps.Pinyin.Audio(strings.TrimSuffix(file, ".mp3"))
		if e != nil {
			writePinyin(c, nil, e)
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(200, "audio/mpeg", data)
	})
}
func writePinyin(c *gin.Context, v any, e error) {
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
