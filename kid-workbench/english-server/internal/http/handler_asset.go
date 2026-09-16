package http

import (
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/english-server/internal/asset"
)

type Assets interface {
	Glyph(context.Context, int64) ([]byte, error)
	Sense(context.Context, int64) ([]byte, error)
	Speech(context.Context, int64) ([]byte, error)
}

func serveWordAsset(assets Assets, catalog Catalog, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
		if err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_kp_id", "单词编号无效")
			return
		}
		if _, err := catalog.GetWord(c.Request.Context(), kpID); errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, stdhttp.StatusNotFound, "content_not_found", "英语单词不存在")
			return
		} else if err != nil {
			writeInternalError(c)
			return
		}
		var data []byte
		var contentType string
		switch kind {
		case "glyph":
			data, err, contentType = callAsset(assets.Glyph, c, kpID, "image/png")
		case "sense":
			data, err, contentType = callAsset(assets.Sense, c, kpID, "image/png")
		default:
			data, err, contentType = callAsset(assets.Speech, c, kpID, "audio/mpeg")
		}
		if errors.Is(err, asset.ErrMissing) {
			writeError(c, stdhttp.StatusNotFound, "asset_missing", "素材尚未生成")
			return
		}
		if err != nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "asset_unavailable", "素材服务暂时不可用")
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(stdhttp.StatusOK, contentType, data)
	}
}

func callAsset(fn func(context.Context, int64) ([]byte, error), c *gin.Context, kpID int64, contentType string) ([]byte, error, string) {
	data, err := fn(c.Request.Context(), kpID)
	return data, err, contentType
}

func serveSentenceSpeech(contentBase string) gin.HandlerFunc {
	client := &stdhttp.Client{Timeout: 15 * time.Second}
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.Param("id"))
		if id == "" || strings.Contains(id, "..") {
			writeError(c, stdhttp.StatusBadRequest, "invalid_sentence", "句子编号无效")
			return
		}
		if contentBase == "" {
			writeError(c, stdhttp.StatusServiceUnavailable, "asset_unavailable", "素材服务暂时不可用")
			return
		}
		req, err := stdhttp.NewRequestWithContext(c.Request.Context(), stdhttp.MethodGet, contentBase+"/api/v1/english/sentences/"+id+"/speech.mp3", nil)
		if err != nil {
			writeInternalError(c)
			return
		}
		res, err := client.Do(req)
		if err != nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "asset_unavailable", "素材服务暂时不可用")
			return
		}
		defer res.Body.Close()
		if res.StatusCode != stdhttp.StatusOK {
			writeError(c, stdhttp.StatusNotFound, "asset_missing", "整句读音尚未准备")
			return
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		if err != nil || len(data) == 0 || len(data) > 8<<20 {
			writeError(c, stdhttp.StatusNotFound, "asset_missing", "整句读音尚未准备")
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(stdhttp.StatusOK, "audio/mpeg", data)
	}
}
