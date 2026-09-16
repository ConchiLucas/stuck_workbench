package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/conchi/literacy-server/internal/progress"
)

type Progress interface {
	Get(context.Context, int64) ([]progress.ItemProgress, error)
}

func getProgress(service Progress) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, err := strconv.ParseInt(c.Param("childId"), 10, 64)
		if err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_child_id", "孩子编号无效")
			return
		}
		items, err := service.Get(c.Request.Context(), childID)
		if errors.Is(err, progress.ErrChildNotFound) {
			writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, items)
	}
}
