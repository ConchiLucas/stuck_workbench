package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/study-science/internal/catalog"
)

type Catalog interface {
	ListModules(context.Context) ([]catalog.Module, error)
	ListItems(context.Context, string) ([]catalog.Item, error)
	GetItem(context.Context, int64) (catalog.Item, error)
}

func listModules(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		modules, err := service.ListModules(c.Request.Context())
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, modules)
	}
}

func listItems(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.ListItems(c.Request.Context(), c.Param("moduleCode"))
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, items)
	}
}

func getItem(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
		if err != nil {
			writeError(c, stdhttp.StatusBadRequest, "invalid_kp_id", "科普条目编号无效")
			return
		}
		item, err := service.GetItem(c.Request.Context(), kpID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, stdhttp.StatusNotFound, "item_not_found", "科普条目不存在或尚未发布")
			return
		}
		if err != nil {
			writeInternalError(c)
			return
		}
		writeData(c, stdhttp.StatusOK, item)
	}
}
