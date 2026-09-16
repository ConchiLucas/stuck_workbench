package poem_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/poem"
	"gorm.io/gorm"
)

func TestParsePayloadReadsAuthorAndLines(t *testing.T) {
	got := poem.ParsePayload(`{"kind":"poem","author":"李白","line1":"床前明月光","line2":"疑是地上霜","lines":["床前明月光","疑是地上霜","举头望明月","低头思故乡"]}`)
	require.Equal(t, "李白", got.Author)
	require.Equal(t, "床前明月光", got.Line1)
	require.Equal(t, "疑是地上霜", got.Line2)
}

func TestParsePayloadFallsBackToLine1Line2(t *testing.T) {
	got := poem.ParsePayload(`{"kind":"poem","author":"孟浩然","line1":"春眠不觉晓","line2":"处处闻啼鸟"}`)
	require.Equal(t, "春眠不觉晓", got.Line1)
	require.Equal(t, "处处闻啼鸟", got.Line2)
}

func TestSyncAndListGroupsPoems(t *testing.T) {
	gdb := setupPoemCatalog(t)
	svc := poem.NewService(gdb)

	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 2, res.Total)
	require.Equal(t, 2, res.Upserted)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	require.Len(t, list.Groups, 1)
	require.Equal(t, "必背古诗", list.Groups[0].ModuleName)
	require.Equal(t, "静夜思", list.Groups[0].Items[0].Title)
	require.Equal(t, "李白", list.Groups[0].Items[0].Author)
	require.Equal(t, []string{"床前明月光", "疑是地上霜", "举头望明月", "低头思故乡"}, list.Groups[0].Items[0].Lines)
	require.Equal(t, "春晓", list.Groups[0].Items[1].Title)
	require.Equal(t, 1, list.Groups[0].Items[0].Difficulty)
}

func TestSyncRefreshesChangedPayload(t *testing.T) {
	gdb := setupPoemCatalog(t)
	svc := poem.NewService(gdb)
	_, err := svc.Sync()
	require.NoError(t, err)

	require.NoError(t, gdb.Exec(`UPDATE knowledge_points SET title = ?, payload = ? WHERE id = 1`,
		"静夜思·新",
		`{"kind":"poem","author":"李白","line1":"床前看月光","line2":"疑是地上霜","lines":["床前看月光","疑是地上霜"]}`,
	).Error)

	_, err = svc.Sync()
	require.NoError(t, err)
	list, err := svc.List("table")
	require.NoError(t, err)
	require.Equal(t, "静夜思·新", list.Items[0].Title)
	require.Equal(t, "床前看月光", list.Items[0].Line1)
	require.Equal(t, []string{"床前看月光", "疑是地上霜"}, list.Items[0].Lines)
}

func setupPoemCatalog(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
CREATE TABLE questions (id INTEGER PRIMARY KEY AUTOINCREMENT, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, speech TEXT, difficulty INTEGER);
CREATE UNIQUE INDEX uq_questions_kp_code ON questions(kp_id, code);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'poem', '古诗', '', 6);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'poem50', '必背古诗', 1);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (1, 1, 'pm001', '静夜思', '{"kind":"poem","author":"李白","line1":"床前明月光","line2":"疑是地上霜","lines":["床前明月光","疑是地上霜","举头望明月","低头思故乡"]}', 1, 1),
 (2, 1, 'pm002', '春晓', '{"kind":"poem","author":"孟浩然","line1":"春眠不觉晓","line2":"处处闻啼鸟","lines":["春眠不觉晓","处处闻啼鸟","夜来风雨声","花落知多少"]}', 1, 2);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return gdb
}
