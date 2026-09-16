package httpapi

import (
	"github.com/conchi/study-task-admin/internal/taskgen"
	"github.com/gin-gonic/gin"
	"strconv"
)

func (h *handlers) registerReview(v1 *gin.RouterGroup) {
	v1.GET("/question-tasks/:id/review-evidence", h.reviewEvidence)
	v1.GET("/question-tasks/:id/review-jobs", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		var jobs []taskgen.ReviewJob
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		if e := h.deps.Generation.DB.Where("source_task_id = ?", id).Order("id DESC").Find(&jobs).Error; e != nil {
			genError(c, e)
		} else {
			c.JSON(200, jobs)
		}
	})
	v1.POST("/review-generation-jobs/:jobId/retry", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		id, _ := strconv.ParseInt(c.Param("jobId"), 10, 64)
		if e := h.deps.Generation.RetryReviewJob(c.Request.Context(), id); e != nil {
			genError(c, e)
		} else {
			c.Status(204)
		}
	})
	v1.GET("/question-tasks/:id/review-sources", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		child, _ := strconv.ParseInt(c.Query("childId"), 10, 64)
		rows, e := h.deps.Generation.ReviewPlans(c.Request.Context(), id, child)
		if e != nil {
			genError(c, e)
		} else {
			c.JSON(200, rows)
		}
	})
	for path, create := range map[string]bool{"/question-tasks/:id/review-preview": false, "/question-tasks/:id/review-tasks": true} {
		isCreate := create
		v1.POST(path, func(c *gin.Context) {
			if h.deps.Generation == nil {
				c.Status(503)
				return
			}
			id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			var in taskgen.ReviewInput
			if e := c.ShouldBindJSON(&in); e != nil {
				c.JSON(400, gin.H{"error": "无效复习请求"})
				return
			}
			if isCreate {
				t, e := h.deps.Generation.CreateReview(c.Request.Context(), id, in)
				if e != nil {
					genError(c, e)
				} else {
					c.JSON(201, t)
				}
			} else {
				p, e := h.deps.Generation.PreviewReview(c.Request.Context(), id, in)
				if e != nil {
					genError(c, e)
				} else {
					c.JSON(200, p)
				}
			}
		})
	}
}
