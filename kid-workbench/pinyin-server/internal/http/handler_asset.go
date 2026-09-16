package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/conchi/pinyin-server/internal/asset"
)

type Assets interface {
	Glyph(context.Context, int64) ([]byte, error)
	Speech(context.Context, int64, string) ([]byte, error)
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

func serveSpeech(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) {
		kpID, ok := assetKPID(c)
		if !ok {
			return
		}
		file := c.Param("file")
		kind := strings.TrimSuffix(file, ".mp3")
		if !strings.HasSuffix(file, ".mp3") || !asset.ValidSpeechKind(kind) {
			writeError(c, stdhttp.StatusBadRequest, "invalid_asset_kind", "语音类型必须是 solo、word 或 word-1…word-4")
			return
		}
		data, err := service.Speech(c.Request.Context(), kpID, kind)
		serveAsset(c, data, err, "audio/mpeg")
	}
}

func assetKPID(c *gin.Context) (int64, bool) {
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		writeError(c, stdhttp.StatusBadRequest, "invalid_kp_id", "拼音条目编号无效")
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

type syllableAssets interface {
	SyllableSpeech(context.Context, int64, string) ([]byte, error)
}

func serveSyllableSpeech(service Assets) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			writeError(c, 400, "invalid_syllable_id", "音节编号无效")
			return
		}
		syllables, ok := service.(syllableAssets)
		if !ok {
			serveAsset(c, nil, asset.ErrMissing, "audio/mpeg")
			return
		}
		data, err := syllables.SyllableSpeech(c.Request.Context(), id, c.Query("v"))
		serveAsset(c, data, err, "audio/mpeg")
	}
}
