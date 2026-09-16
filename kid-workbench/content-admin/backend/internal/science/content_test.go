package science_test

import (
	"testing"
	"time"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/science"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestParseContentSupportsTargetPayload(t *testing.T) {
	got := science.ParseContent(`{"summary":" 过冬的方法 ","explanation":"节省能量","funFact":"心跳变慢"}`)
	require.Equal(t, "过冬的方法", got.Summary)
	require.Equal(t, "节省能量", got.Explanation)
	require.Equal(t, "心跳变慢", got.FunFact)
}

func TestParseContentAcceptsLegacyFactPayload(t *testing.T) {
	got := science.ParseContent(`{"kind":"fact","q":"谁会冬眠？","a":"熊"}`)
	require.Empty(t, got.Summary)
	require.Empty(t, got.Explanation)
	require.Empty(t, got.FunFact)
}

func TestSyncPreservesPublicationAndResetsChangedContent(t *testing.T) {
	gdb := setupScienceCatalog(t, `{"summary":"旧摘要","explanation":"旧解释","funFact":"旧趣闻"}`)
	svc := science.NewService(gdb, nil, nil, nil, nil, nil, nil)

	require.NoError(t, gdb.Create(&science.Asset{
		KpID: 1, Title: "冬眠", NeedsSenseImage: true,
		Summary: "旧摘要", Explanation: "旧解释", FunFact: "旧趣闻",
		ReviewStatus: "published", ContentVersion: 3,
		ReviewedAt: timePtr(time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)),
	}).Error)

	_, err := svc.Sync()
	require.NoError(t, err)
	unchanged := loadScienceAsset(t, gdb, 1)
	require.Equal(t, "published", unchanged.ReviewStatus)
	require.NotNil(t, unchanged.ReviewedAt)
	require.Equal(t, 3, unchanged.ContentVersion)

	require.NoError(t, gdb.Exec(`UPDATE knowledge_points SET payload = ? WHERE id = 1`,
		`{"summary":"新摘要","explanation":"新解释","funFact":"新趣闻"}`).Error)
	_, err = svc.Sync()
	require.NoError(t, err)
	changed := loadScienceAsset(t, gdb, 1)
	require.Equal(t, "draft", changed.ReviewStatus)
	require.Nil(t, changed.ReviewedAt)
	require.Equal(t, 4, changed.ContentVersion)
	require.Equal(t, "新摘要", changed.Summary)
}

func TestSyncCreatesNewScienceContentAsDraft(t *testing.T) {
	gdb := setupScienceCatalog(t, `{"summary":"过冬的方法","explanation":"节省能量","funFact":"心跳变慢"}`)
	svc := science.NewService(gdb, nil, nil, nil, nil, nil, nil)

	_, err := svc.Sync()
	require.NoError(t, err)
	got := loadScienceAsset(t, gdb, 1)
	require.Equal(t, "draft", got.ReviewStatus)
	require.Equal(t, 1, got.ContentVersion)
	require.Equal(t, "过冬的方法", got.Summary)
}

func setupScienceCatalog(t *testing.T, payload string) *gorm.DB {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'science', '科普', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'animal', '动物', 1);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no)
VALUES (1, 1, 'science-1', '冬眠', ?, 1, 1);
`, payload).Error)
	require.NoError(t, db.Migrate(gdb))
	return gdb
}

func loadScienceAsset(t *testing.T, gdb *gorm.DB, kpID int64) science.Asset {
	t.Helper()
	var got science.Asset
	require.NoError(t, gdb.First(&got, "kp_id = ?", kpID).Error)
	return got
}

func timePtr(value time.Time) *time.Time { return &value }
