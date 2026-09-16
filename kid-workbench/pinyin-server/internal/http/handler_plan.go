package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/conchi/pinyin-server/internal/plan"
	"github.com/conchi/pinyin-server/internal/practice"
)

type Plans interface {
	Create(context.Context, int64, int) (plan.Detail, error)
	Get(context.Context, int64, int64) (plan.Detail, error)
	Start(context.Context, int64, int64) (plan.Detail, error)
}

type Practice interface {
	Answer(context.Context, int64, int64, int64, practice.AnswerInput) (practice.AnswerResult, error)
	Finish(context.Context, int64, int64) (plan.StudyPlan, error)
}

type createPlanInput struct {
	Count int `json:"count"`
}

func createPlan(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		var input createPlanInput
		if c.Request.ContentLength > 0 && c.ShouldBindJSON(&input) != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "请求内容无效")
			return
		}
		detail, err := service.Create(c.Request.Context(), childID, input.Count)
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
		if err := c.ShouldBindJSON(&input); err != nil || input.ClientID == "" || input.OptionIndex < 0 || input.OptionIndex > 20 || input.CostMs < 0 || input.CostMs > 3600000 {
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
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "拼音题单不存在")
	case errors.Is(err, practice.ErrItemNotFound):
		writeError(c, stdhttp.StatusNotFound, "item_not_found", "题目不在该题单中")
	case errors.Is(err, practice.ErrItemCompleted):
		writeError(c, stdhttp.StatusConflict, "item_completed", "这道题已经完成")
	case errors.Is(err, practice.ErrIdempotencyConflict):
		writeError(c, stdhttp.StatusConflict, "idempotency_conflict", "该答题请求编号已用于其他题目")
	case errors.Is(err, practice.ErrPlanIncomplete):
		writeError(c, stdhttp.StatusConflict, "plan_incomplete", "还有题目没有完成")
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
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "拼音题单不存在")
	case errors.Is(err, plan.ErrNoQuestions):
		writeError(c, stdhttp.StatusConflict, "no_questions", "暂无可用拼音题目")
	case err != nil:
		writeInternalError(c)
	default:
		writeData(c, status, detail)
	}
}
