package http

import (
	"context"
	stdhttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Readiness interface {
	Check(context.Context) error
}

type Deps struct {
	Quiz      Quiz
	Plans     Plans
	Practice  Practice
	Database  *gorm.DB
	Readiness Readiness
}

func NewRouter(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if deps.Database == nil || deps.Readiness == nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "依赖暂时不可用")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if deps.Readiness.Check(ctx) != nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "依赖暂时不可用")
			return
		}
		writeData(c, stdhttp.StatusOK, gin.H{"status": "ready"})
	})
	if deps.Quiz != nil {
		router.POST("/api/v1/logic/quiz/generate", generateQuiz(deps.Quiz))
	} else {
		router.POST("/api/v1/logic/quiz/generate", func(c *gin.Context) {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "出题服务尚未就绪")
		})
	}
	if deps.Database != nil {
		router.GET("/api/v1/logic/items/:kpId/glyph/:objectId", serveLiveGlyph(deps.Database))
	}
	if deps.Plans != nil {
		router.GET("/api/v1/logic/task-media/:file", serveTaskMedia(deps.Plans))
		plans := router.Group("/api/v1/children/:childId/logic/plans")
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
