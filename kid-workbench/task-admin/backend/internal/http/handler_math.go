package httpapi

import (
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/conchi/study-task-admin/internal/mathtask"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) registerMath(v1 *gin.RouterGroup) {
	v1.GET("/math/detail-audio/:file", func(c *gin.Context) {
		if h.deps.Math == nil {
			c.Status(503)
			return
		}
		data, err := h.deps.Math.Audio(c.Request.Context(), c.Param("file"))
		if err != nil {
			writeMath(c, nil, err)
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(200, "audio/mpeg", data)
	})
	v1.GET("/math/task-media/:file", func(c *gin.Context) {
		if h.deps.Math == nil {
			c.Status(503)
			return
		}
		file := c.Param("file")
		ctype, ok := mediaContentType(file)
		if !ok {
			c.Status(404)
			return
		}
		data, err := h.deps.Math.Media(file[:64])
		if err != nil {
			writeMath(c, nil, err)
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(200, ctype, data)
	})
	group := v1.Group("/math/question-tasks")
	group.Use(func(c *gin.Context) {
		if h.deps.Math == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "算术任务服务未配置"})
			return
		}
		c.Next()
	})
	group.GET("/materials", func(c *gin.Context) {
		catalog, err := h.deps.Math.Materials(c.Request.Context())
		writeMath(c, catalog, err)
	})
	group.GET("", func(c *gin.Context) { items, err := h.deps.Math.List(); writeMath(c, gin.H{"items": items}, err) })
	group.POST("", func(c *gin.Context) {
		var in mathtask.CreateInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(400, gin.H{"error": "无效的请求体"})
			return
		}
		task, err := h.deps.Math.Create(c.Request.Context(), in)
		writeMath(c, task, err)
	})
	group.GET("/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		task, err := h.deps.Math.Get(id)
		writeMath(c, task, err)
	})
	group.POST("/:id/publish", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		task, err := h.deps.Math.Publish(id)
		writeMath(c, task, err)
	})
}
func writeMath(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, data)
		return
	}
	status := http.StatusBadRequest
	if errors.Is(err, gorm.ErrRecordNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, mathtask.ErrMaterials) {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func mediaContentType(file string) (string, bool) {
	if len(file) < 64 || file[64] != '.' {
		return "", false
	}
	for _, c := range file[:64] {
		if c < '0' || c > '9' && (c < 'a' || c > 'f') {
			return "", false
		}
	}
	switch strings.ToLower(path.Ext(file)) {
	case ".mp3":
		return "audio/mpeg", true
	case ".wav":
		return "audio/wav", true
	case ".png":
		return "image/png", true
	case ".webp":
		return "image/webp", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".svg":
		return "image/svg+xml", true
	default:
		return "", false
	}
}
