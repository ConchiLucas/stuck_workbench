package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/conchi/study-content-admin/internal/catalog"
	"github.com/conchi/study-content-admin/internal/chengyu"
	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/english"
	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/conchi/study-content-admin/internal/logic"
	contentmath "github.com/conchi/study-content-admin/internal/math"
	"github.com/conchi/study-content-admin/internal/phrase"
	"github.com/conchi/study-content-admin/internal/pinyin"
	"github.com/conchi/study-content-admin/internal/poem"
	"github.com/conchi/study-content-admin/internal/science"
)

type Deps struct {
	Catalog  *catalog.Service
	Literacy *literacy.Service
	English  *english.Service
	Pinyin   *pinyin.Service
	Math     *contentmath.Service
	Science  *science.Service
	Poem     *poem.Service
	Logic    *logic.Service
	Chengyu  *chengyu.Service
	Phrase   *phrase.Service
}

func NewRouter(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), cors())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	h := &handlers{deps: d}
	v1 := r.Group("/api/v1")
	h.registerLiteracyPreview(v1)
	{
		v1.GET("/runtime-config/catalog", h.getCatalog)
		v1.POST("/runtime-config/refresh", h.refreshCatalog)

		v1.GET("/generation-materials/literacy", h.generationMaterials)
		v1.POST("/generation-materials/literacy/freeze", h.freezeMaterials)
		v1.GET("/material-revisions/:revisionId/media/:kind", h.revisionMedia)

		v1.POST("/literacy/chars/:kpId/writing-template", h.importWritingTemplate)
		v1.GET("/literacy/chars/:kpId/writing-template", h.getWritingTemplate)
		v1.POST("/literacy/sync", h.syncLiteracy)
		v1.GET("/literacy/chars", h.listLiteracy)
		v1.PATCH("/literacy/chars/:kpId", h.patchLiteracyChar)
		v1.POST("/literacy/chars/:kpId/glyph", h.generateGlyph)
		v1.GET("/literacy/chars/:kpId/glyph.png", h.serveGlyphPNG)
		v1.GET("/literacy/chars/:kpId/speech.mp3", h.serveSpeechMP3)
		v1.POST("/literacy/chars/:kpId/speech", h.regenerateSpeech)
		v1.POST("/literacy/speech/batch", h.batchSpeech)
		v1.POST("/literacy/glyphs/batch", h.batchGlyphs)
		v1.POST("/literacy/chars/:kpId/sense", h.generateSense)
		v1.GET("/literacy/chars/:kpId/sense.png", h.serveSensePNG)
		v1.POST("/literacy/senses/batch", h.batchSenses)

		v1.POST("/english/sync", h.syncEnglish)
		v1.GET("/english/words", h.listEnglish)
		v1.PATCH("/english/words/:kpId", h.patchEnglishWord)
		v1.POST("/english/words/:kpId/glyph", h.generateEnglishGlyph)
		v1.GET("/english/words/:kpId/glyph.png", h.serveEnglishGlyphPNG)
		v1.GET("/english/words/:kpId/speech.mp3", h.serveEnglishSpeechMP3)
		v1.POST("/english/words/:kpId/speech", h.regenerateEnglishSpeech)
		v1.POST("/english/speech/batch", h.batchEnglishSpeech)
		v1.POST("/english/glyphs/batch", h.batchEnglishGlyphs)
		v1.POST("/english/words/:kpId/sense", h.generateEnglishSense)
		v1.GET("/english/words/:kpId/sense.png", h.serveEnglishSensePNG)
		v1.POST("/english/senses/batch", h.batchEnglishSenses)
		v1.GET("/english/sentences/:id/speech.mp3", h.serveEnglishSentenceSpeechMP3)
		v1.PUT("/english/sentences/:id/speech", h.putEnglishSentenceSpeech)

		v1.POST("/pinyin/sync", h.syncPinyin)
		v1.GET("/pinyin/items", h.listPinyin)
		v1.POST("/pinyin/items/:kpId/glyph", h.generatePinyinGlyph)
		v1.GET("/pinyin/items/:kpId/glyph.png", h.servePinyinGlyphPNG)
		v1.POST("/pinyin/glyphs/batch", h.batchPinyinGlyphs)
		v1.GET("/pinyin/items/:kpId/speech/:kind", h.servePinyinSpeechMP3)
		v1.GET("/pinyin/syllables", h.listPinyinSyllables)
		v1.PATCH("/pinyin/syllables/:id", h.updatePinyinSyllable)
		v1.GET("/pinyin/syllables/:id/speech.mp3", h.servePinyinSyllableSpeech)
		v1.POST("/pinyin/syllables/human-pack", h.importPinyinSyllables)
		v1.POST("/pinyin/items/:kpId/speech/:kind", h.regeneratePinyinSpeech)
		v1.POST("/pinyin/speech/batch", h.batchPinyinSpeech)
		v1.POST("/pinyin/speech/human-pack", h.importPinyinHumanPack)
		v1.POST("/pinyin/quiz/generate", h.generatePinyinQuiz)

		v1.POST("/math/sync", h.syncMath)
		v1.GET("/math/details", h.mathDetails)
		v1.GET("/math/details/published", h.publishedMathDetails)
		v1.PUT("/math/details/:id", h.saveMathDetail)
		v1.POST("/math/details/:id/publish", h.publishMathDetail)
		v1.POST("/math/details/:id/audio", h.generateMathDetailAudio)
		v1.GET("/math/detail-audio/:file", h.serveMathDetailAudio)
		v1.POST("/math/detail-images", h.uploadMathDetailImage)
		v1.GET("/math/detail-image/:file", h.serveMathDetailImage)
		v1.GET("/math/items", h.listMath)
		v1.PATCH("/math/items/:kpId/review", h.reviewMathItem)
		v1.POST("/math/items/:kpId/glyph", h.generateMathGlyph)
		v1.GET("/math/items/:kpId/glyph.png", h.serveMathGlyphPNG)
		v1.POST("/math/glyphs/batch", h.batchMathGlyphs)
		v1.GET("/math/items/:kpId/speech.mp3", h.serveMathSpeechMP3)
		v1.POST("/math/items/:kpId/speech", h.regenerateMathSpeech)
		v1.POST("/math/speech/batch", h.batchMathSpeech)
		v1.GET("/math/question-speech/status", h.mathQuestionSpeechStatus)
		v1.POST("/math/question-speech/batch", h.batchMathQuestionSpeech)
		v1.POST("/math/questions/:questionId/speech", h.regenerateMathQuestionSpeech)

		v1.POST("/science/sync", h.syncScience)
		v1.GET("/science/items", h.listScience)
		v1.PATCH("/science/items/:kpId", h.patchScienceItem)
		v1.PATCH("/science/items/:kpId/content", h.patchScienceContent)
		v1.POST("/science/items/:kpId/review", h.reviewScienceItem)
		v1.POST("/science/items/:kpId/publish", h.publishScienceItem)
		v1.POST("/science/items/:kpId/unpublish", h.unpublishScienceItem)
		v1.POST("/science/items/:kpId/glyph", h.generateScienceGlyph)
		v1.GET("/science/items/:kpId/glyph.png", h.serveScienceGlyphPNG)
		v1.POST("/science/glyphs/batch", h.batchScienceGlyphs)
		v1.POST("/science/items/:kpId/sense", h.generateScienceSense)
		v1.GET("/science/items/:kpId/sense.png", h.serveScienceSensePNG)
		v1.POST("/science/senses/batch", h.batchScienceSenses)
		v1.POST("/science/items/:kpId/speech", h.regenerateScienceSpeech)
		v1.GET("/science/items/:kpId/speech.mp3", h.serveScienceSpeechMP3)
		v1.GET("/science/items/:kpId/diagram.png", h.serveScienceDiagramPNG)
		v1.POST("/science/speech/batch", h.batchScienceSpeech)

		v1.POST("/poem/sync", h.syncPoem)
		v1.GET("/poem/items", h.listPoem)
		v1.GET("/poem/items/:kpId/speech/:file", h.servePoemSpeech)

		v1.GET("/logic/items", h.listLogic)
		v1.GET("/logic/items/:kpId/glyph/:objectId", h.serveLogicGlyph)
		v1.GET("/chengyu/items", h.listChengyu)
		v1.GET("/chengyu/items/:kpId/speech.mp3", h.serveChengyuSpeech)
		v1.PUT("/chengyu/items/:kpId/speech", h.putChengyuSpeech)
		v1.GET("/chengyu/items/:kpId/meaning.mp3", h.serveChengyuMeaning)
		v1.PUT("/chengyu/items/:kpId/meaning", h.putChengyuMeaning)
		v1.GET("/chengyu/items/:kpId/example.mp3", h.serveChengyuExample)
		v1.PUT("/chengyu/items/:kpId/example", h.putChengyuExample)
		v1.GET("/phrase/items", h.listPhrase)
		v1.GET("/phrase/items/:kpId/speech.mp3", h.servePhraseSpeechMP3)
		v1.PUT("/phrase/items/:kpId/speech", h.putPhraseSpeech)
	}
	mountSPA(r)
	return r
}

type handlers struct{ deps Deps }

func (h *handlers) getCatalog(c *gin.Context) {
	value, err := h.deps.Catalog.Build(c.Request.Context(), false)
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, value)
}

func (h *handlers) refreshCatalog(c *gin.Context) {
	value, err := h.deps.Catalog.Build(c.Request.Context(), true)
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, value)
}

func writeConfigError(c *gin.Context, err error) {
	msg := "无法加载共享配置中心"
	status := http.StatusServiceUnavailable
	if errors.Is(err, configclient.ErrNotLoaded) {
		msg = "配置尚未加载"
	} else if errors.Is(err, configclient.ErrUnavailable) {
		msg = "无法加载共享配置中心"
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{"error": msg})
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
