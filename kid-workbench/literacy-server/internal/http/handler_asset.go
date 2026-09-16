package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/conchi/literacy-server/internal/asset"
)

type Assets interface {
	Glyph(context.Context, int64) ([]byte, error)
	Sense(context.Context, int64) ([]byte, error)
	Speech(context.Context, int64) ([]byte, error)
}

func serveGlyph(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) {
		kpID, ok := assetKPID(c)
		if !ok {
			return
		}
		data, err := service.Glyph(c.Request.Context(), kpID)
		serveAsset(c, data, err, "image/png")
	}
}

func serveSense(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) {
		kpID, ok := assetKPID(c)
		if !ok {
			return
		}
		data, err := service.Sense(c.Request.Context(), kpID)
		serveAsset(c, data, err, "image/png")
	}
}

func serveSpeech(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) {
		kpID, ok := assetKPID(c)
		if !ok {
			return
		}
		data, err := service.Speech(c.Request.Context(), kpID)
		serveAsset(c, data, err, "audio/mpeg")
	}
}

func assetKPID(c *gin.Context) (int64, bool) {
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		writeError(c, stdhttp.StatusBadRequest, "invalid_kp_id", "汉字编号无效")
		return 0, false
	}
	return kpID, true
}

func serveAsset(c *gin.Context, data []byte, err error, contentType string) {
	if errors.Is(err, asset.ErrMissing) {
		writeError(c, stdhttp.StatusNotFound, "asset_missing", "素材尚未生成")
		return
	}
	if err != nil {
		writeInternalError(c)
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(stdhttp.StatusOK, contentType, data)
}
