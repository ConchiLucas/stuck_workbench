package http

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/poem-server/internal/catalog"
	"github.com/conchi/poem-server/internal/plan"
	"github.com/conchi/poem-server/internal/practice"
	"github.com/conchi/poem-server/internal/studyplan"
	"github.com/conchi/study-learning/poemcontent"
)

type Deps struct {
	Pavilions        *plan.Store
	Plans            *studyplan.Service
	Practice         *practice.Service
	Database         *gorm.DB
	AssetsConfigured bool
}

func NewRouter() *gin.Engine {
	return NewRouterWithDeps(Deps{Pavilions: plan.NewStore()})
}

func NewRouterWithStore(store *plan.Store) *gin.Engine {
	return NewRouterWithDeps(Deps{Pavilions: store})
}

func NewRouterWithDeps(deps Deps) *gin.Engine {
	if deps.Pavilions == nil {
		deps.Pavilions = plan.NewStore()
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if deps.Database == nil {
			c.JSON(stdhttp.StatusServiceUnavailable, gin.H{"code": "database_unavailable", "database": "unavailable"})
			return
		}
		sqlDB, err := deps.Database.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			c.JSON(stdhttp.StatusServiceUnavailable, gin.H{"code": "database_unavailable", "database": "unavailable"})
			return
		}
		c.JSON(stdhttp.StatusOK, gin.H{"status": "ready", "database": "ready"})
	})
	api := router.Group("/api/v1/poem")
	api.GET("/pavilions", listPavilions)
	api.GET("/pavilions/:code", getPavilion)
	api.GET("/pavilions/:code/clues", listClues)
	api.GET("/items/:kpId/speech/:file", serveLiveSpeech(deps.Database))
	if deps.Plans != nil {
		api.GET("/task-media/:file", serveTaskMedia(deps.Plans))
	}
	child := router.Group("/api/v1/children/:childId/poem")
	child.GET("/home", getHome)
	child.POST("/plans", createPlan(deps))
	child.GET("/plans/:planId", getPlan(deps))
	child.POST("/plans/:planId/start", startPlan(deps))
	child.POST("/plans/:planId/items/:itemId/answer", answerPlan(deps))
	child.POST("/plans/:planId/finish", finishPlan(deps))
	return router
}

func listPavilions(c *gin.Context) {
	writeData(c, stdhttp.StatusOK, catalog.Pavilions())
}

func getPavilion(c *gin.Context) {
	row, ok := catalog.PavilionByCode(c.Param("code"))
	if !ok {
		writeError(c, stdhttp.StatusNotFound, "pavilion_not_found", "亭台不存在")
		return
	}
	writeData(c, stdhttp.StatusOK, row)
}

func listClues(c *gin.Context) {
	code := c.Param("code")
	if _, ok := catalog.PavilionByCode(code); !ok {
		writeError(c, stdhttp.StatusNotFound, "pavilion_not_found", "亭台不存在")
		return
	}
	writeData(c, stdhttp.StatusOK, catalog.Clues(code))
}

func getHome(c *gin.Context) {
	writeData(c, stdhttp.StatusOK, gin.H{"title": "小儿诗园", "pavilions": catalog.Pavilions()})
}

type createBody struct {
	PavilionCode string   `json:"pavilionCode"`
	Mode         string   `json:"mode"`
	Count        int      `json:"count"`
	Types        []string `json:"types"`
}

type answerBody struct {
	AnswerID    string   `json:"answerId"`
	SelectedID  string   `json:"selectedId"`
	Sequence    []string `json:"sequence"`
	ClientID    string   `json:"clientId"`
	OptionIndex int      `json:"optionIndex"`
	CostMs      int      `json:"costMs"`
}

