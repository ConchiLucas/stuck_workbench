package literacy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/conchi/study-content-admin/internal/storage"
	"github.com/conchi/study-learning/literacycontract"
	"github.com/hajimehoshi/go-mp3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrStoreUnavailable = errors.New("素材存储暂不可用")

const imageLimit int64 = 10 << 20
const audioLimit int64 = 20 << 20

var ErrSourceChanged = errors.New("素材已更新，请刷新后重试")
var ErrMaterialNotReady = errors.New("素材未就绪")
var ErrInvalidFreeze = errors.New("冻结请求必须包含 1–80 个不重复的知识点及来源修订")

type Capability struct {
	Ready   bool     `json:"ready"`
	Reasons []string `json:"reasons"`
}
type GenerationMaterial struct {
	KpID           int64                 `json:"kpId"`
	Text           string                `json:"text"`
	ModuleCode     string                `json:"moduleCode"`
	ModuleName     string                `json:"moduleName"`
	SourceRevision string                `json:"sourceRevision"`
	Capabilities   map[string]Capability `json:"capabilities"`
}
type GenerationMaterialsResult struct {
	SubjectCode string               `json:"subjectCode"`
	ModuleCode  string               `json:"moduleCode"`
	Items       []GenerationMaterial `json:"items"`
}
type FreezeItem struct {
	QuestionTypes  []string `json:"questionTypes,omitempty"`
	KpID           int64    `json:"kpId"`
	SourceRevision string   `json:"sourceRevision"`
}
type MediaRef struct {
	RevisionID string `json:"revisionId"`
	Kind       string `json:"kind"`
	SHA256     string `json:"sha256"`
}
type FrozenMaterial struct {
	WritingTemplate *MediaRef `json:"writingTemplate,omitempty"`
	KpID            int64     `json:"kpId"`
	Text            string    `json:"text"`
	ModuleCode      string    `json:"moduleCode"`
	ModuleName      string    `json:"moduleName"`
	RevisionID      string    `json:"revisionId"`
	Glyph           *MediaRef `json:"glyph,omitempty"`
	Sense           *MediaRef `json:"sense,omitempty"`
	Speech          *MediaRef `json:"speech"`
}
type FreezeResult struct {
	Items []FrozenMaterial `json:"items"`
}
type revisionMedia struct {
	Key         string `json:"key"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
}
type materialRevision struct {
	RevisionID     string `gorm:"primaryKey"`
	SubjectCode    string
	SourceRevision string
	KpID           int64
	Content        string
	Media          string
	CreatedAt      time.Time
}

func (materialRevision) TableName() string { return "material_revisions" }
func digest(b []byte) string               { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func sourceRevision(a Asset) string {
	// Maintenance timestamps and display ordering are not content revisions.
	b, _ := json.Marshal(struct {
		Text, ModuleCode, ModuleName, Glyph, Sense, Speech, Epoch string
		Pending                                                   bool
	}{strings.TrimSpace(a.CharText), a.ModuleCode, a.ModuleName, a.GlyphImageURL, a.SenseImageURL, a.SpeechAudioURL, a.MaterialEpoch, a.MaterialPending})
	if a.WritingTemplateVersion != "" {
		b = append(b, []byte(a.WritingTemplateVersion)...)
	}
	return digest(b)
}

// PostgreSQL row locks coordinate all processes. SQLite is serialized locally for tests;
// its write transaction additionally prevents another process committing a metadata write.
var materialLock = make(chan struct{}, 1)

func (s *Service) withMaterialLocks(ctx context.Context, ids []int64, fn func(*Service) error) error {
	ids = append([]int64(nil), ids...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if s.db.Dialector.Name() == "sqlite" {
		select {
		case materialLock <- struct{}{}:
			defer func() { <-materialLock }()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var a Asset
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, "kp_id = ?", id).Error; err != nil {
				return err
			}
		}
		copy := *s
		copy.db = tx
		return fn(&copy)
	})
}
func metadataReasons(a Asset) []string {
	reasons := []string{}
	if strings.TrimSpace(a.CharText) == "" {
		reasons = append(reasons, "missing_text")
	}
	if a.SenseImageURL == "" {
		reasons = append(reasons, "missing_sense_image")
	}
	if a.SpeechAudioURL == "" {
		reasons = append(reasons, "missing_speech_audio")
	}
	if a.MaterialPending {
		reasons = append(reasons, "media_update_incomplete")
	}
	return reasons
}
func mediaLimit(kind string) int64 {
	if kind == "writing_template" {
		return WritingTemplateLimit
	}
	if kind == "speech" {
		return audioLimit
	}
	return imageLimit
}

// Decode to a bounded discard sink: a header alone is not playable audio.
func validMP3(ctx context.Context, b []byte) (valid bool) {
	defer func() {
		if recover() != nil {
			valid = false
		}
	}()
	decoder, err := mp3.NewDecoder(&contextReader{ctx: ctx, r: bytes.NewReader(b)})
	if err != nil {
		return false
	}
	n, err := io.Copy(io.Discard, io.LimitReader(&contextReader{ctx: ctx, r: decoder}, (256<<20)+1))
	return err == nil && n > 0 && n <= 256<<20
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(b)
}

// Validate actual bytes rather than the legacy key suffix: some .png keys
// contain JPEG. Preserve the original bytes and their real MIME type in revisions.
func validateMaterialImage(ctx context.Context, b []byte) error {
	cfg, format, err := image.DecodeConfig(&contextReader{ctx: ctx, r: bytes.NewReader(b)})
	if err != nil {
		return err
	}
	if (format != "png" && format != "jpeg") || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 20_000_000 {
		return ErrMaterialNotReady
	}
	_, _, err = image.Decode(&contextReader{ctx: ctx, r: bytes.NewReader(b)})
	return err
}

func (s *Service) materialBytes(ctx context.Context, a Asset) (map[string][]byte, []string, error) {
	result := map[string][]byte{}
	reasons := metadataReasons(a)
	if len(reasons) > 0 {
		return result, reasons, nil
	}
	if s.store == nil {
		return nil, nil, ErrStoreUnavailable
	}
	for _, kind := range []string{"glyph", "sense", "speech"} {
		url, key := a.GlyphImageURL, s.store.GlyphKey(a.KpID)
		if kind == "sense" {
			url, key = a.SenseImageURL, s.store.SenseKey(a.KpID)
		}
		if kind == "speech" {
			url, key = a.SpeechAudioURL, s.store.SpeechKey(a.KpID)
		}
		if url == "" {
			continue
		}
		b, err := s.store.GetBytesLimited(ctx, key, mediaLimit(kind))
		if err != nil && !errors.Is(err, storage.ErrNotFound) && !errors.Is(err, storage.ErrObjectTooLarge) {
			return nil, nil, fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
		}
		valid := err == nil && len(b) > 0
		if valid && kind != "speech" {
			err = validateMaterialImage(ctx, b)
			valid = err == nil
		}
		if valid && kind == "speech" {
			valid = validMP3(ctx, b)
		}
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if valid {
			result[kind] = b
		} else if kind != "glyph" {
			reasons = append(reasons, "invalid_"+kind+"_media")
		}
	}
	return result, reasons, nil
}
func (s *Service) GenerationMaterials(ctx context.Context, module string) (GenerationMaterialsResult, error) {
	out := GenerationMaterialsResult{SubjectCode: "literacy", ModuleCode: strings.TrimSpace(module), Items: []GenerationMaterial{}}
	q := s.db.WithContext(ctx)
	if out.ModuleCode != "" {
		q = q.Where("module_code = ?", out.ModuleCode)
	}
	var assets []Asset
	if err := q.Order("module_order, kp_order, kp_id").Find(&assets).Error; err != nil {
		return out, err
	}
	if out.ModuleCode != "" && len(assets) == 0 {
		return out, gorm.ErrRecordNotFound
	}
	for _, a := range assets {
		reasons := metadataReasons(a)
		cap := Capability{Ready: len(reasons) == 0, Reasons: reasons}
		wr := writingReasons(a)
		if a.WritingTemplateVersion != "" {
			if _, err := s.WritingTemplate(ctx, a.KpID, a.WritingTemplateVersion); err != nil {
				wr = append(wr, "invalid_writing_template")
			}
		}
		out.Items = append(out.Items, GenerationMaterial{a.KpID, a.CharText, a.ModuleCode, a.ModuleName, sourceRevision(a), map[string]Capability{"glyph_sense": cap, "sense_char": cap, "write_char": {Ready: len(wr) == 0, Reasons: wr}}})
	}
	return out, nil
}
func (s *Service) FreezeMaterials(ctx context.Context, items []FreezeItem) (FreezeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out := FreezeResult{Items: []FrozenMaterial{}}
	if len(items) == 0 || len(items) > 80 {
		return out, ErrInvalidFreeze
	}
	ids := []int64{}
	seen := map[int64]bool{}
	for _, item := range items {
		for _, typ := range item.QuestionTypes {
			if !literacycontract.Supports(typ) {
				return out, ErrInvalidFreeze
			}
		}
		if item.KpID <= 0 || item.SourceRevision == "" || seen[item.KpID] {
			return out, ErrInvalidFreeze
		}
		seen[item.KpID] = true
		ids = append(ids, item.KpID)
	}
	err := s.withMaterialLocks(ctx, ids, func(locked *Service) error {
		for _, item := range items {
			var a Asset
			if err := locked.db.First(&a, "kp_id = ?", item.KpID).Error; err != nil {
				return err
			}
			if sourceRevision(a) != item.SourceRevision {
				return ErrSourceChanged
			}
			media, reasons, err := locked.materialBytesForTypes(ctx, a, item.QuestionTypes)
			if err != nil {
				return err
			}
			if len(reasons) > 0 {
				return fmt.Errorf("%w: %d: %s", ErrMaterialNotReady, a.KpID, strings.Join(reasons, "；"))
			}
			manifest := map[string]revisionMedia{}
			for _, kind := range []string{"glyph", "sense", "speech", "writing_template"} {
				b, ok := media[kind]
				if !ok {
					continue
				}
				hash := digest(b)
				ext, ct := "png", "image/png"
				if kind == "writing_template" {
					ext, ct = "json", "application/json"
				} else if kind == "speech" {
					ext, ct = "mp3", "audio/mpeg"
				} else if bytes.HasPrefix(b, []byte{0xff, 0xd8}) {
					ext, ct = "jpg", "image/jpeg"
				}
				key := "material-revisions/sha256/" + hash + "." + ext
				existing, readErr := s.store.GetBytesLimited(ctx, key, mediaLimit(kind))
				if readErr == nil {
					if digest(existing) != hash {
						return fmt.Errorf("immutable object hash mismatch")
					}
				} else {
					if !errors.Is(readErr, storage.ErrNotFound) {
						return fmt.Errorf("%w: %v", ErrStoreUnavailable, readErr)
					}
					if _, err := s.store.PutBytes(ctx, key, b, ct); err != nil {
						return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
					}
				}
				verify, err := s.store.GetBytesLimited(ctx, key, mediaLimit(kind))
				if err != nil {
					return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
				}
				if digest(verify) != hash {
					return fmt.Errorf("frozen object verification failed")
				}
				manifest[kind] = revisionMedia{key, hash, ct, len(b)}
			}
			var after Asset
			if err := locked.db.First(&after, "kp_id = ?", a.KpID).Error; err != nil {
				return err
			}
			if sourceRevision(after) != item.SourceRevision {
				return ErrSourceChanged
			}
			frozen := FrozenMaterial{KpID: a.KpID, Text: strings.TrimSpace(a.CharText), ModuleCode: a.ModuleCode, ModuleName: a.ModuleName}
			content, _ := json.Marshal(frozen)
			mediaJSON, _ := json.Marshal(manifest)
			revID := digest(append(content, mediaJSON...))
			frozen.RevisionID = revID
			for kind, m := range manifest {
				ref := &MediaRef{revID, kind, m.SHA256}
				switch kind {
				case "writing_template":
					frozen.WritingTemplate = ref
				case "glyph":
					frozen.Glyph = ref
				case "sense":
					frozen.Sense = ref
				case "speech":
					frozen.Speech = ref
				}
			}
			content, _ = json.Marshal(frozen)
			row := materialRevision{RevisionID: revID, SubjectCode: "literacy", SourceRevision: item.SourceRevision, KpID: a.KpID, Content: string(content), Media: string(mediaJSON), CreatedAt: time.Now().UTC()}
			if err := locked.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
			out.Items = append(out.Items, frozen)
		}
		return nil
	})
	if err != nil {
		return FreezeResult{Items: []FrozenMaterial{}}, err
	}
	return out, nil
}
func (s *Service) RevisionMedia(ctx context.Context, id, kind string) ([]byte, string, error) {
	if kind != "glyph" && kind != "sense" && kind != "speech" && kind != "writing_template" {
		return nil, "", ErrInvalidFreeze
	}
	if len(id) != 64 {
		return nil, "", gorm.ErrRecordNotFound
	}
	if _, err := hex.DecodeString(id); err != nil {
		return nil, "", gorm.ErrRecordNotFound
	}
	var row materialRevision
	if err := s.db.WithContext(ctx).First(&row, "revision_id = ?", id).Error; err != nil {
		return nil, "", err
	}
	var media map[string]revisionMedia
	if err := json.Unmarshal([]byte(row.Media), &media); err != nil {
		return nil, "", err
	}
	m, ok := media[kind]
	if !ok {
		return nil, "", gorm.ErrRecordNotFound
	}
	if s.store == nil {
		return nil, "", ErrStoreUnavailable
	}
	b, err := s.store.GetBytesLimited(ctx, m.Key, mediaLimit(kind))
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
	}
	if len(b) != m.Size || digest(b) != m.SHA256 {
		return nil, "", fmt.Errorf("frozen media integrity check failed")
	}
	return b, m.ContentType, nil
}

func (s *Service) materialBytesForTypes(ctx context.Context, a Asset, types []string) (map[string][]byte, []string, error) {
	writing, choice := false, len(types) == 0
	for _, typ := range types {
		definition, _ := literacycontract.Lookup(typ)
		if definition.Interaction == "handwriting" {
			writing = true
		} else {
			choice = true
		}
	}
	media := map[string][]byte{}
	reasons := []string{}
	if choice {
		var err error
		media, reasons, err = s.materialBytes(ctx, a)
		if err != nil {
			return nil, nil, err
		}
	}
	if choice && len(types) > 0 {
		if _, ok := media["glyph"]; !ok {
			reasons = append(reasons, "missing_or_invalid_glyph_media")
		}
	}
	if writing {
		reasons = append(reasons, writingReasons(a)...)
		if len(reasons) > 0 {
			return media, reasons, nil
		}
		if s.store == nil {
			return nil, nil, ErrStoreUnavailable
		}
		if _, ok := media["speech"]; !ok {
			b, err := s.store.GetBytesLimited(ctx, s.store.SpeechKey(a.KpID), audioLimit)
			if err != nil {
				return nil, nil, fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
			}
			if !validMP3(ctx, b) {
				return media, []string{"invalid_speech_media"}, nil
			}
			media["speech"] = b
		}
		detail, err := s.WritingTemplate(ctx, a.KpID, a.WritingTemplateVersion)
		if err != nil {
			return media, []string{"invalid_writing_template"}, nil
		}
		b, err := json.Marshal(detail.Template)
		if err != nil {
			return nil, nil, err
		}
		media["writing_template"] = b
	}
	return media, reasons, nil
}
