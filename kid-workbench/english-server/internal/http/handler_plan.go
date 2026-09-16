package http

import (
	"context"
	"errors"
	"github.com/conchi/english-server/internal/plan"
	"github.com/gin-gonic/gin"
	stdhttp "net/http"
	"strconv"
)

type Plans interface {
	Create(context.Context, int64, plan.CreateInput) (plan.Detail, error)
	Get(context.Context, int64, int64) (plan.Detail, error)
	Start(context.Context, int64, int64) (plan.Detail, error)
}

func createPlan(s Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		child, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		var in plan.CreateInput
		if c.ShouldBindJSON(&in) != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "请求内容无效")
			return
		}
		out, err := s.Create(c.Request.Context(), child, in)
		writePlanResult(c, out, err, stdhttp.StatusCreated)
	}
}
func getPlan(s Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		child, pid, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		out, err := s.Get(c.Request.Context(), child, pid)
		writePlanResult(c, out, err, stdhttp.StatusOK)
	}
}
func startPlan(s Plans) gin.HandlerFunc {
	return func(c *gin.Context) {
		child, pid, ok := childAndPlanIDs(c)
		if !ok {
			return
		}
		out, err := s.Start(c.Request.Context(), child, pid)
		writePlanResult(c, out, err, stdhttp.StatusOK)
	}
}
func childAndPlanIDs(c *gin.Context) (int64, int64, bool) {
	child, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
	if !ok {
		return 0, 0, false
	}
	pid, ok := routeID(c, "planId", "invalid_plan_id", "题单编号无效")
	return child, pid, ok
}
func routeID(c *gin.Context, key, code, message string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, stdhttp.StatusBadRequest, code, message)
		return 0, false
	}
	return id, true
}
func writePlanResult(c *gin.Context, out plan.Detail, err error, status int) {
	switch {
	case errors.Is(err, plan.ErrChildNotFound):
		writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
	case errors.Is(err, plan.ErrPlanNotFound):
		writeError(c, stdhttp.StatusNotFound, "plan_not_found", "英语题单不存在")
	case errors.Is(err, plan.ErrNoQuestions):
		writeError(c, stdhttp.StatusConflict, "no_eligible_questions", "暂无素材完整的英语题目")
	case err != nil:
		writeInternalError(c)
	default:
		writeData(c, status, out)
	}
}
