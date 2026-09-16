package http

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/conchi/literacy-server/internal/plan"
	"github.com/gin-gonic/gin"
	"io"
	stdhttp "net/http"
	"os"
	"strings"
	"time"
)

type Tasks interface {
	ListTasks(context.Context, int64) (plan.TaskList, error)
	ClaimTask(context.Context, int64, int64, plan.ClaimInput) (plan.Detail, error)
}

func listTasks(s Tasks) gin.HandlerFunc {
	return func(c *gin.Context) {
		child, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		result, err := s.ListTasks(c.Request.Context(), child)
		if err != nil {
			writePlanResult(c, plan.Detail{}, err, 200)
			return
		}
		writeData(c, 200, result)
	}
}
func claimTask(s Tasks) gin.HandlerFunc {
	return func(c *gin.Context) {
		child, ok := routeID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		task, ok := routeID(c, "taskId", "invalid_task_id", "任务编号无效")
		if !ok {
			return
		}
		var input plan.ClaimInput
		if err := c.ShouldBindJSON(&input); err != nil || input.RevisionID <= 0 || strings.TrimSpace(input.ClaimKey) == "" || len(input.ClaimKey) > 120 {
			writeError(c, 400, "invalid_request", "领取请求无效")
			return
		}
		result, err := s.ClaimTask(c.Request.Context(), child, task, input)
		switch {
		case errors.Is(err, plan.ErrTaskConflict):
			writeError(c, 409, "task_conflict", err.Error())
		case errors.Is(err, plan.ErrTaskNotAvailable):
			writeError(c, 404, "task_unavailable", err.Error())
		default:
			writePlanResult(c, result, err, 200)
		}
	}
}
func serveRevisionMedia(c *gin.Context) {
	id, kind := c.Param("revisionId"), c.Param("kind")
	if len(id) != 64 || (kind != "glyph" && kind != "sense" && kind != "speech") {
		writeError(c, 400, "invalid_media", "素材引用无效")
		return
	}
	if _, err := hex.DecodeString(id); err != nil {
		writeError(c, 400, "invalid_media", "素材引用无效")
		return
	}
	base := strings.TrimRight(os.Getenv("APP_CONTENT_ADMIN_URL"), "/")
	if base == "" {
		base = "http://localhost:19091"
	}
	req, err := stdhttp.NewRequestWithContext(c.Request.Context(), "GET", base+"/api/v1/material-revisions/"+id+"/media/"+kind, nil)
	if err != nil {
		writeInternalError(c)
		return
	}
	client := &stdhttp.Client{Timeout: 20 * time.Second, CheckRedirect: func(*stdhttp.Request, []*stdhttp.Request) error { return stdhttp.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		writeError(c, 502, "media_unavailable", "素材暂时无法读取")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			writeError(c, 404, "media_missing", "素材不存在")
		} else {
			writeError(c, 502, "media_unavailable", "素材暂时无法读取")
		}
		return
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024+1))
	if err != nil || len(b) > 20*1024*1024 {
		writeError(c, 502, "media_unavailable", "素材读取失败")
		return
	}
	ct := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if (kind == "speech" && ct != "audio/mpeg") || (kind != "speech" && ct != "image/png" && ct != "image/jpeg") {
		writeError(c, 502, "media_unavailable", "素材类型无效")
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, ct, b)
}
