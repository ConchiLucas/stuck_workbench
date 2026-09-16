package logictask

import (
	"context"
	"testing"

	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
)

func TestCreateSixKindsFromMaterials(t *testing.T) {
	g, err := db.OpenSQLite("file:logic-task-create?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY AUTOINCREMENT, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY AUTOINCREMENT, module_id INTEGER, code TEXT, title TEXT, payload TEXT, difficulty INTEGER, order_no INTEGER);
		CREATE TABLE questions(id INTEGER PRIMARY KEY AUTOINCREMENT, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, difficulty INTEGER);
		INSERT INTO subjects VALUES (1,'logic');
		INSERT INTO modules(subject_id, code, name, order_no) VALUES (1,'playground','逻辑练习',1);
	`).Error)
	require.NoError(t, logiccontent.EnsureMaterials(g))
	require.NoError(t, Migrate(g))
	svc := New(g, "")
	task, err := svc.Create(context.Background(), CreateInput{Title: "逻辑测试", Types: []string{"pattern", "classify", "order", "shape_reason", "diff", "compare"}, Count: 6})
	require.NoError(t, err)
	require.Len(t, task.Items, 6)
	seen := map[string]bool{}
	for _, item := range task.Items {
		require.NoError(t, logiccontent.Validate(item.Example))
		seen[item.Kind] = true
		require.True(t, logiccontent.HasFrozenMedia(item.Example))
	}
	require.True(t, seen["pattern"] && seen["classify"] && seen["order"] && seen["shape_reason"] && seen["diff"] && seen["compare"])
}

func TestCreateRequiresTypes(t *testing.T) {
	g, err := db.OpenSQLite("file:logic-task-invalid?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, Migrate(g))
	_, err = New(g, "").Create(context.Background(), CreateInput{Types: []string{"pattern"}, Count: 0})
	require.Error(t, err)
}
