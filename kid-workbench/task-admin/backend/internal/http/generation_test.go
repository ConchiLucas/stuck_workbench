package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/taskgen"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestGenerationRoutesCreateAndValidate(t *testing.T) {
	g, e := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, e)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, taskgen.Migrate(g))
	r := NewRouter(Deps{Generation: taskgen.New(g, nil)})
	body := []byte(`{"title":"春天","spec":{"subjectCode":"literacy","scope":{"moduleCodes":["g1"]},"targetCount":8,"typeCounts":{"glyph_sense":4,"sense_char":4}}}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/question-tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "material_template", got["sourceMode"])
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/question-tasks/1", nil))
	require.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/question-tasks/1/generate", bytes.NewBufferString(`{"expectedRowVersion":1}`)))
	require.Equal(t, 400, w.Code)
}
