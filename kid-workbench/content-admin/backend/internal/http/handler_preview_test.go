package httpapi_test

import (
	httpapi "github.com/conchi/study-content-admin/internal/http"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewProxyFixedPathsAndLimits(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.True(t, r.URL.Path == "/api/v1/generation-preview/literacy" || r.URL.Path == "/api/v1/generation-preview/literacy/answer")
		b, _ := io.ReadAll(r.Body)
		require.Equal(t, `{"kpId":1}`, string(b))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(422)
		w.Write([]byte(`{"error":"missing_template"}`))
	}))
	defer upstream.Close()
	t.Setenv("APP_TASK_ADMIN_URL", upstream.URL)
	router := httpapi.NewRouter(httpapi.Deps{})
	for _, path := range []string{"/api/v1/generation-preview/literacy", "/api/v1/generation-preview/literacy/answer"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader(`{"kpId":1}`)))
		require.Equal(t, 422, rec.Code)
		require.Contains(t, rec.Body.String(), "missing_template")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/generation-preview/literacy", strings.NewReader(strings.Repeat("x", (512<<10)+1))))
	require.Equal(t, 413, rec.Code)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/question-types/literacy", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"contractVersion":2`)
}
