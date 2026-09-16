package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMathPlanSpeechProxy(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "http://math.test/api/v1/children/1/math/plans/7/items/9/audio.mp3", r.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"audio/mpeg"}}, Body: io.NopCloser(strings.NewReader("mp3"))}, nil
	})
	t.Setenv("MATH_SERVER_URL", "http://math.test")
	r := NewRouter(Deps{})
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/math/children/1/plans/7/items/9/speech.mp3", 200},
		{"GET", "/api/math/children/bad/plans/7/items/9/speech.mp3", 400},
		{"GET", "/api/math/children/0/plans/7/items/9/speech.mp3", 400},
		{"POST", "/api/math/children/1/plans/7/items/9/speech.mp3", 404},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code, tc.path)
	}
	require.Equal(t, 1, calls)
}
