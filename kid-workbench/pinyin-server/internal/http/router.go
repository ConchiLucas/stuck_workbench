package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	Catalog    Catalog
	Assets     Assets
	Progress   Progress
	Plans      Plans
	Practice   Practice
	Home       Home
	Quiz       Quiz
	FormalQuiz FormalQuiz
}

func NewRouter(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
	})
	if deps.Catalog != nil {
		api := router.Group("/api/v1/pinyin")
		api.GET("/modules", listModules(deps.Catalog))
		api.GET("/modules/:moduleCode/items", listItems(deps.Catalog))
		api.GET("/items/:kpId", getItem(deps.Catalog))
	}
	if deps.Quiz != nil {
		router.POST("/api/v1/pinyin/quiz/generate", generateQuiz(deps.Quiz))
	}
	if deps.FormalQuiz != nil {
		api := router.Group("/api/v1/children/:childId/pinyin/quiz")
		api.POST("/generate", generateFormalQuiz(deps.FormalQuiz))
		api.POST("/:instanceId/answer", answerFormalQuiz(deps.FormalQuiz))
		api.GET("/:instanceId", getFormalQuiz(deps.FormalQuiz))
	}
	if deps.Assets != nil {
		router.GET("/api/v1/pinyin/syllables/:id/speech.mp3", serveSyllableSpeech(deps.Assets))
		api := router.Group("/api/v1/pinyin/items/:kpId")
		api.GET("/glyph.png", serveGlyph(deps.Assets))
		api.GET("/speech/:file", serveSpeech(deps.Assets))
	}
	if deps.Progress != nil {
		router.GET("/api/v1/children/:childId/pinyin/progress", getProgress(deps.Progress))
	}
	if deps.Home != nil {
		router.GET("/api/v1/children/:childId/pinyin/home", getHome(deps.Home))
	}
	if deps.Plans != nil {
		plans := router.Group("/api/v1/children/:childId/pinyin/plans")
		plans.POST("", createPlan(deps.Plans))
		plans.GET("/:planId", getPlan(deps.Plans))
		plans.POST("/:planId/start", startPlan(deps.Plans))
		if deps.Practice != nil {
			plans.POST("/:planId/items/:itemId/answer", answerPlanItem(deps.Practice))
			plans.POST("/:planId/finish", finishPlan(deps.Practice))
		}
	}
	return router
}
