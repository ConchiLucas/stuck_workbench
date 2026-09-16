package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChengyuTaskMediaProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://task.test/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"audio/mpeg"}}, Body: io.NopCloser(strings.NewReader("mp3"))}, nil
	})
	t.Setenv("TASK_ADMIN_URL", "http://task.test")
	r := NewRouter(Deps{})
	ok := "/api/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3"
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", ok, 200},
		{"GET", "/api/chengyu/task-media/not-a-hash.mp3", 400},
		{"GET", "/api/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png", 404},
		{"POST", ok, 404},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code, tc.path)
	}
	require.Equal(t, 1, calls)
}
