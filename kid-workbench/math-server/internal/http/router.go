package http

import (
	"context"
	stdhttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/catalog"
	"github.com/conchi/math-server/internal/home"
	"github.com/conchi/math-server/internal/plan"
	"github.com/conchi/math-server/internal/practice"
	"github.com/conchi/math-server/internal/progress"
)

type Catalog interface {
	ListModules(context.Context) ([]catalog.Module, error)
	GetModule(context.Context, string) (catalog.Module, error)
	ListStageItems(context.Context, string, string) ([]catalog.LearningItem, error)
}

type Progress interface {
	Get(context.Context, int64) (progress.Result, error)
}

type Plans interface {
	Create(context.Context, int64, plan.CreateInput) (plan.Detail, error)
	Get(context.Context, int64, int64) (plan.Detail, error)
	Start(context.Context, int64, int64) (plan.Detail, error)
}

type Assets interface {
	QuestionAudio(context.Context, string) ([]byte, error)
}

type Practice interface {
	Answer(context.Context, int64, int64, int64, practice.AnswerInput) (practice.AnswerResult, error)
	Finish(context.Context, int64, int64) (plan.StudyPlan, error)
}

type Home interface {
	Get(context.Context, int64) (home.Home, error)
}

type Readiness interface {
	Ready(context.Context) error
}

type Deps struct {
	Content   Content
	Catalog   Catalog
	Progress  Progress
	Plans     Plans
	Practice  Practice
	Assets    Assets
	Database  *gorm.DB
	Home      Home
	Readiness Readiness
	Quiz      Quiz
}

func NewRouter(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())
	if deps.Content != nil {
		router.GET("/api/v1/math/details", listMathDetails(deps.Content))
		router.GET("/api/v1/math/detail-audio/:file", serveMathDetailAudio(deps.Content))
		router.GET("/api/v1/math/detail-image/:file", serveMathDetailImage(deps.Content))
	}
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
		sqlDB, err := deps.Database.DB()
		if err != nil || sqlDB.PingContext(ctx) != nil || deps.Readiness.Ready(ctx) != nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "依赖暂时不可用")
			return
		}
		writeData(c, stdhttp.StatusOK, gin.H{"status": "ready"})
	})
	if deps.Catalog != nil {
		api := router.Group("/api/v1/math/modules")
		api.GET("", listMathModules(deps.Catalog))
		api.GET("/:moduleCode", getMathModule(deps.Catalog))
		api.GET("/:moduleCode/stages/:stageCode", listMathStageItems(deps.Catalog))
	}
	if deps.Progress != nil {
		router.GET("/api/v1/children/:childId/math/progress", getMathProgress(deps.Progress))
	}
	if deps.Home != nil {
		router.GET("/api/v1/children/:childId/math/home", getMathHome(deps.Home))
	}
	if deps.Plans != nil {
		plans := router.Group("/api/v1/children/:childId/math/plans")
		plans.POST("", createMathPlan(deps.Plans))
		plans.GET("/:planId", getMathPlan(deps.Plans))
		plans.POST("/:planId/start", startMathPlan(deps.Plans))
		if deps.Practice != nil {
			plans.POST("/:planId/items/:itemId/answer", answerMathPlanItem(deps.Practice))
			plans.POST("/:planId/finish", finishMathPlan(deps.Practice))
		}
	}
	if deps.Assets != nil && deps.Database != nil {
		router.GET("/api/v1/children/:childId/math/items/:kpId/audio/:file", serveMathLearningAudio(deps.Database, deps.Assets))
		router.GET("/api/v1/children/:childId/math/plans/:planId/items/:itemId/audio.mp3", serveMathPlanAudio(deps.Database, deps.Assets))
	}
	if deps.Quiz != nil {
		router.POST("/api/v1/math/quiz/generate", generateQuiz(deps.Quiz))
	} else {
		router.POST("/api/v1/math/quiz/generate", func(c *gin.Context) {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "出题服务尚未就绪")
		})
	}
	return router
}
