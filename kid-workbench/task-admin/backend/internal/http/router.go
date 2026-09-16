package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/study-task-admin/internal/chengyutask"
	"github.com/conchi/study-task-admin/internal/englishtask"
	"github.com/conchi/study-task-admin/internal/mathtask"
	"github.com/conchi/study-task-admin/internal/phrasetask"
	"github.com/conchi/study-task-admin/internal/pinyintask"
	"github.com/conchi/study-task-admin/internal/logictask"
	"github.com/conchi/study-task-admin/internal/poemtask"
	"github.com/conchi/study-task-admin/internal/qtask"
	"github.com/conchi/study-task-admin/internal/reviewsuggestion"
	"github.com/conchi/study-task-admin/internal/sciencetask"
	"github.com/conchi/study-task-admin/internal/taskgen"
)

type Deps struct {
	ReviewSuggestions *reviewsuggestion.Service
	Pinyin            *pinyintask.Service
	Math              *mathtask.Service
	English           *englishtask.Service
	Phrase            *phrasetask.Service
	Chengyu           *chengyutask.Service
	Science           *sciencetask.Service
	Poem              *poemtask.Service
	Logic             *logictask.Service
	QTask             *qtask.Service
	Generation        *taskgen.Service
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
	{
		v1.GET("/question-tasks", h.listQuestionTasks)
		v1.POST("/question-tasks", h.createQuestionTask)
		v1.GET("/question-tasks/literacy-modules", h.listQTaskLiteracyModules)
		v1.GET("/question-tasks/:id", h.getQuestionTask)
		v1.POST("/question-tasks/:id/reshuffle", h.reshuffleQuestionTask)
		v1.POST("/question-tasks/:id/publish", h.publishQuestionTask)
		v1.POST("/question-tasks/:id/unpublish", h.unpublishQuestionTask)
		v1.DELETE("/question-tasks/:id", h.deleteQuestionTask)
	}
	h.registerMath(v1)
	h.registerPinyin(v1)
	h.registerEnglish(v1)
	h.registerPhrase(v1)
	h.registerChengyu(v1)
	h.registerScience(v1)
	h.registerPoem(v1)
	h.registerLogic(v1)
	h.registerGeneration(v1)
	h.registerReview(v1)
	h.registerReviewSuggestions(v1)
	mountSPA(r)
	return r
}

type handlers struct{ deps Deps }

func (h *handlers) listQuestionTasks(c *gin.Context) {
	if h.deps.Generation != nil {
		h.listGenerated(c)
		return
	}
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.QTask.List(c.Query("subject"), c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) createQuestionTask(c *gin.Context) {
	if h.createGenerated(c) {
		return
	}
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	var body struct {
		SubjectCode string `json:"subjectCode"`
		ModuleCode  string `json:"moduleCode"`
		Title       string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}
	task, err := h.deps.QTask.Create(qtask.CreateInput{
		SubjectCode: body.SubjectCode,
		ModuleCode:  body.ModuleCode,
		Title:       body.Title,
	})
	writeTask(c, task, err)
}

func (h *handlers) listQTaskLiteracyModules(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.QTask.ListLiteracyModules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) getQuestionTask(c *gin.Context) {
	if h.generated(c) {
		h.generatedAction(c, "get")
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	task, err := h.deps.QTask.Get(id)
	writeTask(c, task, err)
}

func (h *handlers) reshuffleQuestionTask(c *gin.Context) {
	if h.generated(c) {
		c.JSON(409, gin.H{"error": "素材题包请使用生成接口，并携带版本与幂等键"})
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	task, err := h.deps.QTask.Reshuffle(id)
	writeTask(c, task, err)
}

func (h *handlers) publishQuestionTask(c *gin.Context) {
	if h.generated(c) {
		h.generatedAction(c, "publish")
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	task, err := h.deps.QTask.Publish(id)
	writeTask(c, task, err)
}

func (h *handlers) unpublishQuestionTask(c *gin.Context) {
	if h.generated(c) {
		h.generatedAction(c, "unpublish")
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	task, err := h.deps.QTask.Unpublish(id)
	writeTask(c, task, err)
}

func (h *handlers) deleteQuestionTask(c *gin.Context) {
	if h.generated(c) {
		h.generatedAction(c, "delete")
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	err := h.deps.QTask.Delete(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		writeQTaskError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 id"})
		return 0, false
	}
	return id, true
}

func writeTask(c *gin.Context, task qtask.TaskDTO, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		writeQTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func writeQTaskError(c *gin.Context, err error) {
	msg := err.Error()
	status := http.StatusBadRequest
	if strings.Contains(msg, "可出题") || strings.Contains(msg, "draft") || strings.Contains(msg, "published") {
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{"error": msg})
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Idempotency-Key")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
