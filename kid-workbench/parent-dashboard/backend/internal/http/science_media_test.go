package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScienceTaskMediaProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://science.test/api/v1/science/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/png"}}, Body: io.NopCloser(strings.NewReader("png"))}, nil
	})
	t.Setenv("SCIENCE_SERVER_URL", "http://science.test")
	r := NewRouter(Deps{})
	ok := "/api/science/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", ok, 200},
		{"GET", "/api/science/task-media/not-a-hash.png", 400},
		{"POST", ok, 404},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code, tc.path)
	}
	require.Equal(t, 1, calls)
}
