package poemcontent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MediaPrefix = "/api/v1/poem/task-media/"

var (
	frozenURL = regexp.MustCompile(`^/api/v1/poem/task-media/([a-f0-9]{64})\.(wav|mp3)$`)
	liveURL   = regexp.MustCompile(`^/api/v1/poem/items/(\d+)/speech/(\d+)\.(wav|mp3)$`)
	sha256Hex = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type FrozenMedia struct {
	SHA256 string `gorm:"primaryKey;size:64"`
	Kind   string
	Ext    string
	Data   []byte
}

func (FrozenMedia) TableName() string { return "poem_question_task_media" }

type ItemMedia struct {
	KpID   int64  `gorm:"primaryKey;column:kp_id"`
	Kind   string `gorm:"primaryKey"`
	Ord    int    `gorm:"primaryKey"`
	SHA256 string
	MIME   string
	Data   []byte
}

func (ItemMedia) TableName() string { return "poem_item_media" }

func MigrateMedia(db *gorm.DB) error {
	if err := db.AutoMigrate(&FrozenMedia{}, &ItemMedia{}); err != nil {
		return err
	}
	if db.Dialector.Name() != "postgres" || !db.Migrator().HasTable("plan_items") {
		return nil
	}
	return db.Exec(`ALTER TABLE plan_items ALTER COLUMN option_order TYPE TEXT, ALTER COLUMN picks TYPE TEXT`).Error
}

func SHAFromFile(file string) (string, string, bool) {
	file = strings.TrimSpace(file)
	ext := strings.ToLower(path.Ext(file))
	hash := strings.TrimSuffix(file, ext)
	if !sha256Hex.MatchString(hash) {
		return "", "", false
	}
	switch ext {
	case ".wav", ".mp3":
		return hash, ext, true
	}
	return "", "", false
}

func MediaBytes(db *gorm.DB, file string) ([]byte, string, error) {
	hash, ext, ok := SHAFromFile(file)
	if !ok {
		return nil, "", gorm.ErrRecordNotFound
	}
	var row FrozenMedia
	if err := db.Where("sha256 = ?", hash).First(&row).Error; err != nil {
		return nil, "", err
	}
	ctype := "application/octet-stream"
	switch ext {
	case ".wav":
		ctype = "audio/wav"
	case ".mp3":
		ctype = "audio/mpeg"
	}
	return row.Data, ctype, nil
}

func ItemSpeech(db *gorm.DB, kpID int64, ord int) ([]byte, string, error) {
	var row ItemMedia
	if err := db.Where("kp_id = ? AND kind = ? AND ord = ?", kpID, "speech", ord).First(&row).Error; err != nil {
		return nil, "", err
	}
	mime := row.MIME
	if mime == "" {
		mime = "audio/wav"
	}
	return row.Data, mime, nil
}

func HasFrozenMedia(e PoemExample) bool {
	urls := CollectMediaURLs(e)
	if len(urls) == 0 {
		return !NeedsSpeech(e.Kind)
	}
	for _, u := range urls {
		if !frozenURL.MatchString(u) {
			return false
		}
	}
	return true
}

func VerifyMedia(db *gorm.DB, e PoemExample) error {
	urls := CollectMediaURLs(e)
	if len(urls) == 0 {
		if NeedsSpeech(e.Kind) {
			return fmt.Errorf("历史媒体未冻结")
		}
		return nil
	}
	for _, u := range urls {
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

func FreezeMedia(ctx context.Context, db *gorm.DB, base string, snap *HistorySnapshot) error {
	urls := CollectMediaURLs(snap.Example)
	if len(urls) == 0 {
		if NeedsSpeech(snap.Example.Kind) {
			return fmt.Errorf("缺少古诗读音")
		}
		return nil
	}
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	hashes := map[string]string{}
	rewritten := map[string]string{}
	for _, source := range urls {
		if frozenURL.MatchString(source) {
			continue
		}
		if !liveURL.MatchString(source) || strings.ContainsAny(source, "?%\\") || strings.Contains(source, "..") {
			return fmt.Errorf("invalid poem media source")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+source, nil)
		if err != nil {
			return err
		}
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		res.Body.Close()
		if err != nil {
			return err
		}
		if res.StatusCode != 200 {
			return fmt.Errorf("media unavailable: HTTP %d", res.StatusCode)
		}
		if len(data) == 0 || len(data) > 8<<20 {
			return fmt.Errorf("invalid media size")
		}
		ext, kind, err := detectAudio(data)
		if err != nil {
			return err
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		row := FrozenMedia{SHA256: hash, Kind: kind, Ext: ext, Data: data}
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
		hashes[source] = hash
		rewritten[source] = MediaPrefix + hash + ext
	}
	if snap.MediaSHA256 == nil {
		snap.MediaSHA256 = map[string]string{}
	}
	for k, v := range hashes {
		snap.MediaSHA256[k] = v
	}
	RewriteMediaURLs(&snap.Example, func(u string) string {
		if next, ok := rewritten[u]; ok {
			return next
		}
		return u
	})
	return nil
}

func detectAudio(data []byte) (ext, kind string, err error) {
	if bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 && bytes.Equal(data[8:12], []byte("WAVE")) {
		return ".wav", "audio", nil
	}
	if bytes.HasPrefix(data, []byte("ID3")) || (len(data) >= 2 && data[0] == 255 && data[1]&224 == 224) {
		return ".mp3", "audio", nil
	}
	return "", "", fmt.Errorf("invalid media bytes")
}

func EnsureLineSpeech(db *gorm.DB, kpID int64, ord int, seed string) error {
	var n int64
	if err := db.Model(&ItemMedia{}).Where("kp_id = ? AND kind = ? AND ord = ?", kpID, "speech", ord).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	data := SpeechWAV(fmt.Sprintf("%d:%d:%s", kpID, ord, seed))
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&ItemMedia{
		KpID: kpID, Kind: "speech", Ord: ord, SHA256: hash, MIME: "audio/wav", Data: data,
	}).Error
}

func ParseLiveSpeech(fileOrd string) (int, bool) {
	ord, err := strconv.Atoi(strings.TrimSpace(fileOrd))
	if err != nil || ord < 1 || ord > 32 {
		return 0, false
	}
	return ord, true
}
