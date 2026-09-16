package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/study-diagnosis-admin/internal/diagnosis"
	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/conchi/study-diagnosis-admin/internal/reviewclient"
)

type Deps struct {
	Diagnosis *diagnosis.Service
	Knowledge *knowledge.Service
	Reviews   *reviewclient.Client
}

func NewRouter(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), cors())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	h := &handlers{deps: d}
	v1 := r.Group("/api/v1")
	{
		v1.GET("/children/:cid/diagnosis/overview", h.overview)
		v1.GET("/children/:cid/diagnosis/subjects/:code", h.subject)
		v1.GET("/children/:cid/knowledge-points/:kpId", h.kpArchive)
		v1.GET("/children/:cid/error-patterns", h.errorPatterns)
	}
	h.registerKnowledge(v1)
	mountSPA(r)
	return r
}

type handlers struct{ deps Deps }

func (h *handlers) overview(c *gin.Context) {
	svc, ok := h.svc(c)
	if !ok {
		return
	}
	cid, ok := parseID(c, "cid", "无效的孩子 id")
	if !ok {
		return
	}
	out, err := svc.Overview(cid)
	writeJSON(c, out, err, "孩子不存在")
}

func (h *handlers) subject(c *gin.Context) {
	svc, ok := h.svc(c)
	if !ok {
		return
	}
	cid, ok := parseID(c, "cid", "无效的孩子 id")
	if !ok {
		return
	}
	out, err := svc.Subject(cid, c.Param("code"))
	writeJSON(c, out, err, "学科不存在")
}

func (h *handlers) kpArchive(c *gin.Context) {
	svc, ok := h.svc(c)
	if !ok {
		return
	}
	cid, ok := parseID(c, "cid", "无效的孩子 id")
	if !ok {
		return
	}
	kpID, ok := parseID(c, "kpId", "无效的知识点 id")
	if !ok {
		return
	}
	out, err := svc.KpArchive(cid, kpID)
	writeJSON(c, out, err, "知识点不存在")
}

func (h *handlers) errorPatterns(c *gin.Context) {
	svc, ok := h.svc(c)
	if !ok {
		return
	}
	cid, ok := parseID(c, "cid", "无效的孩子 id")
	if !ok {
		return
	}
	patterns, err := svc.ErrorPatterns(cid)
	if err != nil {
		writeJSON(c, nil, err, "孩子不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{"patterns": patterns})
}

func (h *handlers) svc(c *gin.Context) (*diagnosis.Service, bool) {
	if h.deps.Diagnosis == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return nil, false
	}
	return h.deps.Diagnosis, true
}

func parseID(c *gin.Context, name, msg string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return 0, false
	}
	return id, true
}

func writeJSON(c *gin.Context, body any, err error, notFound string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": notFound})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, body)
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
