package http

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strconv"

	"github.com/conchi/study-learning/sciencecontent"
	"github.com/conchi/study-science/internal/asset"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Assets interface {
	Glyph(context.Context, int64) ([]byte, int, error)
	Sense(context.Context, int64) ([]byte, int, error)
	Speech(context.Context, int64) ([]byte, int, error)
}

func serveGlyph(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) { serveScienceAsset(c, service, "glyph") }
}
func serveSense(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) { serveScienceAsset(c, service, "sense") }
}
func serveSpeech(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) { serveScienceAsset(c, service, "speech") }
}

func serveScienceAsset(c *gin.Context, service Assets, kind string) {
	kpID, ok := assetKPID(c)
	if !ok {
		return
	}
	var data []byte
	var version int
	var err error
	contentType := "image/png"
	switch kind {
	case "glyph":
		data, version, err = service.Glyph(c.Request.Context(), kpID)
	case "sense":
		data, version, err = service.Sense(c.Request.Context(), kpID)
	default:
		data, version, err = service.Speech(c.Request.Context(), kpID)
		contentType = "audio/mpeg"
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, stdhttp.StatusNotFound, "item_not_found", "科普条目不存在或尚未发布")
		return
	}
	if errors.Is(err, asset.ErrMissing) {
		writeError(c, stdhttp.StatusNotFound, "asset_missing", "素材尚未生成")
		return
	}
	if err != nil {
		writeError(c, stdhttp.StatusServiceUnavailable, "asset_unavailable", "素材服务暂不可用")
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("ETag", fmt.Sprintf(`W/"science-%d-%s-v%d"`, kpID, kind, version))
	c.Data(stdhttp.StatusOK, contentType, data)
}

func serveDiagram(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "database_unavailable", "数据库未配置")
			return
		}
		kpID, ok := assetKPID(c)
		if !ok {
			return
		}
		data, mime, err := sciencecontent.ItemDiagram(db.WithContext(c.Request.Context()), kpID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, stdhttp.StatusNotFound, "asset_missing", "结构图尚未准备")
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(stdhttp.StatusOK, mime, data)
	}
}

func serveTaskMedia(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		file := c.Param("file")
		data, ctype, err := service.FrozenMedia(c.Request.Context(), file)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, stdhttp.StatusNotFound, "asset_missing", "历史媒体不存在")
			return
		}
		if err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_media", "历史媒体编号无效")
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(stdhttp.StatusOK, ctype, data)
	}
}
func assetKPID(c *gin.Context) (int64, bool) {
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil || kpID <= 0 {
		writeError(c, stdhttp.StatusBadRequest, "invalid_kp_id", "科普条目编号无效")
		return 0, false
	}
	return kpID, true
}
