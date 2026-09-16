package http

import (
	"context"
	stdhttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Readiness interface{ Check(context.Context) error }
type ReadinessGroup []Readiness

func (g ReadinessGroup) Check(ctx context.Context) error {
	for _, item := range g {
		if err := item.Check(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Deps struct {
	Readiness Readiness
	Catalog   Catalog
	Assets    Assets
	Content   string
	Progress  Progress
	Home      Home
	Plans     Plans
	Practice  Practice
	Quiz      Quiz
}

func NewRouter(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/readyz", func(c *gin.Context) {
		if deps.Readiness == nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "服务依赖尚未就绪")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := deps.Readiness.Check(ctx); err != nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "服务依赖尚未就绪")
			return
		}
		writeData(c, stdhttp.StatusOK, gin.H{"status": "ready"})
	})
	if deps.Catalog != nil {
		api := router.Group("/api/v1/english")
		api.GET("/modules", listModules(deps.Catalog))
		api.GET("/modules/:moduleCode/words", listWords(deps.Catalog))
		api.GET("/words/:kpId", getWord(deps.Catalog))
		if deps.Assets != nil {
			api.GET("/words/:kpId/glyph.png", serveWordAsset(deps.Assets, deps.Catalog, "glyph"))
			api.GET("/words/:kpId/sense.png", serveWordAsset(deps.Assets, deps.Catalog, "sense"))
			api.GET("/words/:kpId/speech.mp3", serveWordAsset(deps.Assets, deps.Catalog, "speech"))
			api.GET("/sentences/:id/speech.mp3", serveSentenceSpeech(deps.Content))
		}
	}
	if deps.Progress != nil {
		router.GET("/api/v1/children/:childId/english/progress", getProgress(deps.Progress))
	}
	if deps.Home != nil {
		router.GET("/api/v1/children/:childId/english/home", getHome(deps.Home))
	}
	if deps.Plans != nil {
		plans := router.Group("/api/v1/children/:childId/english/plans")
		plans.POST("", createPlan(deps.Plans))
		plans.GET("/:planId", getPlan(deps.Plans))
		plans.POST("/:planId/start", startPlan(deps.Plans))
		if deps.Practice != nil {
			plans.POST("/:planId/items/:itemId/answer", answerPlanItem(deps.Practice))
			plans.POST("/:planId/finish", finishPlan(deps.Practice))
		}
	}
	if deps.Quiz != nil {
		router.POST("/api/v1/english/quiz/generate", generateQuiz(deps.Quiz))
	} else {
		router.POST("/api/v1/english/quiz/generate", func(c *gin.Context) {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "出题服务尚未就绪")
		})
	}
	return router
}
