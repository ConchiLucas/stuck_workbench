package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoemTaskMediaProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://poem.test/api/v1/poem/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.wav", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"audio/wav"}}, Body: io.NopCloser(strings.NewReader("RIFF"))}, nil
	})
	t.Setenv("POEM_SERVER_URL", "http://poem.test")
	r := NewRouter(Deps{})
	ok := "/api/poem/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.wav"
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", ok, 200},
		{"GET", "/api/poem/task-media/not-a-hash.wav", 400},
		{"POST", ok, 404},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code, tc.path)
	}
	require.Equal(t, 1, calls)
}
