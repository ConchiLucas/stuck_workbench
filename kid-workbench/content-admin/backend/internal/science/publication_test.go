package science_test

import (
	"context"
	"testing"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/science"
	"github.com/stretchr/testify/require"
)

type recordingStore struct {
	pngKey string
	onPut  func()
}

func (s *recordingStore) PutPNG(_ context.Context, key string, _ []byte) (string, error) {
	s.pngKey = key
	if s.onPut != nil {
		s.onPut()
	}
	return key, nil
}

func TestGenerateGlyphDoesNotOverwriteNewerContentVersion(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&science.Asset{
		KpID: 8, Title: "候鸟", NeedsSenseImage: true, ReviewStatus: "published", ContentVersion: 3,
	}).Error)
	store := &recordingStore{onPut: func() {
		require.NoError(t, gdb.Model(&science.Asset{}).Where("kp_id = ?", 8).
			Updates(map[string]any{"content_version": 5, "summary": "更新后的内容"}).Error)
	}}
	svc := science.NewService(gdb, store, glyphRenderer{}, nil, nil, nil, nil)

	_, err = svc.GenerateGlyph(context.Background(), 8)
	require.ErrorIs(t, err, science.ErrAssetSuperseded)
	var got science.Asset
	require.NoError(t, gdb.First(&got, "kp_id = ?", 8).Error)
	require.Equal(t, 5, got.ContentVersion)
	require.Equal(t, "更新后的内容", got.Summary)
	require.Empty(t, got.GlyphObjectKey)
}
func (*recordingStore) GetPNG(context.Context, string) ([]byte, error) { return nil, nil }
func (*recordingStore) PutBytes(context.Context, string, []byte, string) (string, error) {
	return "", nil
}
func (*recordingStore) GetBytes(context.Context, string) ([]byte, error) { return nil, nil }
func (*recordingStore) ScienceGlyphKey(int64) string                     { return "legacy-glyph" }
func (*recordingStore) ScienceSenseKey(int64) string                     { return "legacy-sense" }
func (*recordingStore) ScienceSpeechKey(int64) string                    { return "legacy-speech" }

type glyphRenderer struct{}

func (glyphRenderer) RenderEnglishPNG(string) ([]byte, error) { return []byte("png"), nil }

func TestMigrateAddsSciencePublicationColumns(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`CREATE TABLE science_assets (
		kp_id INTEGER PRIMARY KEY,
		title TEXT NOT NULL,
		needs_sense_image INTEGER NOT NULL,
		module_code TEXT NOT NULL DEFAULT '',
		module_name TEXT NOT NULL DEFAULT '',
		module_order INTEGER NOT NULL DEFAULT 0,
		kp_order INTEGER NOT NULL DEFAULT 0,
		needs_sense_image_override INTEGER,
		glyph_image_url TEXT NOT NULL DEFAULT '',
		sense_image_url TEXT NOT NULL DEFAULT '',
		speech_audio_url TEXT NOT NULL DEFAULT '',
		synced_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO science_assets
		(kp_id,title,needs_sense_image,synced_at,updated_at)
		VALUES (1,'冬眠',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`).Error)

	require.NoError(t, db.Migrate(gdb))

	var got struct {
		ReviewStatus   string `gorm:"column:review_status"`
		ContentVersion int    `gorm:"column:content_version"`
	}
	require.NoError(t, gdb.Table("science_assets").First(&got, "kp_id = 1").Error)
	require.Equal(t, "published", got.ReviewStatus)
	require.Equal(t, 1, got.ContentVersion)
}

func TestGenerateGlyphReservesDraftVersionBeforeUsingVersionedKey(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&science.Asset{
		KpID: 7, Title: "冬眠", NeedsSenseImage: true, ReviewStatus: "published", ContentVersion: 3,
	}).Error)
	store := &recordingStore{}
	svc := science.NewService(gdb, store, glyphRenderer{}, nil, nil, nil, nil)

	got, err := svc.GenerateGlyph(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "science/glyphs/7-v4.png", store.pngKey)
	require.Equal(t, 4, got.ContentVersion)
	require.Equal(t, "draft", got.ReviewStatus)
	require.Nil(t, got.ReviewedAt)
}
