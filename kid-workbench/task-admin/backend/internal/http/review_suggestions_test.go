package httpapi

import (
	"github.com/conchi/study-task-admin/internal/reviewsuggestion"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewSuggestionsUnavailableIsJSON(t *testing.T) {
	r := NewRouter(Deps{})
	for _, path := range []string{"/api/v1/children/7/review-suggestions", "/api/v1/children/7/review-suggestions/1", "/api/v1/children/7/review-suggestions/1/evidence", "/api/v1/children/7/review-suggestions/1/tasks", "/api/v1/children/7/review-suggestions/1/plans"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 503, w.Code, path)
		require.Contains(t, w.Header().Get("Content-Type"), "application/json")
	}
}
func TestReviewSuggestionCommandUnavailable(t *testing.T) {
	r := NewRouter(Deps{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/children/7/review-suggestions/1/generate", strings.NewReader(`{"expectedRowVersion":1}`)))
	require.Equal(t, 503, w.Code)
}

func TestReviewSuggestionStrictBodies(t *testing.T) {
	r := NewRouter(Deps{ReviewSuggestions: &reviewsuggestion.Service{}})
	for _, body := range []string{`{"schemaVersion":1,"unknown":true}`, `{} {}`, `{"title":"` + strings.Repeat("x", 1<<20) + `"}`} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/children/7/review-suggestions", strings.NewReader(body)))
		require.Contains(t, []int{400, 413}, w.Code)
	}
}
