package httpapi

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) listChengyu(c *gin.Context) {
	if h.deps.Chengyu == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	view := c.DefaultQuery("view", "groups")
	res, err := h.deps.Chengyu.List(view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveChengyuSpeech(c *gin.Context) {
	h.serveChengyuKind(c, "chengyu")
}

func (h *handlers) serveChengyuMeaning(c *gin.Context) {
	h.serveChengyuKind(c, "meaning")
}

func (h *handlers) serveChengyuExample(c *gin.Context) {
	h.serveChengyuKind(c, "example")
}

func (h *handlers) serveChengyuKind(c *gin.Context, kind string) {
	if h.deps.Chengyu == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成语编号无效"})
		return
	}
	mp3, err := h.deps.Chengyu.SpeechMP3(c.Request.Context(), kpID, kind)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "成语音频暂不可用"})
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

func (h *handlers) putChengyuSpeech(c *gin.Context) {
	h.putChengyuKind(c, "chengyu")
}

func (h *handlers) putChengyuMeaning(c *gin.Context) {
	h.putChengyuKind(c, "meaning")
}

func (h *handlers) putChengyuExample(c *gin.Context) {
	h.putChengyuKind(c, "example")
}

func (h *handlers) putChengyuKind(c *gin.Context, kind string) {
	if h.deps.Chengyu == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成语编号无效"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, (8<<20)+1))
	if err != nil || len(data) == 0 || len(data) > 8<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成语音频无效"})
		return
	}
	item, err := h.deps.Chengyu.StoreSpeech(c.Request.Context(), kpID, kind, data)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "成语不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}
