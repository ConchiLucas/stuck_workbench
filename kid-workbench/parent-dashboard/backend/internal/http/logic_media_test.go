package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogicTaskMediaProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://logic.test/api/v1/logic/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.svg", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/svg+xml"}}, Body: io.NopCloser(strings.NewReader("<svg/>"))}, nil
	})
	t.Setenv("LOGIC_SERVER_URL", "http://logic.test")
	r := NewRouter(Deps{})
	ok := "/api/logic/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.svg"
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", ok, 200},
		{"GET", "/api/logic/task-media/not-a-hash.svg", 400},
		{"GET", "/api/logic/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png", 400},
		{"POST", ok, 404},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code, tc.path)
	}
	require.Equal(t, 1, calls)
}
