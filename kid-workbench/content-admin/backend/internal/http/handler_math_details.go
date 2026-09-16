package httpapi

import (
	"errors"
	"io"
	"net/http"

	contentmath "github.com/conchi/study-content-admin/internal/math"
	"github.com/conchi/study-learning/mathcontent"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) mathDetails(c *gin.Context)          { h.listMathDetails(c, false) }
func (h *handlers) publishedMathDetails(c *gin.Context) { h.listMathDetails(c, true) }
func (h *handlers) listMathDetails(c *gin.Context, published bool) {
	if h.deps.Math == nil {
		c.JSON(503, gin.H{"error": "素材数据库未配置"})
		return
	}
	data, e := h.deps.Math.Details(c.Request.Context(), published)
	if e != nil {
		c.JSON(503, gin.H{"error": "详情目录暂不可用"})
		return
	}
	c.JSON(200, data)
}
func (h *handlers) saveMathDetail(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(503, gin.H{"error": "素材数据库未配置"})
		return
	}
	var d mathcontent.MathDetail
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if c.ShouldBindJSON(&d) != nil || d.ID != c.Param("id") {
		c.JSON(400, gin.H{"error": "详情身份或请求无效"})
		return
	}
	out, e := h.deps.Math.SaveDetail(c.Request.Context(), d)
	if e != nil {
		mathDetailError(c, e)
		return
	}
	c.JSON(200, out)
}
func (h *handlers) publishMathDetail(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(503, gin.H{"error": "素材数据库未配置"})
		return
	}
	var in struct {
		Revision int `json:"revision"`
	}
	if c.ShouldBindJSON(&in) != nil || in.Revision < 1 {
		c.JSON(400, gin.H{"error": "需要有效版本"})
		return
	}
	out, e := h.deps.Math.PublishDetail(c.Request.Context(), c.Param("id"), in.Revision)
	if e != nil {
		mathDetailError(c, e)
		return
	}
	c.JSON(200, out)
}
func mathDetailError(c *gin.Context, e error) {
	status := 400
	if errors.Is(e, gorm.ErrRecordNotFound) {
		status = 404
	}
	if errors.Is(e, contentmath.ErrDetailConflict) {
		status = 409
	}
	c.JSON(status, gin.H{"error": e.Error()})
}

func (h *handlers) generateMathDetailAudio(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(503, gin.H{"error": "素材服务未就绪"})
		return
	}
	var in struct {
		Revision int `json:"revision"`
	}
	if c.ShouldBindJSON(&in) != nil || in.Revision < 1 {
		c.JSON(400, gin.H{"error": "需要有效版本"})
		return
	}
	out, e := h.deps.Math.GenerateDetailAudio(c.Request.Context(), c.Param("id"), in.Revision)
	if e != nil {
		mathDetailError(c, e)
		return
	}
	c.JSON(200, out)
}
func (h *handlers) serveMathDetailAudio(c *gin.Context) {
	if h.deps.Math == nil {
		c.Status(503)
		return
	}
	data, e := h.deps.Math.DetailAudio(c.Request.Context(), c.Param("file"))
	if e != nil {
		c.JSON(404, gin.H{"error": "音频暂不可用"})
		return
	}
	c.Header("Cache-Control", "public,max-age=31536000,immutable")
	c.Data(200, "audio/mpeg", data)
}

func (h *handlers) uploadMathDetailImage(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(503, gin.H{"error": "素材服务未就绪"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<20)
	data, e := io.ReadAll(c.Request.Body)
	if e != nil {
		c.JSON(400, gin.H{"error": "图片无效或过大"})
		return
	}
	url, e := h.deps.Math.PutDetailImage(c.Request.Context(), data)
	if e != nil {
		mathDetailError(c, e)
		return
	}
	c.JSON(200, gin.H{"url": url})
}

func (h *handlers) serveMathDetailImage(c *gin.Context) {
	if h.deps.Math == nil {
		c.Status(503)
		return
	}
	file := c.Param("file")
	data, e := h.deps.Math.DetailImage(c.Request.Context(), file)
	if e != nil {
		c.JSON(404, gin.H{"error": "图片暂不可用"})
		return
	}
	c.Header("Cache-Control", "public,max-age=31536000,immutable")
	c.Data(200, contentmath.DetailImageContentType(file), data)
}
