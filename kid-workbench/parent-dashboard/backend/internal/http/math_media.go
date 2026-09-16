package http

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func mathPlanSpeech(c *gin.Context) {
	childID, err := strconv.ParseInt(c.Param("childId"), 10, 64)
	planID, err2 := strconv.ParseInt(c.Param("planId"), 10, 64)
	itemID, err3 := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil || err2 != nil || err3 != nil || childID <= 0 || planID <= 0 || itemID <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	base := strings.TrimRight(os.Getenv("MATH_SERVER_URL"), "/")
	if base == "" {
		base = "http://localhost:19141"
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		c.Status(http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, base+"/api/v1/children/"+strconv.FormatInt(childID, 10)+"/math/plans/"+strconv.FormatInt(planID, 10)+"/items/"+strconv.FormatInt(itemID, 10)+"/audio.mp3", nil)
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
