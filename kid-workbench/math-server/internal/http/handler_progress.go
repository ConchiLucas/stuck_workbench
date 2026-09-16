package http

import (
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/conchi/math-server/internal/progress"
)

func getMathProgress(service Progress) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, err := strconv.ParseInt(c.Param("childId"), 10, 64)
		if err != nil || childID <= 0 {
			writeError(c, stdhttp.StatusBadRequest, "invalid_child_id", "孩子编号无效")
			return
		}
		result, err := service.Get(c.Request.Context(), childID)
		if errors.Is(err, progress.ErrChildNotFound) {
			writeError(c, stdhttp.StatusNotFound, "child_not_found", "孩子不存在")
			return
		}
		if err != nil {
			writeDatabaseUnavailable(c)
			return
		}
		writeData(c, stdhttp.StatusOK, result)
	}
}
