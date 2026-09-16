package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Deps struct {
	Catalog          Catalog
	Assets           Assets
	Progress         Progress
	Plans            Plans
	Practice         Practice
	Home             Home
	Quiz             Quiz
	Database         *gorm.DB
	AssetsConfigured bool
}

func NewRouter(deps Deps) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if deps.Database == nil {
			c.JSON(stdhttp.StatusServiceUnavailable, gin.H{"code": "database_unavailable", "database": "unavailable", "assets": deps.AssetsConfigured})
			return
		}
		sqlDB, err := deps.Database.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			c.JSON(stdhttp.StatusServiceUnavailable, gin.H{"code": "database_unavailable", "database": "unavailable", "assets": deps.AssetsConfigured})
			return
		}
		c.JSON(stdhttp.StatusOK, gin.H{"status": "ready", "database": "ready", "assets": deps.AssetsConfigured})
	})
	if deps.Catalog != nil {
		api := router.Group("/api/v1/science")
		api.GET("/modules", listModules(deps.Catalog))
		api.GET("/modules/:moduleCode/items", listItems(deps.Catalog))
		api.GET("/items/:kpId", getItem(deps.Catalog))
	}
	if deps.Assets != nil {
		api := router.Group("/api/v1/science/items/:kpId")
		api.GET("/glyph.png", serveGlyph(deps.Assets))
		api.GET("/sense.png", serveSense(deps.Assets))
		api.GET("/speech.mp3", serveSpeech(deps.Assets))
		api.GET("/diagram.png", serveDiagram(deps.Database))
	} else if deps.Database != nil {
		router.GET("/api/v1/science/items/:kpId/diagram.png", serveDiagram(deps.Database))
	}
	if deps.Plans != nil {
		router.GET("/api/v1/science/task-media/:file", serveTaskMedia(deps.Plans))
	}
	if deps.Progress != nil {
		router.GET("/api/v1/children/:childId/science/progress", getProgress(deps.Progress))
	}
	if deps.Home != nil {
		router.GET("/api/v1/children/:childId/science/home", getHome(deps.Home))
	}
	if deps.Plans != nil {
		plans := router.Group("/api/v1/children/:childId/science/plans")
		plans.POST("", createPlan(deps.Plans))
		plans.GET("/:planId", getPlan(deps.Plans))
		plans.POST("/:planId/start", startPlan(deps.Plans))
		if deps.Practice != nil {
			plans.POST("/:planId/items/:itemId/answer", answerPlanItem(deps.Practice))
			plans.POST("/:planId/finish", finishPlan(deps.Practice))
		}
	}
	if deps.Quiz != nil {
		router.POST("/api/v1/science/quiz/generate", generateQuiz(deps.Quiz))
	} else {
		router.POST("/api/v1/science/quiz/generate", func(c *gin.Context) {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "出题服务尚未就绪")
		})
	}
	return router
}
