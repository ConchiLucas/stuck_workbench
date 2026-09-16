package http

import (
	"context"
	"errors"
	stdhttp "net/http"

	"github.com/conchi/chengyu-server/internal/practice"
	"github.com/gin-gonic/gin"
)

type Practice interface {
	Answer(context.Context, int64, int64, int64, practice.AnswerInput) (practice.AnswerResult, error)
	Finish(context.Context, int64, int64) (practice.FinishResult, error)
}

func answerPlanItem(s Practice) gin.HandlerFunc {
	return func(c *gin.Context) {
		child, pid, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		item, ok := routeID(c, "itemId", "invalid_item_id", "题目编号无效")
		if !ok {
			return
		}
		var in practice.AnswerInput
		if c.ShouldBindJSON(&in) != nil || in.ClientID == "" || in.OptionIndex < 0 || in.CostMs < 0 {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "答题内容无效")
			return
		}
		out, err := s.Answer(c.Request.Context(), child, pid, item, in)
		if writePracticeError(c, err) {
			return
		}
		writeData(c, stdhttp.StatusOK, out)
	}
}

func finishPlan(s Practice) gin.HandlerFunc {
	return func(c *gin.Context) {
		child, pid, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		out, err := s.Finish(c.Request.Context(), child, pid)
		if writePracticeError(c, err) {
			return
		}
		writeData(c, stdhttp.StatusOK, out)
	}
}

func writePracticeError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, practice.ErrPlanNotFound):
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "成语题单不存在")
	case errors.Is(err, practice.ErrItemNotFound):
		writeError(c, stdhttp.StatusNotFound, "item_not_found", "题目不在该题单中")
	case errors.Is(err, practice.ErrItemCompleted):
		writeError(c, stdhttp.StatusConflict, "item_completed", "这道题已经完成")
	case errors.Is(err, practice.ErrPlanCompleted):
		writeError(c, stdhttp.StatusConflict, "plan_completed", "题单已经完成")
	case errors.Is(err, practice.ErrIdempotencyConflict):
		writeError(c, stdhttp.StatusConflict, "idempotency_conflict", "该请求编号已用于其他题目")
	case errors.Is(err, practice.ErrPlanIncomplete):
		writeError(c, stdhttp.StatusConflict, "plan_incomplete", "还有题目没有完成")
	default:
		writeInternalError(c)
	}
	return true
}
