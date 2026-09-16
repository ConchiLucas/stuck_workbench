package httpapi

import (
	"github.com/conchi/study-learning/literacycontract"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/conchi/study-task-admin/internal/taskgen"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (h *handlers) registerPreview(v1 *gin.RouterGroup) {
	v1.GET("/question-types/literacy", func(c *gin.Context) {
		if h.deps.Generation != nil && h.deps.Generation.Support != nil {
			c.JSON(200, h.deps.Generation.Support.Check(c.Request.Context()))
			return
		}
		c.JSON(200, literacycontract.Registry())
	})
	v1.POST("/generation-preview/literacy", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		var in struct {
			KpID         int64  `json:"kpId"`
			QuestionType string `json:"questionType"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.Status(400)
			return
		}
		q, e := h.deps.Generation.BuildPreview(c.Request.Context(), in.KpID, in.QuestionType)
		if e != nil {
			genError(c, e)
			return
		}
		c.JSON(200, q)
	})
	v1.POST("/generation-preview/literacy/answer", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512<<10)
		var in struct {
			Snapshot generation.Snapshot     `json:"snapshot"`
			Response taskgen.PreviewResponse `json:"response"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "预览作答无效或超过512KiB"})
			return
		}
		r, e := h.deps.Generation.EvaluatePreview(c.Request.Context(), in.Snapshot, in.Response)
		if e != nil {
			genError(c, e)
			return
		}
		c.JSON(200, r)
	})
	v1.POST("/question-tasks/:id/items/:seq/preview-answer", func(c *gin.Context) {
		if h.deps.Generation == nil {
			c.Status(503)
			return
		}
		id, e := strconv.ParseInt(c.Param("id"), 10, 64)
		seq, se := strconv.Atoi(c.Param("seq"))
		if e != nil || se != nil || id < 1 || seq < 1 {
			c.Status(400)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512<<10)
		var in struct {
			RevisionID *int64                  `json:"revisionId"`
			Response   taskgen.PreviewResponse `json:"response"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.Status(400)
			return
		}
		t, e := h.deps.Generation.Get(c.Request.Context(), id)
		if e != nil {
			genError(c, e)
			return
		}
		items := t.Items
		if in.RevisionID != nil {
			r, e := h.deps.Generation.Revision(c.Request.Context(), id, *in.RevisionID)
			if e != nil {
				genError(c, e)
				return
			}
			items = r.Items
		}
		if seq > len(items) {
			c.Status(404)
			return
		}
		out, e := h.deps.Generation.EvaluatePreview(c.Request.Context(), items[seq-1].Snapshot, in.Response)
		if e != nil {
			genError(c, e)
			return
		}
		c.JSON(200, out)
	})
}
