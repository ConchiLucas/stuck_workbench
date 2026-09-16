package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/study-learning/poemcontent"
)

func (h *handlers) syncPoem(c *gin.Context) {
	if h.deps.Poem == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.Poem.Sync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) listPoem(c *gin.Context) {
	if h.deps.Poem == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	view := c.DefaultQuery("view", "groups")
	res, err := h.deps.Poem.List(view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) servePoemSpeech(c *gin.Context) {
	if h.deps.Poem == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品编号"})
		return
	}
	file := strings.TrimSuffix(c.Param("file"), ".wav")
	file = strings.TrimSuffix(file, ".mp3")
	ord, ok := poemcontent.ParseLiveSpeech(file)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的诗行读音编号"})
		return
	}
	data, mime, err := h.deps.Poem.Speech(kpID, ord)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "读音素材暂不可用"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, mime, data)
}
