package http

import (
	"context"
	"errors"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/conchi/math-server/internal/quiz"
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
