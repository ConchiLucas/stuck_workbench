package englishcontent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const MediaPrefix = "/api/v1/english/task-media/"

var frozenURL = regexp.MustCompile(`^/api/v1/english/task-media/([a-f0-9]{64})\.(mp3|png|jpg|jpeg|webp)$`)

// FrozenMedia shares the existing task media store. Never overwrite a digest.
type FrozenMedia struct {
	SHA256 string `gorm:"primaryKey;size:64"`
	Kind   string
	Data   []byte
}

func (FrozenMedia) TableName() string { return "english_question_task_media" }
func MigrateMedia(db *gorm.DB) error  { return db.AutoMigrate(&FrozenMedia{}) }
func mediaURLs(e EnglishExample) []string {
	urls := []string{e.SpeechURL, e.Cue}
	for _, o := range e.Options {
		urls = append(urls, o.Picture)
	}
	return urls
}
func HasFrozenMedia(e EnglishExample) bool {
	for _, u := range mediaURLs(e) {
		if u != "" && !frozenURL.MatchString(u) {
			return false
		}
	}
	return true
}

// VerifyMedia checks actual stored bytes, including old digest-addressed snapshots.
func VerifyMedia(db *gorm.DB, e EnglishExample) error {
	for _, u := range mediaURLs(e) {
		if u == "" {
			continue
		}
		m := frozenURL.FindStringSubmatch(u)
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
	}
	return nil
}

// FreezeMedia must run inside the plan transaction. Failure rolls back media and plan.
func FreezeMedia(ctx context.Context, db *gorm.DB, base string, snap *HistorySnapshot) error {
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	sums := map[string]string{}
	cache := map[string]string{}
	freeze := func(dest *string, image bool) error {
		source := *dest
		if source == "" {
			return nil
		}
		if v, ok := cache[source]; ok {
			*dest = v
			return nil
		}
		if !(strings.HasPrefix(source, "/api/v1/english/words/") || strings.HasPrefix(source, "/api/v1/english/sentences/")) || strings.ContainsAny(source, "?%\\") || strings.Contains(source, "..") {
			return fmt.Errorf("invalid English media source")
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
		kind, ext := "audio", ".mp3"
		if image {
			switch {
			case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
				kind, ext = "png", ".png"
			case bytes.HasPrefix(data, []byte{255, 216, 255}):
				kind, ext = "jpeg", ".jpg"
			case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
				kind, ext = "webp", ".webp"
			default:
				return fmt.Errorf("invalid image bytes")
			}
		} else if !(bytes.HasPrefix(data, []byte("ID3")) || (len(data) >= 2 && data[0] == 255 && data[1]&224 == 224)) {
			return fmt.Errorf("invalid MP3 bytes")
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		row := FrozenMedia{SHA256: hash, Kind: kind, Data: data}
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
		sums[source] = hash
		*dest = MediaPrefix + hash + ext
		cache[source] = *dest
		return nil
	}
	if err := freeze(&snap.Example.SpeechURL, false); err != nil {
		return err
	}
	if err := freeze(&snap.Example.Cue, true); err != nil {
		return err
	}
	for i := range snap.Example.Options {
		if err := freeze(&snap.Example.Options[i].Picture, true); err != nil {
			return err
		}
	}
	snap.MediaSHA256 = sums
	return nil
}
