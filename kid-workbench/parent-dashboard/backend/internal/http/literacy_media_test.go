package http

import (
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFrozenMediaProxy(t *testing.T) {
	id := strings.Repeat("a", 64)
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://content.test/api/v1/material-revisions/"+id+"/media/sense", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/png"}}, Body: io.NopCloser(strings.NewReader("frozen"))}, nil
	})
	t.Setenv("CONTENT_ADMIN_URL", "http://content.test")
	r := NewRouter(Deps{})
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/api/literacy/material-revisions/" + id + "/media/sense?url=bad", 200}, {"GET", "/api/literacy/material-revisions/bad/media/sense", 400}, {"GET", "/api/literacy/material-revisions/" + id + "/media/other", 400}, {"POST", "/api/literacy/material-revisions/" + id + "/media/sense", 404}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code)
	}
	require.Equal(t, 1, calls)
}

type mediaTransport func(*http.Request) (*http.Response, error)

func (f mediaTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
