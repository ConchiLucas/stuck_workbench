package logic_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/logic"
	"gorm.io/gorm"
)

func TestParsePayloadReadsPatternFields(t *testing.T) {
	got := logic.ParsePayload(`{"kind":"pattern","seq":["🔴","🔵","🔴","🔵"],"a":"🔴","wrong":["🟢","🟡"],"prompt":"下一个是哪个？","speech":"下一个是哪个？"}`)
	require.Equal(t, "pattern", got.Kind)
	require.Equal(t, []string{"🔴", "🔵", "🔴", "🔵"}, got.Seq)
	require.Equal(t, "🔴", got.A)
	require.Equal(t, []string{"🟢", "🟡"}, got.Wrong)
	require.Equal(t, "下一个是哪个？", got.Prompt)
}

func TestParsePayloadDefaultsPromptFromKind(t *testing.T) {
	got := logic.ParsePayload(`{"kind":"classify","a":"🚗","wrong":["🍎","🍌"]}`)
	require.Equal(t, "哪个和其他不一样？", got.Prompt)
}

func TestListGroupsLogicItemsFromCatalog(t *testing.T) {
	gdb := setupLogicCatalog(t)
	svc := logic.NewService(gdb)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	require.Len(t, list.Groups, 2)
	require.Equal(t, "找规律", list.Groups[0].ModuleName)
	require.Equal(t, "红蓝交替", list.Groups[0].Items[0].Title)
	require.Equal(t, "pattern", list.Groups[0].Items[0].Kind)
	require.Equal(t, []string{"🔴", "🔵", "🔴", "🔵"}, list.Groups[0].Items[0].Seq)
	require.Equal(t, "🔴", list.Groups[0].Items[0].Answer)
	require.Equal(t, []string{"🟢", "🟡"}, list.Groups[0].Items[0].Wrong)
	require.Equal(t, "下一个是哪个？", list.Groups[0].Items[0].Prompt)
	require.Equal(t, "分类", list.Groups[1].ModuleName)
	require.Equal(t, "哪个不是水果", list.Groups[1].Items[0].Title)
	require.Equal(t, "🚗", list.Groups[1].Items[0].Answer)
}

func setupLogicCatalog(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'logic', '逻辑', '', 7);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'pattern', '找规律', 1),
 (2, 1, 'classify', '分类', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (1, 1, 'pt001', '红蓝交替', '{"kind":"pattern","seq":["🔴","🔵","🔴","🔵"],"a":"🔴","wrong":["🟢","🟡"],"prompt":"下一个是哪个？"}', 1, 1),
 (2, 2, 'cl001', '哪个不是水果', '{"kind":"classify","a":"🚗","wrong":["🍎","🍌"],"prompt":"哪个不是水果？"}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return gdb
}
