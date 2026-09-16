package http

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var frozenRevisionID = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// Only immutable revision media may be read; callers cannot choose an upstream or path.
func literacyRevisionMedia(c *gin.Context) {
	id, kind := c.Param("revision"), c.Param("kind")
	if !frozenRevisionID.MatchString(id) || (kind != "glyph" && kind != "sense" && kind != "speech") {
		c.Status(http.StatusBadRequest)
		return
	}
	base := strings.TrimRight(os.Getenv("CONTENT_ADMIN_URL"), "/")
	if base == "" {
		base = "http://localhost:19091"
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		c.Status(http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, base+"/api/v1/material-revisions/"+id+"/media/"+kind, nil)
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
		if res.StatusCode == 404 {
			c.Status(404)
		} else {
			c.Status(http.StatusBadGateway)
		}
		return
	}
	c.Header("Content-Type", res.Header.Get("Content-Type"))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(200)
	_, _ = io.Copy(c.Writer, res.Body)
}
