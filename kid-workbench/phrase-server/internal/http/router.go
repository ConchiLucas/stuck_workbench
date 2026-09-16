package http

import (
	"context"
	stdhttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Readiness interface{ Check(context.Context) error }

type Deps struct {
	Readiness Readiness
	Plans     Plans
	Practice  Practice
	Media     FrozenMedia
}

type FrozenMedia interface {
	FrozenSpeech(context.Context, string) ([]byte, error)
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
	if deps.Plans != nil {
		plans := router.Group("/api/v1/children/:childId/phrase/plans")
		plans.POST("", createPlan(deps.Plans))
		plans.GET("/:planId", getPlan(deps.Plans))
		plans.POST("/:planId/start", startPlan(deps.Plans))
		if deps.Practice != nil {
			plans.POST("/:planId/items/:itemId/answer", answerPlanItem(deps.Practice))
			plans.POST("/:planId/finish", finishPlan(deps.Practice))
		}
	}
	if deps.Media != nil {
		router.GET("/api/v1/phrase/task-media/:file", serveFrozenSpeech(deps.Media))
	}
	return router
}

func serveFrozenSpeech(media FrozenMedia) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := media.FrozenSpeech(c.Request.Context(), c.Param("file"))
		if err != nil {
			c.Status(stdhttp.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(stdhttp.StatusOK, "audio/mpeg", data)
	}
}
