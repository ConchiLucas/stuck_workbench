package sciencetask

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conchi/study-learning/sciencecontent"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
)

func TestCreateFourKindsFromMaterials(t *testing.T) {
	g, err := db.OpenSQLite("file:science-task-create?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY AUTOINCREMENT, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY AUTOINCREMENT, module_id INTEGER, code TEXT, title TEXT, payload TEXT, difficulty INTEGER, order_no INTEGER);
		CREATE TABLE questions(id INTEGER PRIMARY KEY AUTOINCREMENT, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, difficulty INTEGER);
		CREATE TABLE science_assets(
			kp_id INTEGER PRIMARY KEY, title TEXT, module_code TEXT, module_name TEXT, module_order INTEGER, kp_order INTEGER,
			needs_sense_image INTEGER, needs_sense_image_override INTEGER, glyph_image_url TEXT, sense_image_url TEXT, speech_audio_url TEXT,
			glyph_object_key TEXT, sense_object_key TEXT, speech_object_key TEXT, summary TEXT, explanation TEXT, fun_fact TEXT,
			review_status TEXT, reviewed_at DATETIME, content_version INTEGER, synced_at DATETIME, updated_at DATETIME
		);
		INSERT INTO subjects VALUES (1,'science');
	`).Error)
	require.NoError(t, sciencecontent.EnsureMaterials(g))
	png := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 1, 2, 3})
	}))
	t.Cleanup(png.Close)
	require.NoError(t, Migrate(g))
	svc := New(g, png.URL)
	task, err := svc.Create(context.Background(), CreateInput{Title: "科普测试", Types: []string{"choice", "match", "sequence", "label"}, Count: 4})
	require.NoError(t, err)
	require.Len(t, task.Items, 4)
	seen := map[string]bool{}
	for _, item := range task.Items {
		require.NoError(t, sciencecontent.Validate(item.Example))
		seen[item.Kind] = true
	}
	require.True(t, seen["choice"] && seen["match"] && seen["sequence"] && seen["label"])
	reload, err := svc.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, task.Items[0].Example.Prompt, reload.Items[0].Example.Prompt)
}

func TestCreateRequiresTypes(t *testing.T) {
	g, err := db.OpenSQLite("file:science-task-invalid?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, Migrate(g))
	_, err = New(g, "").Create(context.Background(), CreateInput{Types: []string{"choice"}, Count: 0})
	require.Error(t, err)
}
