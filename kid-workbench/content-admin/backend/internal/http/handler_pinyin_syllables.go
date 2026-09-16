package httpapi

import (
	"errors"
	"github.com/conchi/study-content-admin/internal/pinyin"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func (h *handlers) listPinyinSyllables(c *gin.Context) {
	if h.deps.Pinyin == nil {
		c.JSON(503, gin.H{"error": "数据库未配置"})
		return
	}
	items, err := h.deps.Pinyin.ListSyllables(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}
func (h *handlers) updatePinyinSyllable(c *gin.Context) {
	if h.deps.Pinyin == nil {
		c.JSON(503, gin.H{"error": "数据库未配置"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "无效音节"})
		return
	}
	var update pinyin.SyllableUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}
	item, err := h.deps.Pinyin.UpdateSyllable(c.Request.Context(), id, update)
	if err != nil {
		status := 400
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = 404
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, item)
}
func (h *handlers) servePinyinSyllableSpeech(c *gin.Context) {
	if h.deps.Pinyin == nil {
		c.JSON(503, gin.H{"error": "数据库未配置"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "无效音节"})
		return
	}
	data, err := h.deps.Pinyin.SyllableSpeechMP3(c.Request.Context(), id, c.Query("v"))
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "audio/mpeg", data)
}
func (h *handlers) importPinyinSyllables(c *gin.Context) {
	if h.deps.Pinyin == nil {
		c.JSON(503, gin.H{"error": "数据库未配置"})
		return
	}
	root := strings.TrimSpace(os.Getenv("PINYIN_SYLLABLE_PACK_DIR"))
	if root == "" {
		root = "assets/pinyin-human-pack"
	}
	result, err := h.deps.Pinyin.ImportSyllableHumanPack(c.Request.Context(), root)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}
