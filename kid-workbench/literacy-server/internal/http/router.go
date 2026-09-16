package http

import (
	"github.com/conchi/study-learning/literacycontract"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	Catalog  Catalog
	Assets   Assets
	Progress Progress
	Plans    Plans
	Practice Practice
	Home     Home
}

func NewRouter(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/api/v1/question-types/literacy", func(c *gin.Context) {
		contract := literacycontract.Registry()
		writeData(c, 200, gin.H{"contractVersion": contract.ContractVersion, "schemaVersions": []int{1, 2}, "responseSchemaVersion": 2, "types": contract.Types})
	})
	if deps.Catalog != nil {
		api := router.Group("/api/v1/literacy")
		api.GET("/modules", listModules(deps.Catalog))
		api.GET("/modules/:moduleCode/items", listItems(deps.Catalog))
		api.GET("/items/:kpId", getItem(deps.Catalog))
	}
	if deps.Assets != nil {
		api := router.Group("/api/v1/literacy/items/:kpId")
		api.GET("/glyph.png", serveGlyph(deps.Assets))
		api.GET("/sense.png", serveSense(deps.Assets))
		api.GET("/speech.mp3", serveSpeech(deps.Assets))
	}
	if deps.Progress != nil {
		router.GET("/api/v1/children/:childId/literacy/progress", getProgress(deps.Progress))
	}
	if deps.Home != nil {
		router.GET("/api/v1/children/:childId/literacy/home", getHome(deps.Home))
	}
	router.GET("/api/v1/literacy/material-revisions/:revisionId/media/:kind", serveRevisionMedia)
	if tasks, ok := deps.Plans.(Tasks); ok {
		router.GET("/api/v1/children/:childId/literacy/question-tasks", listTasks(tasks))
		router.POST("/api/v1/children/:childId/literacy/question-tasks/:taskId/claim", claimTask(tasks))
	}
	if deps.Plans != nil {
		plans := router.Group("/api/v1/children/:childId/literacy/plans")
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
