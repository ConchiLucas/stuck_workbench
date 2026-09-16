package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/study-content-admin/internal/science"
)

func (h *handlers) syncScience(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.Science.Sync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) listScience(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	view := c.DefaultQuery("view", "groups")
	var filter *bool
	switch c.Query("needsSenseImage") {
	case "true", "1":
		v := true
		filter = &v
	case "false", "0":
		v := false
		filter = &v
	}
	res, err := h.deps.Science.List(view, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) patchScienceItem(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	var body struct {
		NeedsSenseImageOverride *bool `json:"needsSenseImageOverride"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}
	// Allow explicit null to clear override: client sends {"needsSenseImageOverride": null}
	dto, err := h.deps.Science.PatchOverride(kpID, body.NeedsSenseImageOverride)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) patchScienceContent(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "service_unavailable", "error": "数据库未配置"})
		return
	}
	kpID, ok := parseScienceKPID(c)
	if !ok {
		return
	}
	var body science.ContentPatch
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "无效的请求体"})
		return
	}
	dto, err := h.deps.Science.PatchContent(kpID, body)
	writeSciencePublicationResult(c, dto, err)
}

func (h *handlers) reviewScienceItem(c *gin.Context) {
	kpID, ok := sciencePublicationRequest(c, h.deps.Science)
	if !ok {
		return
	}
	dto, err := h.deps.Science.Review(kpID, time.Now().UTC())
	writeSciencePublicationResult(c, dto, err)
}

func (h *handlers) publishScienceItem(c *gin.Context) {
	kpID, ok := sciencePublicationRequest(c, h.deps.Science)
	if !ok {
		return
	}
	dto, err := h.deps.Science.Publish(kpID)
	writeSciencePublicationResult(c, dto, err)
}

func (h *handlers) unpublishScienceItem(c *gin.Context) {
	kpID, ok := sciencePublicationRequest(c, h.deps.Science)
	if !ok {
		return
	}
	dto, err := h.deps.Science.Unpublish(kpID)
	writeSciencePublicationResult(c, dto, err)
}

func sciencePublicationRequest(c *gin.Context, service *science.Service) (int64, bool) {
	if service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "service_unavailable", "error": "数据库未配置"})
		return 0, false
	}
	return parseScienceKPID(c)
}

func parseScienceKPID(c *gin.Context) (int64, bool) {
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_kp_id", "error": "无效的 kpId"})
		return 0, false
	}
	return kpID, true
}

func writeSciencePublicationResult(c *gin.Context, dto science.ItemDTO, err error) {
	switch {
	case err == nil:
		c.JSON(http.StatusOK, dto)
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "条目不存在，请先同步"})
	case errors.Is(err, science.ErrReviewRequired):
		c.JSON(http.StatusConflict, gin.H{"code": "review_required", "error": "请先完成内容审核"})
	case errors.Is(err, science.ErrNoPublishedQuestion):
		c.JSON(http.StatusConflict, gin.H{"code": "no_published_question", "error": "缺少 recognize 题目，暂不能发布"})
	case errors.Is(err, science.ErrInvalidReviewState):
		c.JSON(http.StatusConflict, gin.H{"code": "invalid_review_state", "error": "当前状态不允许此操作"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"code": "internal_error", "error": "科普内容操作失败"})
	}
}

func (h *handlers) generateScienceGlyph(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Science.GenerateGlyph(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchScienceGlyphs(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	force := c.Query("force") == "1" || c.Query("force") == "true"
	res, err := h.deps.Science.BatchGenerateGlyphs(c.Request.Context(), force)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveScienceGlyphPNG(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	png, err := h.deps.Science.GlyphPNG(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", png)
}

func (h *handlers) serveScienceSpeechMP3(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	mp3, err := h.deps.Science.SpeechMP3(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
		return
	}
	if err != nil {
		msg := err.Error()
		status := http.StatusBadRequest
		if strings.Contains(msg, "TTS 未就绪") || strings.Contains(msg, "加载语音配置失败") || strings.Contains(msg, "没有可用的 TTS") || strings.Contains(msg, "语音存储未就绪") {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "audio/mpeg", mp3)
}

func (h *handlers) regenerateScienceSpeech(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Science.RegenerateSpeech(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
		return
	}
	if err != nil {
		msg := err.Error()
		status := http.StatusBadRequest
		if strings.Contains(msg, "TTS 未就绪") || strings.Contains(msg, "加载语音配置失败") || strings.Contains(msg, "没有可用的 TTS") || strings.Contains(msg, "语音存储未就绪") {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchScienceSpeech(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	moduleCode := strings.TrimSpace(c.Query("moduleCode"))
	if moduleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moduleCode 不能为空"})
		return
	}
	res, err := h.deps.Science.BatchGenerateSpeech(c.Request.Context(), moduleCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) generateScienceSense(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Science.GenerateSense(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchScienceSenses(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	workers, _ := strconv.Atoi(c.DefaultQuery("workers", "0"))
	maxRetries, _ := strconv.Atoi(c.DefaultQuery("maxRetries", "3"))
	res, err := h.deps.Science.BatchGenerateSenses(c.Request.Context(), c.Query("moduleCode"), workers, maxRetries)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveScienceSensePNG(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	png, err := h.deps.Science.SensePNG(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", png)
}

func (h *handlers) serveScienceDiagramPNG(c *gin.Context) {
	if h.deps.Science == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	data, mime, err := h.deps.Science.Diagram(kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "结构图尚未准备"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, mime, data)
}
