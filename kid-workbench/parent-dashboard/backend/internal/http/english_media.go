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

var englishMediaFile = regexp.MustCompile(`^[a-f0-9]{64}\.(mp3|png|jpe?g|webp)$`)
var englishKpID = regexp.MustCompile(`^[1-9][0-9]{0,17}$`)
var englishSentenceID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func englishTaskMedia(c *gin.Context) {
	file := c.Param("file")
	if !englishMediaFile.MatchString(file) || strings.Contains(file, "..") {
		c.Status(http.StatusBadRequest)
		return
	}
	proxyEnglish(c, os.Getenv("TASK_ADMIN_URL"), "http://localhost:19201", "/api/v1/english/task-media/"+file)
}

func englishWordSpeech(c *gin.Context) {
	proxyEnglishWord(c, "speech.mp3")
}

func englishWordSense(c *gin.Context) {
	proxyEnglishWord(c, "sense.png")
}

func englishSentenceSpeech(c *gin.Context) {
	id := c.Param("id")
	if !englishSentenceID.MatchString(id) || strings.Contains(id, "..") {
		c.Status(http.StatusBadRequest)
		return
	}
	proxyEnglish(c, os.Getenv("ENGLISH_SERVER_URL"), "http://localhost:19131", "/api/v1/english/sentences/"+id+"/speech.mp3")
}

func proxyEnglishWord(c *gin.Context, file string) {
	kpID := c.Param("kpId")
	if !englishKpID.MatchString(kpID) {
		c.Status(http.StatusBadRequest)
		return
	}
	proxyEnglish(c, os.Getenv("ENGLISH_SERVER_URL"), "http://localhost:19131", "/api/v1/english/words/"+kpID+"/"+file)
}

func proxyEnglish(c *gin.Context, base, fallback, path string) {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = fallback
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		c.Status(http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, base+path, nil)
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
