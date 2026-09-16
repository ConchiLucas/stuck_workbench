package http

import (
	"context"
	"github.com/conchi/study-learning/mathcontent"
	"github.com/gin-gonic/gin"
)

type Content interface {
	List(context.Context) (mathcontent.Catalog, error)
	Audio(context.Context, string) ([]byte, error)
	Image(context.Context, string) ([]byte, error)
}

func listMathDetails(s Content) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, e := s.List(c.Request.Context())
		if e != nil {
			writeError(c, 503, "materials_unavailable", "素材详情暂不可用，请稍后重试")
			return
		}
		writeData(c, 200, data)
	}
}
func serveMathDetailAudio(s Content) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, e := s.Audio(c.Request.Context(), c.Param("file"))
		if e != nil {
			writeError(c, 503, "audio_unavailable", "题干音频暂不可用")
			return
		}
		c.Header("Cache-Control", "public,max-age=31536000,immutable")
		c.Data(200, "audio/mpeg", data)
	}
}

func serveMathDetailImage(s Content) gin.HandlerFunc {
	return func(c *gin.Context) {
		file := c.Param("file")
		data, e := s.Image(c.Request.Context(), file)
		if e != nil {
			writeError(c, 503, "image_unavailable", "题目图片暂不可用")
			return
		}
		c.Header("Cache-Control", "public,max-age=31536000,immutable")
		ctype := "image/jpeg"
		switch {
		case len(file) >= 4 && file[len(file)-4:] == ".png":
			ctype = "image/png"
		case len(file) >= 5 && file[len(file)-5:] == ".webp":
			ctype = "image/webp"
		}
		c.Data(200, ctype, data)
	}
}
