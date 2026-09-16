package http

import (
	"context"
	"errors"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/conchi/study-science/internal/home"
)

type Home interface {
	Get(context.Context, int64) (home.Summary, error)
}

func getHome(service Home) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		summary, err := service.Get(c.Request.Context(), childID)
		if errors.Is(err, home.ErrChildNotFound) {
			writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, summary)
	}
}
