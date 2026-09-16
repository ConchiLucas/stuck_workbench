package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPinyinMediaProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://content.test/api/v1/pinyin/items/12/speech/solo.mp3", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"audio/mpeg"}}, Body: io.NopCloser(strings.NewReader("mp3"))}, nil
	})
	t.Setenv("CONTENT_ADMIN_URL", "http://content.test")
	r := NewRouter(Deps{})
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/pinyin/items/12/speech/solo.mp3?url=bad", 200},
		{"GET", "/api/pinyin/items/bad/speech/solo.mp3", 400},
		{"GET", "/api/pinyin/items/12/speech/word-9.mp3", 400},
		{"GET", "/api/pinyin/items/0/speech/solo.mp3", 400},
		{"POST", "/api/pinyin/items/12/speech/solo.mp3", 404},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code, tc.path)
	}
	require.Equal(t, 1, calls)
}

func TestPinyinGlyphProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://content.test/api/v1/pinyin/items/7/glyph.png", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/png"}}, Body: io.NopCloser(strings.NewReader("png"))}, nil
	})
	t.Setenv("CONTENT_ADMIN_URL", "http://content.test")
	r := NewRouter(Deps{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/pinyin/items/7/glyph.png", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, 1, calls)
}

func TestPinyinSyllableMediaProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "content.test", r.URL.Host)
		require.Equal(t, "/api/v1/pinyin/syllables/12/speech.mp3", r.URL.Path)
		if calls == 2 {
			require.Equal(t, "v=0123456789abcdef", r.URL.RawQuery)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"audio/mpeg"}}, Body: io.NopCloser(strings.NewReader("mp3"))}, nil
	})
	t.Setenv("CONTENT_ADMIN_URL", "http://content.test")
	r := NewRouter(Deps{})
	for _, tc := range []struct {
		path   string
		status int
	}{{"/api/pinyin/syllables/12/speech.mp3", 200}, {"/api/pinyin/syllables/12/speech.mp3?v=0123456789abcdef", 200}, {"/api/pinyin/syllables/12/speech.mp3?v=bad", 400}, {"/api/pinyin/syllables/0/speech.mp3", 400}, {"/api/pinyin/syllables/bad/speech.mp3", 400}, {"/api/pinyin/syllables/12/other.mp3", 404}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		require.Equal(t, tc.status, w.Code)
	}
	require.Equal(t, 2, calls)
}