func createPlan(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, ok := parseID(c, "childId")
		if !ok {
			return
		}
		var body createBody
		if c.Request.ContentLength > 0 && json.NewDecoder(c.Request.Body).Decode(&body) != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "请求内容无效")
			return
		}
		if strings.TrimSpace(body.PavilionCode) != "" {
			detail, err := deps.Pavilions.Create(childID, body.PavilionCode)
			if errors.Is(err, plan.ErrPavilionNotFound) {
				writeError(c, stdhttp.StatusNotFound, "pavilion_not_found", "亭台不存在")
				return
			}
			if err != nil {
				writeInternalError(c)
				return
			}
			writeData(c, stdhttp.StatusOK, detail)
			return
		}
		if deps.Plans == nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_pavilion", "亭台编号无效")
			return
		}
		detail, err := deps.Plans.Create(c.Request.Context(), childID, studyplan.CreateInput{Mode: body.Mode, Count: body.Count, Types: body.Types})
		writeDBPlan(c, detail, err, stdhttp.StatusCreated)
	}
}

func getPlan(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := parseChildPlan(c)
		if !ok {
			return
		}
		if detail, err := deps.Pavilions.Get(childID, planID); err == nil {
			writeData(c, stdhttp.StatusOK, detail)
			return
		}
		if deps.Plans == nil {
			writeError(c, stdhttp.StatusNotFound, "plan_not_found", "计划不存在")
			return
		}
		detail, err := deps.Plans.Get(c.Request.Context(), childID, planID)
		writeDBPlan(c, detail, err, stdhttp.StatusOK)
	}
}

func startPlan(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := parseChildPlan(c)
		if !ok {
			return
		}
		if detail, err := deps.Pavilions.Start(childID, planID); err == nil {
			writeData(c, stdhttp.StatusOK, detail)
			return
		}
		if deps.Plans == nil {
			writeError(c, stdhttp.StatusNotFound, "plan_not_found", "计划不存在")
			return
		}
		detail, err := deps.Plans.Start(c.Request.Context(), childID, planID)
		writeDBPlan(c, detail, err, stdhttp.StatusOK)
	}
}

func answerPlan(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := parseChildPlan(c)
		if !ok {
			return
		}
		itemID, ok := parseID(c, "itemId")
		if !ok {
			return
		}
		var body answerBody
		if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_answer", "答案无效")
			return
		}
		if _, err := deps.Pavilions.Get(childID, planID); err == nil {
			answerID := strings.TrimSpace(body.AnswerID)
			if answerID == "" {
				answerID = strings.TrimSpace(body.SelectedID)
			}
			if answerID == "" {
				writeError(c, stdhttp.StatusBadRequest, "invalid_answer", "答案无效")
				return
			}
			result, err := deps.Pavilions.Answer(childID, planID, itemID, answerID)
			if errors.Is(err, plan.ErrPlanNotFound) {
				writeError(c, stdhttp.StatusNotFound, "plan_not_found", "计划不存在")
				return
			}
			if errors.Is(err, plan.ErrItemNotFound) {
				writeError(c, stdhttp.StatusNotFound, "item_not_found", "题目不存在")
				return
			}
			if err != nil {
				writeInternalError(c)
				return
			}
			writeData(c, stdhttp.StatusOK, result)
			return
		}
		if deps.Practice == nil {
			writeError(c, stdhttp.StatusNotFound, "plan_not_found", "计划不存在")
			return
		}
		if strings.TrimSpace(body.ClientID) == "" || body.CostMs < 0 || body.CostMs > 3600000 {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "答题内容无效")
			return
		}
		structured := strings.TrimSpace(body.SelectedID) != "" || strings.TrimSpace(body.AnswerID) != "" || len(body.Sequence) > 0
		if !structured && (body.OptionIndex < 0 || body.OptionIndex > 32) {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "答题内容无效")
			return
		}
		result, err := deps.Practice.Answer(c.Request.Context(), childID, planID, itemID, practice.AnswerInput{
			ClientID: body.ClientID, OptionIndex: body.OptionIndex, SelectedID: body.SelectedID,
			AnswerID: body.AnswerID, Sequence: body.Sequence, CostMs: body.CostMs,
		})
		if writePracticeError(c, err) {
			return
		}
		writeData(c, stdhttp.StatusOK, result)
	}
}

