package httpapi

import (
	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestKnowledgeHTTPAndChildIsolation(t *testing.T) {
	r := NewRouter(Deps{Knowledge: knowledge.New(testdb.Open(t))})
	for _, path := range []string{"/summary", "/points", "/wrongs", "/points/10", "/attempts/1", "/review-candidates"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/1/knowledge"+path, nil))
		require.Equal(t, 200, w.Code, w.Body.String())
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children/2/knowledge/attempts/1", nil))
	require.Equal(t, 404, w.Code)
}

func TestKnowledgeCalendarRoute(t *testing.T) {
	r := NewRouter(Deps{Knowledge: knowledge.New(testdb.Open(t))})
	for _, tc := range []struct {
		path   string
		status int
	}{{"/1/knowledge/calendar?month=2026-09", 200}, {"/1/knowledge/calendar?month=2026-13", 400}, {"/2/knowledge/calendar?month=2026-09", 404}, {"/1/knowledge/wrongs?from=2026-02-30", 400}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/children"+tc.path, nil))
		require.Equal(t, tc.status, w.Code, w.Body.String())
	}
}
