package chengyucontent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFrozenMediaSurvivesSourceReplacement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, MigrateMedia(db))
	audio := []byte("ID3original-chengyu-audio")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(audio)
	}))
	defer server.Close()
	original := append([]byte(nil), audio...)
	snap := HistorySnapshot{Example: ChengyuExample{Kind: "meaning", SpeechURL: "/api/v1/chengyu/items/1/speech.mp3"}}
	require.NoError(t, FreezeMedia(context.Background(), db, server.URL, &snap))
	require.True(t, HasFrozenMedia(snap.Example))
	audio = []byte("ID3replacement")
	res, err := http.Get(server.URL + "/api/v1/chengyu/items/1/speech.mp3")
	require.NoError(t, err)
	changed, err := io.ReadAll(res.Body)
	res.Body.Close()
	require.NoError(t, err)
	require.NotEqual(t, snap.MediaSHA256["/api/v1/chengyu/items/1/speech.mp3"], fmt.Sprintf("%x", sha256.Sum256(changed)))
	require.NoError(t, VerifyMedia(db, snap.Example))
	hash := strings.TrimSuffix(strings.TrimPrefix(snap.Example.SpeechURL, MediaPrefix), ".mp3")
	var row FrozenMedia
	require.NoError(t, db.First(&row, "sha256 = ?", hash).Error)
	require.Equal(t, original, row.Data)
	require.False(t, HasFrozenMedia(ChengyuExample{Kind: "meaning", SpeechURL: "/api/v1/chengyu/items/1/speech.mp3"}))
}

func TestFreezeMediaRejectsFailureAndCorruptExistingObject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, MigrateMedia(db))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "missing", 404) }))
	defer server.Close()
	snap := HistorySnapshot{Example: ChengyuExample{Kind: "meaning", SpeechURL: "/api/v1/chengyu/items/1/speech.mp3"}}
	require.Error(t, FreezeMedia(context.Background(), db, server.URL, &snap))
	data := []byte("ID3original")
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	require.NoError(t, db.Create(&FrozenMedia{SHA256: hash, Kind: "audio", Data: []byte("corrupt")}).Error)
	valid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer valid.Close()
	require.Error(t, FreezeMedia(context.Background(), db, valid.URL, &snap))
	frozen := ChengyuExample{Kind: "meaning", SpeechURL: MediaPrefix + hash + ".mp3"}
	require.ErrorContains(t, VerifyMedia(db, frozen), "校验失败")
	require.NoError(t, db.Delete(&FrozenMedia{}, "sha256 = ?", hash).Error)
	require.ErrorContains(t, VerifyMedia(db, frozen), "缺失")
}

func TestLiveURLIsNotImmutable(t *testing.T) {
	require.False(t, HasFrozenMedia(ChengyuExample{Kind: "meaning", SpeechURL: "/api/v1/chengyu/items/1/speech.mp3"}))
	require.True(t, HasFrozenMedia(ChengyuExample{Kind: "pick"}))
}

func TestFreezeMediaRejectsEnglishPath(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, MigrateMedia(db))
	snap := HistorySnapshot{Example: ChengyuExample{Kind: "meaning", SpeechURL: "/api/v1/english/words/1/speech.mp3"}}
	require.Error(t, FreezeMedia(context.Background(), db, "http://example.invalid", &snap))
}
