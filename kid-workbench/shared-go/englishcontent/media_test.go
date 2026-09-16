package englishcontent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFrozenMediaSurvivesSourceReplacement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, MigrateMedia(db))
	audio := []byte("ID3original-audio")
	picture := []byte("\x89PNG\r\n\x1a\noriginal-image")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".mp3") {
			w.Write(audio)
		} else {
			w.Write(picture)
		}
	}))
	defer server.Close()
	originalAudio := append([]byte(nil), audio...)
	originalPicture := append([]byte(nil), picture...)
	snap := HistorySnapshot{Example: EnglishExample{SpeechURL: "/api/v1/english/words/1/speech.mp3", Options: []Choice{{Picture: "/api/v1/english/words/2/sense.png"}}}}
	require.NoError(t, FreezeMedia(context.Background(), db, server.URL, &snap))
	require.True(t, HasFrozenMedia(snap.Example))
	audio = []byte("ID3replacement")
	picture = []byte("\x89PNG\r\n\x1a\nreplacement")
	for source, oldHash := range snap.MediaSHA256 {
		res, err := http.Get(server.URL + source)
		require.NoError(t, err)
		changed, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)
		require.NotEqual(t, oldHash, fmt.Sprintf("%x", sha256.Sum256(changed)), "source object must actually change")
	}
	require.NoError(t, VerifyMedia(db, snap.Example))
	for i, url := range []string{snap.Example.SpeechURL, snap.Example.Options[0].Picture} {
		hash := strings.Split(strings.TrimPrefix(url, MediaPrefix), ".")[0]
		var row FrozenMedia
		require.NoError(t, db.First(&row, "sha256 = ?", hash).Error)
		expected := originalAudio
		if i == 1 {
			expected = originalPicture
		}
		require.Equal(t, expected, row.Data)
		require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(expected)), hash)
	}
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(originalAudio)), snap.MediaSHA256["/api/v1/english/words/1/speech.mp3"])
	require.False(t, HasFrozenMedia(EnglishExample{SpeechURL: "/api/v1/english/words/1/speech.mp3"}))
}
func TestFreezeMediaRejectsFailureAndCorruptExistingObject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, MigrateMedia(db))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "missing", 404) }))
	defer server.Close()
	snap := HistorySnapshot{Example: EnglishExample{SpeechURL: "/api/v1/english/words/1/speech.mp3"}}
	require.Error(t, FreezeMedia(context.Background(), db, server.URL, &snap))
	data := []byte("ID3original")
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	require.NoError(t, db.Create(&FrozenMedia{SHA256: hash, Kind: "audio", Data: []byte("corrupt")}).Error)
	valid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer valid.Close()
	require.Error(t, FreezeMedia(context.Background(), db, valid.URL, &snap))
	frozen := EnglishExample{SpeechURL: MediaPrefix + hash + ".mp3"}
	require.ErrorContains(t, VerifyMedia(db, frozen), "校验失败")
	require.NoError(t, db.Delete(&FrozenMedia{}, "sha256 = ?", hash).Error)
	require.ErrorContains(t, VerifyMedia(db, frozen), "缺失")
}
