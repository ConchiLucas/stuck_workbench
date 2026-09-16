package catalog_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/catalog"
)

func TestRepositoryReturnsOnlyChildSafeMathCatalog(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, difficulty INTEGER, order_no INTEGER);
INSERT INTO subjects VALUES(1,'math'),(2,'literacy');
INSERT INTO modules VALUES(1,1,'add10','20以内加法',1),(2,1,'shape','认识图形',2),(3,2,'add10','识字同名组',1);
INSERT INTO knowledge_points VALUES
 (10,1,'2+3','{"kind":"add","a":2,"b":3,"secret":"raw"}',1,1),
 (11,2,'圆形','{}',1,1),
 (12,3,'人','{"kind":"add","a":2,"b":3}',1,1);
`).Error)
	repo := catalog.NewRepository(gdb)

	modules, err := repo.ListModules(context.Background())
	require.NoError(t, err)
	require.Len(t, modules, 2)
	require.Equal(t, "add10", modules[0].Code)
	require.Equal(t, "within5", modules[0].Stages[0].Code)

	items, err := repo.ListStageItems(context.Background(), "add10", "within5")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(10), items[0].KpID)
	require.Equal(t, 2, items[0].A)
	require.Equal(t, 3, items[0].B)

	raw, err := json.Marshal(items[0])
	require.NoError(t, err)
	require.NotContains(t, string(raw), "secret")
	require.NotContains(t, string(raw), "payload")
}
