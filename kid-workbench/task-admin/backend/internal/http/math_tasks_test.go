package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conchi/study-learning/mathcontent"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/mathtask"
	"github.com/stretchr/testify/require"
)

func TestMathTaskHTTPCreateReadPublish(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		_ = json.NewEncoder(w).Encode(mathcontent.Defaults())
	}))
	defer source.Close()
	gdb, err := db.OpenSQLite("file:math-http?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, mathtask.Migrate(gdb))
	router := NewRouter(Deps{Math: mathtask.New(gdb, source.URL)})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/math/question-tasks", bytes.NewBufferString(`{"title":"测试","detailIds":["addition-equation"],"rangeMax":5,"count":3}`)))
	require.Equal(t, 200, w.Code, w.Body.String())
	var task mathtask.Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &task))
	require.Len(t, task.Items, 3)
	require.NotEmpty(t, task.Items[0].Detail.Example.AnswerOptionID)
	for _, path := range []string{"/api/v1/math/question-tasks", "/api/v1/math/question-tasks/materials", "/api/v1/math/question-tasks/1"} {
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 200, w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/math/question-tasks", nil))
	require.NotContains(t, w.Body.String(), `"sequence"`)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/math/question-tasks/1/publish", nil))
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"status":"published"`)
	require.False(t, gdb.Migrator().HasTable("attempts"))
	require.False(t, gdb.Migrator().HasTable("mastery_skills"))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/math/question-tasks/1/attempts", bytes.NewBufferString(`{"answer":"3"}`)))
	require.NotEqual(t, 200, w.Code)
}
func TestMathTaskHTTPMissingService(t *testing.T) {
	w := httptest.NewRecorder()
	NewRouter(Deps{}).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/math/question-tasks", nil))
	require.Equal(t, 503, w.Code)
}
