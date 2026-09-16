package httpapi

import (
	"errors"
	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
)

func (h *handlers) importWritingTemplate(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(503, gin.H{"error": "素材服务未就绪"})
		return
	}
	id, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "无效知识点"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, literacy.WritingTemplateLimit)
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var size *http.MaxBytesError
		if errors.As(err, &size) {
			c.JSON(413, gin.H{"error": "模板最大2MiB"})
		} else {
			c.JSON(400, gin.H{"error": "无法读取模板"})
		}
		return
	}
	result, err := h.deps.Literacy.ImportWritingTemplate(c.Request.Context(), id, b)
	if err != nil {
		materialError(c, err)
		return
	}
	c.JSON(200, result)
}
func (h *handlers) getWritingTemplate(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(503, gin.H{"error": "素材服务未就绪"})
		return
	}
	id, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "无效知识点"})
		return
	}
	result, err := h.deps.Literacy.WritingTemplate(c.Request.Context(), id, c.Query("version"))
	if err != nil {
		materialError(c, err)
		return
	}
	c.JSON(200, result)
}
