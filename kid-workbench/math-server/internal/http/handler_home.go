package http

import (
	"errors"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/conchi/math-server/internal/home"
)

func getMathHome(service Home) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, ok := positiveID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		result, err := service.Get(c.Request.Context(), childID)
		if errors.Is(err, home.ErrChildNotFound) {
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
