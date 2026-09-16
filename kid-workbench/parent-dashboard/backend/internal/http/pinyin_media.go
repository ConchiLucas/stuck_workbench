package http

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func validPinyinSpeechFile(file string) bool {
	if file == "solo.mp3" || file == "word.mp3" {
		return true
	}
	rest, ok := strings.CutPrefix(file, "word-")
	if !ok || !strings.HasSuffix(file, ".mp3") {
		return false
	}
	n := strings.TrimSuffix(rest, ".mp3")
	parsed, err := strconv.Atoi(n)
	return err == nil && parsed >= 1 && parsed <= 4 && strconv.Itoa(parsed) == n
}

func pinyinItemSpeech(c *gin.Context) {
	proxyPinyinMedia(c, "items", c.Param("kpId"), "speech/"+c.Param("file"), validPinyinSpeechFile(c.Param("file")))
}

func pinyinItemGlyph(c *gin.Context) {
	proxyPinyinMedia(c, "items", c.Param("kpId"), "glyph.png", true)
}

var pinyinVersionQuery = regexp.MustCompile(`^v=[0-9a-f]{16}$`)

func pinyinSyllableSpeech(c *gin.Context) {
	proxyPinyinMedia(c, "syllables", c.Param("assetId"), "speech.mp3", true)
}

func proxyPinyinMedia(c *gin.Context, resource, kpID, suffix string, allowed bool) {
	id, err := strconv.ParseInt(kpID, 10, 64)
	if err != nil || id <= 0 || !allowed || strings.Contains(suffix, "..") || strings.Contains(suffix, "//") {
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
	mediaQuery := ""
	if resource == "syllables" && c.Request.URL.RawQuery != "" {
		if !pinyinVersionQuery.MatchString(c.Request.URL.RawQuery) {
			c.Status(http.StatusBadRequest)
			return
		}
		mediaQuery = "?" + c.Request.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, base+"/api/v1/pinyin/"+resource+"/"+kpID+"/"+suffix+mediaQuery, nil)
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
