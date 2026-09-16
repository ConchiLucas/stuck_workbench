package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-diagnosis-admin/internal/diagnosis"
	httpapi "github.com/conchi/study-diagnosis-admin/internal/http"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
)

func setupRouter(t *testing.T) http.Handler {
	t.Helper()
	return httpapi.NewRouter(httpapi.Deps{Diagnosis: diagnosis.NewService(testdb.Open(t))})
}

func doJSON(t *testing.T, h http.Handler, path string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var out map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &out)
	}
	return w.Code, out
}

func TestHealthz(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, "/healthz")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "ok", body["status"])
}

func TestOverviewHTTP(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, "/api/v1/children/1/diagnosis/overview")
	require.Equal(t, http.StatusOK, code)
	child := body["child"].(map[string]any)
	require.Equal(t, "卢沁一", child["name"])
	require.Contains(t, body["headline"], "识字能认、手写偏弱")
	urgent := body["urgent"].([]any)
	require.NotEmpty(t, urgent)
}

func TestOverviewHTTPMissingChild(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, "/api/v1/children/99/diagnosis/overview")
	require.Equal(t, http.StatusNotFound, code)
	require.Equal(t, "孩子不存在", body["error"])
}

func TestSubjectHTTPOmitsGame(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, "/api/v1/children/1/diagnosis/subjects/game")
	require.Equal(t, http.StatusNotFound, code)
	require.Equal(t, "学科不存在", body["error"])

	code, overview := doJSON(t, h, "/api/v1/children/1/diagnosis/overview")
	require.Equal(t, http.StatusOK, code)
	for _, raw := range overview["subjects"].([]any) {
		s := raw.(map[string]any)
		require.NotEqual(t, "game", s["code"])
	}
}

func TestSubjectHTTP(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, "/api/v1/children/1/diagnosis/subjects/literacy")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "识字", body["name"])
	gaps := body["skill_gaps"].([]any)
	require.NotEmpty(t, gaps)
}

func TestKpArchiveHTTP(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, "/api/v1/children/1/knowledge-points/10")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "一", body["title"])
	require.NotEmpty(t, body["skills"])
}

func TestErrorPatternsHTTP(t *testing.T) {
	h := setupRouter(t)
	code, body := doJSON(t, h, "/api/v1/children/1/error-patterns")
	require.Equal(t, http.StatusOK, code)
	patterns := body["patterns"].([]any)
	require.GreaterOrEqual(t, len(patterns), 3)
}
