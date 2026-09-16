package httpapi

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func materialError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, literacy.ErrStoreUnavailable), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		status = http.StatusServiceUnavailable
	case errors.Is(err, literacy.ErrSourceChanged):
		status = http.StatusConflict
	case errors.Is(err, literacy.ErrInvalidFreeze), errors.Is(err, literacy.ErrInvalidWritingTemplate):
		status = http.StatusBadRequest
	case errors.Is(err, literacy.ErrMaterialNotReady):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
func (h *handlers) generationMaterials(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(503, gin.H{"error": "素材服务未就绪"})
		return
	}
	result, err := h.deps.Literacy.GenerationMaterials(c.Request.Context(), c.Query("moduleCode"))
	if err != nil {
		materialError(c, err)
		return
	}
	c.JSON(200, result)
}
func (h *handlers) freezeMaterials(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(503, gin.H{"error": "素材服务未就绪"})
		return
	}
	var body struct {
		Items []literacy.FreezeItem `json:"items"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "无效冻结请求"})
		return
	}
	result, err := h.deps.Literacy.FreezeMaterials(c.Request.Context(), body.Items)
	if err != nil {
		materialError(c, err)
		return
	}
	c.JSON(200, result)
}
func (h *handlers) revisionMedia(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(503, gin.H{"error": "素材服务未就绪"})
		return
	}
	b, ct, err := h.deps.Literacy.RevisionMedia(c.Request.Context(), c.Param("revisionId"), c.Param("kind"))
	if err != nil {
		materialError(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	etag := fmt.Sprintf("\"%x\"", sha256.Sum256(b))
	c.Header("ETag", etag)
	for _, tag := range strings.Split(c.GetHeader("If-None-Match"), ",") {
		tag = strings.TrimSpace(tag)
		if tag == "*" || strings.TrimPrefix(tag, "W/") == etag {
			c.Status(http.StatusNotModified)
			return
		}
	}
	c.Data(200, ct, b)
}
