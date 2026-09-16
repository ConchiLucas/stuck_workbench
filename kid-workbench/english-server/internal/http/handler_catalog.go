package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/english-server/internal/catalog"
)

type Catalog interface {
	ListModules(context.Context) ([]catalog.Module, error)
	ListWords(context.Context, string) ([]catalog.Word, error)
	GetWord(context.Context, int64) (catalog.Word, error)
}

func listModules(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := service.ListModules(c.Request.Context())
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, rows)
	}
}
func listWords(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := service.ListWords(c.Request.Context(), c.Param("moduleCode"))
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, rows)
	}
}
func getWord(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
		if err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_kp_id", "单词编号无效")
			return
		}
		word, err := service.GetWord(c.Request.Context(), id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, stdhttp.StatusNotFound, "content_not_found", "英语单词不存在")
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, word)
	}
}
