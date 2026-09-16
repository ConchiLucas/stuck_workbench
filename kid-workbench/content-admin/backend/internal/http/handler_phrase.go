package httpapi

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) listPhrase(c *gin.Context) {
	if h.deps.Phrase == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	view := c.DefaultQuery("view", "groups")
	res, err := h.deps.Phrase.List(view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) servePhraseSpeechMP3(c *gin.Context) {
	if h.deps.Phrase == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "短句编号无效"})
		return
	}
	mp3, err := h.deps.Phrase.SpeechMP3(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "整句读音暂不可用"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "audio/mpeg", mp3)
}

func (h *handlers) putPhraseSpeech(c *gin.Context) {
	if h.deps.Phrase == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "短句编号无效"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, (8<<20)+1))
	if err != nil || len(data) == 0 || len(data) > 8<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "整句读音无效"})
		return
	}
	item, err := h.deps.Phrase.StoreSpeech(c.Request.Context(), kpID, data)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "短句不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}
