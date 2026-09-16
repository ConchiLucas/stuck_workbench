package http

import (
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/conchi/math-server/internal/plan"
	"github.com/conchi/math-server/internal/practice"
)

func createMathPlan(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, ok := positiveID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		var input plan.CreateInput
		if c.ShouldBindJSON(&input) != nil || plan.ValidateCreateInput(input) != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_plan_scope", "练习范围无效")
			return
		}
		detail, err := service.Create(c.Request.Context(), childID, input)
		if errors.Is(err, plan.ErrChildNotFound) {
			writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
			return
		}
		if errors.Is(err, plan.ErrNoQuestions) {
			writeError(c, stdhttp.StatusConflict, "no_questions", "暂无可练习的算数题目")
			return
		}
		if errors.Is(err, plan.ErrInvalidScope) {
			writeError(c, stdhttp.StatusBadRequest, "invalid_plan_scope", "练习范围无效")
			return
		}
		if err != nil {
			writeDatabaseUnavailable(c)
			return
		}
		writeData(c, stdhttp.StatusCreated, detail)
	}
}

func answerMathPlanItem(service Practice) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := planIDs(c)
		if !ok {
			return
		}
		itemID, ok := positiveID(c, "itemId", "invalid_item_id", "题目编号无效")
		if !ok {
			return
		}
		var input practice.AnswerInput
		if c.ShouldBindJSON(&input) != nil || input.ClientID == "" || input.OptionIndex < 0 || input.OptionIndex > 3 || input.CostMs < 0 || input.CostMs > 3600000 {
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

func finishMathPlan(service Practice) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := planIDs(c)
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
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "练习计划不存在")
	case errors.Is(err, practice.ErrItemNotFound):
		writeError(c, stdhttp.StatusNotFound, "item_not_found", "题目不在该计划中")
	case errors.Is(err, practice.ErrItemCompleted):
		writeError(c, stdhttp.StatusConflict, "item_completed", "这道题已经完成")
	case errors.Is(err, practice.ErrIdempotencyConflict):
		writeError(c, stdhttp.StatusConflict, "idempotency_conflict", "该答题请求编号已用于其他题目")
	case errors.Is(err, practice.ErrPlanIncomplete):
		writeError(c, stdhttp.StatusConflict, "plan_incomplete", "还有题目没有完成")
	default:
		writeDatabaseUnavailable(c)
	}
	return true
}

func getMathPlan(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := planIDs(c)
		if !ok {
			return
		}
		detail, err := service.Get(c.Request.Context(), childID, planID)
		writePlanResult(c, detail, err)
	}
}

func startMathPlan(service Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := planIDs(c)
		if !ok {
			return
		}
		detail, err := service.Start(c.Request.Context(), childID, planID)
		writePlanResult(c, detail, err)
	}
}

func writePlanResult(c *gin.Context, detail plan.Detail, err error) {
	if errors.Is(err, plan.ErrPlanNotFound) {
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "练习计划不存在")
		return
	}
	if err != nil {
		writeDatabaseUnavailable(c)
		return
	}
	writeData(c, stdhttp.StatusOK, detail)
}

func positiveID(c *gin.Context, param, code, message string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, stdhttp.StatusBadRequest, code, message)
		return 0, false
	}
	return id, true
}

func planIDs(c *gin.Context) (int64, int64, bool) {
	childID, ok := positiveID(c, "childId", "invalid_child_id", "孩子编号无效")
	if !ok {
		return 0, 0, false
	}
	planID, ok := positiveID(c, "planId", "invalid_plan_id", "计划编号无效")
	return childID, planID, ok
}
