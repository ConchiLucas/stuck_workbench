package http

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var phraseMediaFile = regexp.MustCompile(`^[a-f0-9]{64}\.mp3$`)

func phraseTaskMedia(c *gin.Context) {
	file := c.Param("file")
	if strings.Contains(file, "..") || strings.Contains(file, "/") || strings.Contains(file, "\\") {
		c.Status(http.StatusBadRequest)
		return
	}
	if !strings.HasSuffix(file, ".mp3") {
		c.Status(http.StatusNotFound)
		return
	}
	if !phraseMediaFile.MatchString(file) {
		c.Status(http.StatusBadRequest)
		return
	}
	base := strings.TrimRight(os.Getenv("TASK_ADMIN_URL"), "/")
	if base == "" {
		base = "http://localhost:19201"
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		c.Status(http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, base+"/api/v1/phrase/task-media/"+file, nil)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusNotFound {
			c.Status(http.StatusNotFound)
		} else {
			c.Status(http.StatusBadGateway)
		}
		return
	}
	c.Header("Content-Type", res.Header.Get("Content-Type"))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, res.Body)
}
