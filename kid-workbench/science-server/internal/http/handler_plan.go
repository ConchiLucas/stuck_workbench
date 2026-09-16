package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/conchi/study-science/internal/plan"
	"github.com/conchi/study-science/internal/practice"
)

type Plans interface {
	Create(context.Context, int64, plan.CreateInput) (plan.Detail, error)
	Get(context.Context, int64, int64) (plan.Detail, error)
	Start(context.Context, int64, int64) (plan.Detail, error)
	FrozenMedia(context.Context, string) ([]byte, string, error)
}

type Practice interface {
	Answer(context.Context, int64, int64, int64, practice.AnswerInput) (practice.AnswerResult, error)
	Finish(context.Context, int64, int64) (plan.StudyPlan, error)
}

func createPlan(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		var input plan.CreateInput
		if c.Request.ContentLength > 0 && c.ShouldBindJSON(&input) != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "请求内容无效")
			return
		}
		detail, err := service.Create(c.Request.Context(), childID, input)
		writePlanResult(c, detail, err, stdhttp.StatusCreated)
	}
}

func getPlan(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		detail, err := service.Get(c.Request.Context(), childID, planID)
		writePlanResult(c, detail, err, stdhttp.StatusOK)
	}
}

func startPlan(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		detail, err := service.Start(c.Request.Context(), childID, planID)
		writePlanResult(c, detail, err, stdhttp.StatusOK)
	}
}

func answerPlanItem(service Practice) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		itemID, ok := routeID(c, "itemId", "invalid_item_id", "题目编号无效")
		if !ok {
			return
		}
		var input practice.AnswerInput
		if err := c.ShouldBindJSON(&input); err != nil || input.ClientID == "" || input.CostMs < 0 || input.CostMs > 3600000 {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "答题内容无效")
			return
		}
		structured := strings.TrimSpace(input.SelectedID) != "" || len(input.Pairs) > 0 || len(input.Sequence) > 0 || len(input.Labels) > 0
		if !structured && (input.OptionIndex < 0 || input.OptionIndex > 12) {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "答题内容无效")
			return
		}
		result, err := service.Answer(c.Request.Context(), childID, planID, itemID, input)
		if writePracticeError(c, err) {
			return
		}
		writeData(c, stdhttp.StatusOK, result)
	}
}

func finishPlan(service Practice) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		result, err := service.Finish(c.Request.Context(), childID, planID)
		if writePracticeError(c, err) {
			return
		}
		writeData(c, stdhttp.StatusOK, result)
	}
}

func writePracticeError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, practice.ErrPlanNotFound):
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "科普题单不存在")
	case errors.Is(err, practice.ErrItemNotFound):
		writeError(c, stdhttp.StatusNotFound, "item_not_found", "题目不在该题单中")
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

func childAndPlanIDs(c *gin.Context) (int64, int64, bool) {
	childID, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
	if !ok {
		return 0, 0, false
	}
	planID, ok := routeID(c, "planId", "invalid_plan_id", "题单编号无效")
	return childID, planID, ok
}

func routeID(c *gin.Context, parameter, code, message string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(parameter), 10, 64)
	if err != nil || value <= 0 {
		writeError(c, stdhttp.StatusBadRequest, code, message)
		return 0, false
	}
	return value, true
}

func writePlanResult(c *gin.Context, detail plan.Detail, err error, status int) {
	switch {
	case errors.Is(err, plan.ErrChildNotFound):
		writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
	case errors.Is(err, plan.ErrPlanNotFound):
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "科普题单不存在")
	case errors.Is(err, plan.ErrNoQuestions):
		writeError(c, stdhttp.StatusConflict, "no_questions", "暂无可用科普题目")
	case errors.Is(err, plan.ErrInvalidMode):
		writeError(c, stdhttp.StatusBadRequest, "invalid_plan_mode", "题单模式无效")
	case errors.Is(err, plan.ErrModuleNotFound):
		writeError(c, stdhttp.StatusNotFound, "module_not_found", "科普主题不存在或没有已发布题目")
	case err != nil:
		writeInternalError(c)
	default:
		writeData(c, status, detail)
	}
}
