package http

import (
	"context"
	"errors"
	"github.com/conchi/english-server/internal/home"
	"github.com/gin-gonic/gin"
	stdhttp "net/http"
	"strconv"
)

type Home interface {
	Get(context.Context, int64) (home.Summary, error)
}

func getHome(s Home) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("childId"), 10, 64)
		if err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_child_id", "孩子编号无效")
			return
		}
		row, err := s.Get(c.Request.Context(), id)
		if errors.Is(err, home.ErrChildNotFound) {
			writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, row)
	}
}
