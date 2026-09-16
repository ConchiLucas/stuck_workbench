package pinyin_test

import (
	"context"
	"github.com/conchi/study-content-admin/internal/pinyin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestSyllableRecordingImportAndRead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&pinyin.SyllableAsset{}))
	require.NoError(t, db.Create(&pinyin.SyllableAsset{ID: 1, InitialText: "b", FinalText: "ā", Tone: 1, SyllableText: "bā", SpeechText: "八", Enabled: true}).Error)
	store := &syllableStore{files: map[string][]byte{}}
	svc := pinyin.NewService(db, store, nil, nil, nil)
	_, err = svc.SyllableSpeechMP3(context.Background(), 1)
	require.ErrorContains(t, err, "录音")
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "syllabs"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "syllabs", "cmn-ba1.mp3"), []byte("recorded-ba"), 0644))
	res, err := svc.ImportSyllableHumanPack(context.Background(), root)
	require.NoError(t, err)
	require.Equal(t, 1, res.Generated)
	items, err := svc.ListSyllables(context.Background())
	require.NoError(t, err)
	require.Contains(t, items[0].SpeechURL, "/syllables/1/speech.mp3")
	data, err := svc.SyllableSpeechMP3(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "recorded-ba", string(data))
	oldURL, _ := url.Parse(items[0].SpeechURL)
	require.NoError(t, os.WriteFile(filepath.Join(root, "syllabs", "cmn-ba1.mp3"), []byte("replacement"), 0644))
	_, err = svc.ImportSyllableHumanPack(context.Background(), root)
	require.NoError(t, err)
	data, err = svc.SyllableSpeechMP3(context.Background(), 1, oldURL.Query().Get("v"))
	require.NoError(t, err)
	require.Equal(t, "recorded-ba", string(data))
	_, err = svc.SyllableSpeechMP3(context.Background(), 1, "../../bad")
	require.Error(t, err)

	_, err = svc.UpdateSyllable(context.Background(), 1, pinyin.SyllableUpdate{SpeechText: "八", Enabled: false})
	require.NoError(t, err)
	items, err = svc.ListSyllables(context.Background())
	require.NoError(t, err)
	require.False(t, items[0].Enabled)
}

type syllableStore struct{ files map[string][]byte }

func (s *syllableStore) PutBytes(_ context.Context, k string, b []byte, _ string) (string, error) {
	s.files[k] = b
	return k, nil
}
func (s *syllableStore) GetBytes(_ context.Context, k string) ([]byte, error) { return s.files[k], nil }
