package http

import (
	"context"
	"errors"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/conchi/study-learning/pinyincontract"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/conchi/pinyin-server/internal/quiz"
)

type Quiz interface {
	Generate(context.Context, string, []int64) (quiz.Question, error)
}

func generateQuiz(service Quiz) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Type             string  `json:"type"`
			ExcludeTargetIDs []int64 `json:"excludeTargetIds"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_request", "请求格式错误")
			return
		}
		question, err := service.Generate(c.Request.Context(), request.Type, request.ExcludeTargetIDs)
		if errors.Is(err, quiz.ErrInvalidType) {
			writeError(c, stdhttp.StatusBadRequest, "invalid_quiz_type", err.Error())
			return
		}
		if errors.Is(err, quiz.ErrNoMaterial) {
			writeError(c, stdhttp.StatusConflict, "no_quiz_material", err.Error())
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		c.Header("Cache-Control", "no-store")
		writeData(c, stdhttp.StatusOK, question)
	}
}

type FormalQuiz interface {
	GenerateForChild(context.Context, int64, string, []int64) (pinyincontract.GeneratedQuestion, error)
	Answer(context.Context, int64, string, pinyincontract.AnswerRequest) (pinyincontract.AnswerResult, error)
	GetInstance(context.Context, int64, string) (pinyincontract.InstanceSnapshot, error)
}

func formalChild(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("childId"), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, 400, "invalid_child_id", "孩子编号无效")
		return 0, false
	}
	return id, true
}
func formalError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, quiz.ErrInvalidType):
		writeError(c, 400, "invalid_quiz_type", err.Error())
	case errors.Is(err, quiz.ErrInvalidRequest):
		writeError(c, 400, "invalid_request", err.Error())
	case errors.Is(err, quiz.ErrNotFound):
		writeError(c, 404, "quiz_not_found", err.Error())
	case errors.Is(err, quiz.ErrConflict):
		writeError(c, 409, "idempotency_conflict", err.Error())
	case errors.Is(err, quiz.ErrNoMaterial):
		writeError(c, 409, "no_quiz_material", err.Error())
	case errors.Is(err, pinyincatalog.ErrIdentityConflict):
		writeError(c, 409, "catalog_identity_conflict", "音节内容身份发生变化，请检查素材")
	case errors.Is(err, quiz.ErrExpired):
		writeError(c, 410, "quiz_expired", err.Error())
	case errors.Is(err, pinyincatalog.ErrUnavailable):
		writeError(c, 503, "catalog_unavailable", "音节内容暂不可用")
	default:
		writeError(c, 503, "service_unavailable", "学习服务暂不可用，请重试")
	}
}
func generateFormalQuiz(service FormalQuiz) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		child, ok := formalChild(c)
		if !ok {
			return
		}
		var in struct {
			Type             string  `json:"type"`
			ExcludeTargetIDs []int64 `json:"excludeTargetIds"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			writeError(c, 400, "invalid_request", "请求格式错误")
			return
		}
		out, err := service.GenerateForChild(c.Request.Context(), child, in.Type, in.ExcludeTargetIDs)
		if err != nil {
			formalError(c, err)
			return
		}
		writeData(c, 200, out)
	}
}
func answerFormalQuiz(service FormalQuiz) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		child, ok := formalChild(c)
		if !ok {
			return
		}
		var in pinyincontract.AnswerRequest
		if err := c.ShouldBindJSON(&in); err != nil {
			writeError(c, 400, "invalid_request", "请求格式错误")
			return
		}
		out, err := service.Answer(c.Request.Context(), child, c.Param("instanceId"), in)
		if err != nil {
			formalError(c, err)
			return
		}
		writeData(c, 200, out)
	}
}
func getFormalQuiz(service FormalQuiz) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		child, ok := formalChild(c)
		if !ok {
			return
		}
		out, err := service.GetInstance(c.Request.Context(), child, c.Param("instanceId"))
		if err != nil {
			formalError(c, err)
			return
		}
		writeData(c, 200, out)
	}
}