func finishPlan(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := parseChildPlan(c)
		if !ok {
			return
		}
		if detail, err := deps.Pavilions.Finish(childID, planID); err == nil {
			writeData(c, stdhttp.StatusOK, detail)
			return
		}
		if deps.Practice == nil {
			writeError(c, stdhttp.StatusNotFound, "plan_not_found", "计划不存在")
			return
		}
		result, err := deps.Practice.Finish(c.Request.Context(), childID, planID)
		if writePracticeError(c, err) {
			return
		}
		writeData(c, stdhttp.StatusOK, result)
	}
}

func serveTaskMedia(service *studyplan.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		file := c.Param("file")
		data, ctype, err := service.FrozenMedia(c.Request.Context(), file)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, stdhttp.StatusNotFound, "asset_missing", "历史媒体不存在")
			return
		}
		if err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_media", "历史媒体编号无效")
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(stdhttp.StatusOK, ctype, data)
	}
}

func serveLiveSpeech(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "database_unavailable", "数据库未配置")
			return
		}
		kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
		if err != nil || kpID <= 0 {
			writeError(c, stdhttp.StatusBadRequest, "invalid_kp_id", "作品编号无效")
			return
		}
		file := strings.TrimSuffix(c.Param("file"), ".wav")
		file = strings.TrimSuffix(file, ".mp3")
		ord, ok := poemcontent.ParseLiveSpeech(file)
		if !ok {
			writeError(c, stdhttp.StatusBadRequest, "invalid_speech", "诗行读音编号无效")
			return
		}
		data, mime, err := poemcontent.ItemSpeech(db.WithContext(c.Request.Context()), kpID, ord)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, stdhttp.StatusNotFound, "asset_missing", "读音素材暂不可用")
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(stdhttp.StatusOK, mime, data)
	}
}

func writePracticeError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, practice.ErrPlanNotFound):
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "计划不存在")
	case errors.Is(err, practice.ErrItemNotFound):
		writeError(c, stdhttp.StatusNotFound, "item_not_found", "题目不存在")
	case errors.Is(err, practice.ErrItemCompleted):
		writeError(c, stdhttp.StatusConflict, "item_completed", "这道题已经完成")
	case errors.Is(err, practice.ErrPlanIncomplete):
		writeError(c, stdhttp.StatusConflict, "plan_incomplete", "还有题目没有完成")
	case errors.Is(err, practice.ErrClientIDConflict):
		writeError(c, stdhttp.StatusConflict, "client_id_conflict", "该答题请求编号已用于另一道题")
	default:
		writeInternalError(c)
	}
	return true
}

func writeDBPlan(c *gin.Context, detail studyplan.Detail, err error, status int) {
	switch {
	case errors.Is(err, studyplan.ErrChildNotFound):
		writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
	case errors.Is(err, studyplan.ErrPlanNotFound):
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "计划不存在")
	case errors.Is(err, studyplan.ErrNoQuestions):
		writeError(c, stdhttp.StatusConflict, "no_questions", "暂无可用古诗题目")
	case errors.Is(err, studyplan.ErrInvalidMode):
		writeError(c, stdhttp.StatusBadRequest, "invalid_plan_mode", "题单模式无效")
	case err != nil:
		writeInternalError(c)
	default:
		writeData(c, status, detail)
	}
}

func parseChildPlan(c *gin.Context) (int64, int64, bool) {
	childID, ok := parseID(c, "childId")
	if !ok {
		return 0, 0, false
	}
	planID, ok := parseID(c, "planId")
	if !ok {
		return 0, 0, false
	}
	return childID, planID, true
}

func parseID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, stdhttp.StatusBadRequest, "invalid_id", "编号无效")
		return 0, false
	}
	return id, true
}

func writeInternalError(c *gin.Context) {
	writeError(c, stdhttp.StatusInternalServerError, "internal_error", "服务暂时不可用")
}