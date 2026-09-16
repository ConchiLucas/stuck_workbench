package logiccontent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MediaPrefix = "/api/v1/logic/task-media/"

var (
	frozenURL = regexp.MustCompile(`^/api/v1/logic/task-media/([a-f0-9]{64})\.svg$`)
	liveURL   = regexp.MustCompile(`^/api/v1/logic/items/(\d+)/glyph/([A-Za-z0-9_-]+)\.svg$`)
	sha256Hex = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type FrozenMedia struct {
	SHA256    string `gorm:"primaryKey;size:64"`
	Kind      string
	Ext       string
	Data      []byte
	CreatedAt time.Time
}

func (FrozenMedia) TableName() string { return TableName }

type ItemMedia struct {
	KpID     int64  `gorm:"primaryKey;column:kp_id"`
	ObjectID string `gorm:"primaryKey;column:object_id"`
	SHA256   string
	MIME     string
	Data     []byte
}

func (ItemMedia) TableName() string { return ItemMediaTable }

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
	if ext == ".svg" {
		return hash, ext, true
	}
	return "", "", false
}

func MediaBytes(db *gorm.DB, file string) ([]byte, string, error) {
	hash, _, ok := SHAFromFile(file)
	if !ok {
		return nil, "", gorm.ErrRecordNotFound
	}
	var row FrozenMedia
	if err := db.Where("sha256 = ?", hash).First(&row).Error; err != nil {
		return nil, "", err
	}
	return row.Data, "image/svg+xml", nil
}

func ItemGlyph(db *gorm.DB, kpID int64, objectID string) ([]byte, string, error) {
	return LiveGlyph(db, kpID, objectID)
}

func LiveGlyph(db *gorm.DB, kpID int64, objectID string) ([]byte, string, error) {
	var row ItemMedia
	err := db.Where("kp_id = ? AND object_id = ?", kpID, objectID).First(&row).Error
	if err != nil {
		return nil, "", err
	}
	mime := row.MIME
	if mime == "" {
		mime = "image/svg+xml"
	}
	return row.Data, mime, nil
}

func HasFrozenMedia(e LogicExample) bool {
	urls := CollectImageURLs(e)
	if len(urls) == 0 {
		return true
	}
	for _, u := range urls {
		if !frozenURL.MatchString(u) {
			return false
		}
	}
	return true
}

func CollectImageURLs(e LogicExample) []string {
	seen := map[string]bool{}
	var out []string
	for _, u := range e.ImageURLs {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

func RewriteImageURLs(e *LogicExample, fn func(string) string) {
	if e.ImageURLs == nil {
		return
	}
	next := map[string]string{}
	for k, u := range e.ImageURLs {
		next[k] = fn(u)
	}
	e.ImageURLs = next
}

func VerifyMedia(db *gorm.DB, e LogicExample) error {
	for _, u := range CollectImageURLs(e) {
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

func FreezeMedia(ctx context.Context, db *gorm.DB, _ string, snap *HistorySnapshot) error {
	if snap == nil {
		return fmt.Errorf("空快照")
	}
	example := SnapshotToExample(*snap)
	urls := CollectImageURLs(example)
	if len(urls) == 0 {
		return nil
	}
	hashes := map[string]string{}
	rewritten := map[string]string{}
	for _, source := range urls {
		if frozenURL.MatchString(source) {
			continue
		}
		m := liveURL.FindStringSubmatch(source)
		if m == nil || strings.ContainsAny(source, "?%\\") || strings.Contains(source, "..") {
			return fmt.Errorf("invalid logic media source")
		}
		kpID, _ := strconv.ParseInt(m[1], 10, 64)
		objectID := m[2]
		data, _, err := LiveGlyph(db, kpID, objectID)
		if err != nil {
			return fmt.Errorf("素材读取失败: %w", err)
		}
		if len(data) == 0 || !isSVG(data) {
			return fmt.Errorf("invalid media bytes")
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		row := FrozenMedia{SHA256: hash, Kind: "image", Ext: ".svg", Data: append([]byte(nil), data...)}
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
		rewritten[source] = MediaPrefix + hash + ".svg"
	}
	if snap.MediaSHA256 == nil {
		snap.MediaSHA256 = map[string]string{}
	}
	for k, v := range hashes {
		snap.MediaSHA256[k] = v
	}
	RewriteImageURLs(&example, func(u string) string {
		if next, ok := rewritten[u]; ok {
			return next
		}
		return u
	})
	snap.ImageURLs = example.ImageURLs
	snap.MediaImmutable = HasFrozenMedia(example)
	return nil
}

func isSVG(data []byte) bool {
	trim := bytes.TrimSpace(data)
	return bytes.HasPrefix(trim, []byte("<svg")) || bytes.HasPrefix(trim, []byte("<?xml"))
}
