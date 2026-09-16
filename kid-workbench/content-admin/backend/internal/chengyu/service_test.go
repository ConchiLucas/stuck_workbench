package chengyu_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/chengyu"
	"github.com/conchi/study-content-admin/internal/db"
	"gorm.io/gorm"
)

func TestParsePayloadReadsChengyuFields(t *testing.T) {
	got := chengyu.ParsePayload(`{"kind":"chengyu","pinyin":"yì xīn yì yì","meaning":"集中精神，做事专心","example":"做作业要一心一意。","wrong":["三心二意","慢慢来"]}`)
	require.Equal(t, "chengyu", got.Kind)
	require.Equal(t, "yì xīn yì yì", got.Pinyin)
	require.Equal(t, "集中精神，做事专心", got.Meaning)
	require.Equal(t, "做作业要一心一意。", got.Example)
	require.Equal(t, []string{"三心二意", "慢慢来"}, got.Wrong)
}

func TestListGroupsChengyuFromCatalog(t *testing.T) {
	gdb := setupChengyuCatalog(t)
	svc := chengyu.NewService(gdb)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	require.Len(t, list.Groups, 2)
	require.Equal(t, "日常成语", list.Groups[0].ModuleName)
	require.Equal(t, "一心一意", list.Groups[0].Items[0].Title)
	require.Equal(t, "yì xīn yì yì", list.Groups[0].Items[0].Pinyin)
	require.Equal(t, "集中精神，做事专心", list.Groups[0].Items[0].Meaning)
	require.Equal(t, "做作业要一心一意。", list.Groups[0].Items[0].Example)
	require.Equal(t, []string{"三心二意", "慢慢来"}, list.Groups[0].Items[0].Wrong)
	require.Equal(t, "动物成语", list.Groups[1].ModuleName)
	require.Equal(t, "狐假虎威", list.Groups[1].Items[0].Title)
}

func TestStoreSpeechRejectsNonMP3AndKeepsKindsSeparate(t *testing.T) {
	gdb := setupChengyuCatalog(t)
	svc := chengyu.NewService(gdb)
	_, err := svc.StoreSpeech(context.Background(), 1, "chengyu", []byte("not-an-mp3"))
	require.Error(t, err)
	stored, err := svc.StoreSpeech(context.Background(), 1, "chengyu", []byte("ID3chengyu-read"))
	require.NoError(t, err)
	require.True(t, stored.HasChengyuSpeech)
	require.False(t, stored.HasMeaningSpeech)
	mp3, err := svc.SpeechMP3(context.Background(), 1, "chengyu")
	require.NoError(t, err)
	require.Equal(t, []byte("ID3chengyu-read"), mp3)
}

func setupChengyuCatalog(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'chengyu', '成语', '', 8);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'daily', '日常成语', 1),
 (2, 1, 'animal', '动物成语', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (1, 1, 'cy001', '一心一意', '{"kind":"chengyu","pinyin":"yì xīn yì yì","meaning":"集中精神，做事专心","example":"做作业要一心一意。","wrong":["三心二意","慢慢来"]}', 1, 1),
 (2, 2, 'ca001', '狐假虎威', '{"kind":"chengyu","pinyin":"hú jiǎ hǔ wēi","meaning":"仗着别人的势力欺负人","example":"他借爸爸的名义吓唬人，真是狐假虎威。","wrong":["靠自己努力"]}', 2, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return gdb
}
