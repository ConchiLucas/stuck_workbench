package phrase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/phrase"
	"gorm.io/gorm"
)

func TestParsePayloadReadsPhraseFields(t *testing.T) {
	got := phrase.ParsePayload(`{"kind":"phrase","zh":"早上好。","wrong":["下午好。","晚上好。"],"scene":"早上见到老师","replyTo":""}`)
	require.Equal(t, "phrase", got.Kind)
	require.Equal(t, "早上好。", got.Zh)
	require.Equal(t, []string{"下午好。", "晚上好。"}, got.Wrong)
	require.Equal(t, "早上见到老师", got.Scene)
	require.Equal(t, "", got.ReplyTo)
}

func TestListGroupsPhrasesFromCatalog(t *testing.T) {
	gdb := setupPhraseCatalog(t)
	svc := phrase.NewService(gdb)

	list, err := svc.List("groups")
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	require.Len(t, list.Groups, 2)
	require.Equal(t, "问候", list.Groups[0].ModuleName)
	require.Equal(t, "Good morning.", list.Groups[0].Items[0].Title)
	require.Equal(t, "早上好。", list.Groups[0].Items[0].Zh)
	require.Equal(t, "早上见到老师", list.Groups[0].Items[0].Scene)
	require.Equal(t, []string{"下午好。", "晚上好。"}, list.Groups[0].Items[0].Wrong)
	require.Equal(t, "日常", list.Groups[1].ModuleName)
	require.Equal(t, "You're welcome.", list.Groups[1].Items[0].Title)
	require.Equal(t, "Thank you.", list.Groups[1].Items[0].ReplyTo)
	require.False(t, list.Groups[0].Items[0].HasSpeech)

	stored, err := svc.StoreSpeech(context.Background(), 1, []byte("ID3good-morning-phrase"))
	require.NoError(t, err)
	require.True(t, stored.HasSpeech)
	require.Equal(t, "/api/v1/phrase/items/1/speech.mp3", stored.SpeechAudioURL)
	mp3, err := svc.SpeechMP3(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, []byte("ID3good-morning-phrase"), mp3)
	_, err = svc.StoreSpeech(context.Background(), 1, []byte("not-an-mp3"))
	require.Error(t, err)
}

func setupPhraseCatalog(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'phrase', '英语短句', '', 9);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'greet', '问候', 1),
 (2, 1, 'daily', '日常', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (1, 1, 'ph001', 'Good morning.', '{"kind":"phrase","zh":"早上好。","wrong":["下午好。","晚上好。"],"scene":"早上见到老师","replyTo":""}', 1, 1),
 (2, 2, 'pd002', 'You''re welcome.', '{"kind":"phrase","zh":"不客气。","wrong":["谢谢你。"],"scene":"别人跟你说谢谢","replyTo":"Thank you."}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return gdb
}
