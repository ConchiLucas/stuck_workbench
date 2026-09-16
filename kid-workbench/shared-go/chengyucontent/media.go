package chengyucontent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MediaPrefix = "/api/v1/chengyu/task-media/"

var frozenURL = regexp.MustCompile(`^/api/v1/chengyu/task-media/([a-f0-9]{64})\.mp3$`)

type FrozenMedia struct {
	SHA256 string `gorm:"primaryKey;size:64"`
	Kind   string
	Data   []byte
}

func (FrozenMedia) TableName() string { return "chengyu_question_task_media" }
func MigrateMedia(db *gorm.DB) error  { return db.AutoMigrate(&FrozenMedia{}) }

var sha256Hex = regexp.MustCompile(`^[a-f0-9]{64}$`)

func SHAFromFile(file string) (string, bool) {
	file = strings.TrimSpace(file)
	if !strings.HasSuffix(file, ".mp3") {
		return "", false
	}
	hash := strings.TrimSuffix(file, ".mp3")
	if !sha256Hex.MatchString(hash) {
		return "", false
	}
	return hash, true
}

func MediaBytes(db *gorm.DB, file string) ([]byte, error) {
	hash, ok := SHAFromFile(file)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	var row FrozenMedia
	if err := db.Where("sha256 = ?", hash).First(&row).Error; err != nil {
		return nil, err
	}
	return row.Data, nil
}

func HasFrozenMedia(e ChengyuExample) bool {
	if e.SpeechURL == "" {
		return !NeedsSpeech(e.Kind)
	}
	return frozenURL.MatchString(e.SpeechURL)
}

func VerifyMedia(db *gorm.DB, e ChengyuExample) error {
	if e.SpeechURL == "" {
		if NeedsSpeech(e.Kind) {
			return fmt.Errorf("历史媒体未冻结")
		}
		return nil
	}
	m := frozenURL.FindStringSubmatch(e.SpeechURL)
	if m == nil {
		return fmt.Errorf("历史媒体未冻结")
	}
	var row FrozenMedia
	if err := db.Where("sha256 = ?", m[1]).First(&row).Error; err != nil {
		return fmt.Errorf("历史媒体缺失")
	}
	if fmt.Sprintf("%x", sha256.Sum256(row.Data)) != m[1] {
		return fmt.Errorf("历史媒体校验失败")
	}
	return nil
}

func FreezeMedia(ctx context.Context, db *gorm.DB, base string, snap *HistorySnapshot) error {
	if snap.Example.SpeechURL == "" {
		if NeedsSpeech(snap.Example.Kind) {
			return fmt.Errorf("缺少成语读音")
		}
		return nil
	}
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	source := snap.Example.SpeechURL
	if !(strings.HasPrefix(source, "/api/v1/chengyu/items/") && strings.HasSuffix(source, "/speech.mp3")) || strings.ContainsAny(source, "?%\\") || strings.Contains(source, "..") {
		return fmt.Errorf("invalid chengyu media source")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+source, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("media unavailable: HTTP %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if err != nil {
		return err
	}
	if len(data) == 0 || len(data) > 8<<20 {
		return fmt.Errorf("invalid media size")
	}
	if !(bytes.HasPrefix(data, []byte("ID3")) || (len(data) >= 2 && data[0] == 255 && data[1]&224 == 224)) {
		return fmt.Errorf("invalid MP3 bytes")
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	row := FrozenMedia{SHA256: hash, Kind: "audio", Data: data}
	if err = db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return err
	}
	var stored FrozenMedia
	if err = db.WithContext(ctx).Where("sha256 = ?", hash).First(&stored).Error; err != nil {
		return err
	}
	if !bytes.Equal(stored.Data, data) {
		return fmt.Errorf("stored media digest mismatch")
	}
	snap.MediaSHA256 = map[string]string{source: hash}
	snap.Example.SpeechURL = MediaPrefix + hash + ".mp3"
	return nil
}
