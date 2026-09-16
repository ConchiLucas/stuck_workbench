package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func listMathModules(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		modules, err := service.ListModules(c.Request.Context())
		if err != nil {
			writeDatabaseUnavailable(c)
			return
		}
		writeData(c, stdhttp.StatusOK, modules)
	}
}

func getMathModule(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		module, err := service.GetModule(c.Request.Context(), c.Param("moduleCode"))
		if err == gorm.ErrRecordNotFound {
			writeError(c, stdhttp.StatusNotFound, "catalog_not_found", "算数模块不存在")
			return
		}
		if err != nil {
			writeDatabaseUnavailable(c)
			return
		}
		writeData(c, stdhttp.StatusOK, module)
	}
}

func listMathStageItems(service Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.ListStageItems(c.Request.Context(), c.Param("moduleCode"), c.Param("stageCode"))
		if err == gorm.ErrRecordNotFound {
			writeError(c, stdhttp.StatusNotFound, "catalog_not_found", "算数阶段不存在")
			return
		}
		if err != nil {
			writeDatabaseUnavailable(c)
			return
		}
		writeData(c, stdhttp.StatusOK, items)
	}
}
