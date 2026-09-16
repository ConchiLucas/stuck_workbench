package literacy_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFrozenMaterialSurvivesOverwriteAndRejectsStaleSource(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	a := literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", ModuleName: "第一组", SenseImageURL: "sense", SpeechAudioURL: "speech"}
	require.NoError(t, gdb.Create(&a).Error)
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	store := &memStore{objects: map[string][]byte{}}
	store.PutPNG(context.Background(), store.SenseKey(1), b.Bytes())
	store.PutBytes(context.Background(), store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.True(t, list.Items[0].Capabilities["sense_char"].Ready)
	request := []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}}
	frozen, err := svc.FreezeMaterials(context.Background(), request)
	require.NoError(t, err)
	require.Len(t, frozen.Items, 1)
	store.PutBytes(context.Background(), store.SpeechKey(1), []byte("ID3new speech"), "audio/mpeg")
	got, ct, err := svc.RevisionMedia(context.Background(), frozen.Items[0].RevisionID, "speech")
	require.NoError(t, err)
	require.Equal(t, "audio/mpeg", ct)
	require.Equal(t, fixtureMP3(t), got)
	require.NoError(t, gdb.Model(&a).Update("char_text", "水").Error)
	_, err = svc.FreezeMaterials(context.Background(), request)
	require.ErrorIs(t, err, literacy.ErrSourceChanged)
	_, _, err = svc.RevisionMedia(context.Background(), frozen.Items[0].RevisionID, "../../secret")
	require.Error(t, err)
}
func TestMissingMediaNeverFreezes(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", SenseImageURL: "broken", SpeechAudioURL: "broken"}).Error)
	svc := literacy.NewService(gdb, &memStore{objects: map[string][]byte{}}, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	require.True(t, list.Items[0].Capabilities["glyph_sense"].Ready)
	_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}})
	require.Error(t, err)
	var count int64
	require.NoError(t, gdb.Table("material_revisions").Count(&count).Error)
	require.Zero(t, count)
}

func TestFreezeBatchRollsBackWhenLaterMaterialIsMissing(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	for _, id := range []int64{1, 2} {
		require.NoError(t, gdb.Create(&literacy.Asset{KpID: id, CharText: "山", ModuleCode: "g1", SenseImageURL: "sense", SpeechAudioURL: "speech"}).Error)
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	store := &memStore{objects: map[string][]byte{}}
	store.PutPNG(context.Background(), store.SenseKey(1), b.Bytes())
	store.PutBytes(context.Background(), store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}, {KpID: 2, SourceRevision: list.Items[1].SourceRevision}})
	require.ErrorIs(t, err, literacy.ErrMaterialNotReady)
	var count int64
	require.NoError(t, gdb.Table("material_revisions").Count(&count).Error)
	require.Zero(t, count)
	_, err = svc.FreezeMaterials(context.Background(), make([]literacy.FreezeItem, 81))
	require.ErrorIs(t, err, literacy.ErrInvalidFreeze)
	_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{{KpID: 1, SourceRevision: "a"}, {KpID: 1, SourceRevision: "a"}})
	require.ErrorIs(t, err, literacy.ErrInvalidFreeze)
}

func TestFrozenIdenticalMediaReusesRevisionAndDetectsCorruption(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", SenseImageURL: "sense", SpeechAudioURL: "speech"}).Error)
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	store := &memStore{objects: map[string][]byte{}}
	store.PutPNG(context.Background(), store.SenseKey(1), b.Bytes())
	store.PutBytes(context.Background(), store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	request := []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}}
	one, err := svc.FreezeMaterials(context.Background(), request)
	require.NoError(t, err)
	puts := store.puts
	two, err := svc.FreezeMaterials(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, one, two)
	require.Equal(t, puts, store.puts)
	key := "material-revisions/sha256/" + one.Items[0].Speech.SHA256 + ".mp3"
	store.objects[key] = []byte("corrupt")
	_, _, err = svc.RevisionMedia(context.Background(), one.Items[0].RevisionID, "speech")
	require.Error(t, err)
	_, err = svc.FreezeMaterials(context.Background(), request)
	require.Error(t, err)
	require.Equal(t, "corrupt", string(store.objects[key]))
}

type coordinatedStore struct {
	*memStore
	mu       sync.Mutex
	entered  chan struct{}
	release  chan struct{}
	pauseKey string
	once     sync.Once
}

func (s *coordinatedStore) GetBytes(ctx context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	b, err := s.memStore.GetBytes(ctx, key)
	s.mu.Unlock()
	if key == s.pauseKey {
		s.once.Do(func() { close(s.entered); <-s.release })
	}
	return b, err
}
func (s *coordinatedStore) GetPNG(ctx context.Context, key string) ([]byte, error) {
	return s.GetBytes(ctx, key)
}
func (s *coordinatedStore) PutBytes(ctx context.Context, key string, b []byte, ct string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.memStore.PutBytes(ctx, key, b, ct)
}
func (s *coordinatedStore) PutPNG(ctx context.Context, key string, b []byte) (string, error) {
	return s.PutBytes(ctx, key, b, "image/png")
}

type readyRenderer struct {
	b      []byte
	called chan struct{}
}

func (r readyRenderer) RenderPNG(text string) ([]byte, error) { close(r.called); return r.b, nil }

func TestFreezeCoordinatesWithConcurrentGlyphOverwrite(t *testing.T) {
	ctx := context.Background()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", GlyphImageURL: "glyph", SenseImageURL: "sense", SpeechAudioURL: "speech"}).Error)
	var original, next bytes.Buffer
	require.NoError(t, png.Encode(&original, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	require.NoError(t, png.Encode(&next, image.NewRGBA(image.Rect(0, 0, 3, 3))))
	store := &coordinatedStore{memStore: &memStore{objects: map[string][]byte{}}, entered: make(chan struct{}), release: make(chan struct{})}
	store.PutPNG(ctx, store.GlyphKey(1), original.Bytes())
	store.PutPNG(ctx, store.SenseKey(1), original.Bytes())
	store.PutBytes(ctx, store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	renderer := readyRenderer{next.Bytes(), make(chan struct{})}
	svc := literacy.NewService(gdb, store, renderer, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(ctx, "g1")
	require.NoError(t, err)
	store.pauseKey = store.SenseKey(1)
	type outcome struct {
		res literacy.FreezeResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		r, e := svc.FreezeMaterials(ctx, []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}})
		done <- outcome{r, e}
	}()
	<-store.entered
	generated := make(chan error, 1)
	go func() { _, e := svc.GenerateGlyph(ctx, 1); generated <- e }()
	<-renderer.called
	select {
	case e := <-generated:
		t.Fatalf("overwrite completed during freeze: %v", e)
	case <-time.After(30 * time.Millisecond):
	}
	close(store.release)
	frozen := <-done
	require.NoError(t, frozen.err)
	require.NoError(t, <-generated)
	b, _, err := svc.RevisionMedia(ctx, frozen.res.Items[0].RevisionID, "glyph")
	require.NoError(t, err)
	require.Equal(t, original.Bytes(), b)
	current, err := store.GetPNG(ctx, store.GlyphKey(1))
	require.NoError(t, err)
	require.Equal(t, next.Bytes(), current)
	_, err = svc.FreezeMaterials(ctx, []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}})
	require.ErrorIs(t, err, literacy.ErrSourceChanged)
}

type failingRevisionStore struct{ *memStore }

func (s *failingRevisionStore) PutBytes(ctx context.Context, key string, b []byte, ct string) (string, error) {
	if strings.HasPrefix(key, "material-revisions/") {
		return "", errNotFound
	}
	return s.memStore.PutBytes(ctx, key, b, ct)
}
func TestFailedObjectUploadCannotCreateReadyRevision(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", SenseImageURL: "sense", SpeechAudioURL: "speech"}).Error)
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	store := &failingRevisionStore{&memStore{objects: map[string][]byte{}}}
	store.PutPNG(context.Background(), store.SenseKey(1), b.Bytes())
	store.PutBytes(context.Background(), store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}})
	require.Error(t, err)
	var count int64
	require.NoError(t, gdb.Table("material_revisions").Count(&count).Error)
	require.Zero(t, count)
}

func fixtureMP3(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/tone.mp3")
	require.NoError(t, err)
	return b
}
func materialFixture(t *testing.T) (*gorm.DB, *memStore, *literacy.Service, literacy.FreezeItem) {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", SenseImageURL: "sense", SpeechAudioURL: "speech"}).Error)
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	store := &memStore{objects: map[string][]byte{}}
	store.PutPNG(context.Background(), store.SenseKey(1), b.Bytes())
	store.PutBytes(context.Background(), store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	return gdb, store, svc, literacy.FreezeItem{KpID: 1, SourceRevision: list.Items[0].SourceRevision}
}
func TestCandidateMetadataAndStableRevision(t *testing.T) {
	gdb, store, svc, item := materialFixture(t)
	require.Zero(t, store.gets, "candidate listing must not fetch objects")
	require.NoError(t, gdb.Model(&literacy.Asset{}).Where("kp_id = 1").Updates(map[string]any{"synced_at": time.Now(), "updated_at": time.Now()}).Error)
	list, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	require.Equal(t, item.SourceRevision, list.Items[0].SourceRevision)
	_, err = svc.GenerationMaterials(context.Background(), "unknown")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
func TestFreezeRejectsInvalidAndOversizeMedia(t *testing.T) {
	for name, b := range map[string][]byte{"id3_only": []byte("ID3"), "sync_only": {0xff, 0xfb}, "truncated": fixtureMP3(t)[:50], "oversize": append(fixtureMP3(t), make([]byte, 20*1024*1024)...)} {
		t.Run(name, func(t *testing.T) {
			_, store, svc, item := materialFixture(t)
			store.objects[store.SpeechKey(1)] = b
			_, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
			require.Error(t, err)
		})
	}
}
func TestFailedMetadataSaveInvalidatesOldSource(t *testing.T) {
	gdb, store, _, item := materialFixture(t)
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 3, 3))))
	renderer := readyRenderer{b.Bytes(), make(chan struct{})}
	svc := literacy.NewService(gdb, store, renderer, nil, nil, nil, nil)
	// Allow a durable invalidation update but fail the final full Asset save after the object write.
	require.NoError(t, gdb.Callback().Update().Before("gorm:update").Register("fail_final_asset_save", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*literacy.Asset); ok {
			tx.AddError(fmt.Errorf("injected metadata save failure"))
		}
	}))
	_, err := svc.GenerateGlyph(context.Background(), 1)
	require.Error(t, err)
	_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
	require.ErrorIs(t, err, literacy.ErrSourceChanged)
	fresh, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	require.False(t, fresh.Items[0].Capabilities["glyph_sense"].Ready)
	_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{{KpID: 1, SourceRevision: fresh.Items[0].SourceRevision}})
	require.ErrorIs(t, err, literacy.ErrMaterialNotReady)
	require.NoError(t, gdb.Callback().Update().Remove("fail_final_asset_save"))
	svc = literacy.NewService(gdb, store, readyRenderer{b.Bytes(), make(chan struct{})}, nil, nil, nil, nil)
	_, err = svc.GenerateGlyph(context.Background(), 1)
	require.NoError(t, err)
	fresh, err = svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	require.True(t, fresh.Items[0].Capabilities["glyph_sense"].Ready)
	_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{{KpID: 1, SourceRevision: fresh.Items[0].SourceRevision}})
	require.NoError(t, err)
	got, err := svc.GlyphPNG(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, b.Bytes(), got)
}

func (s *coordinatedStore) GetBytesLimited(ctx context.Context, key string, limit int64) ([]byte, error) {
	return s.GetBytes(ctx, key)
}

type faultReadStore struct {
	*memStore
	failure     error
	sawDeadline bool
}

func (s *faultReadStore) GetBytesLimited(ctx context.Context, key string, limit int64) ([]byte, error) {
	deadline, ok := ctx.Deadline()
	s.sawDeadline = ok && time.Until(deadline) <= 30*time.Second
	if strings.HasPrefix(key, "material-revisions/") && s.failure != nil {
		return nil, s.failure
	}
	return s.memStore.GetBytesLimited(ctx, key, limit)
}
func TestFreezeStorageFailuresNeverOverwrite(t *testing.T) {
	for _, failure := range []error{errors.New("permission denied"), context.DeadlineExceeded} {
		t.Run(failure.Error(), func(t *testing.T) {
			gdb, base, _, item := materialFixture(t)
			store := &faultReadStore{memStore: base, failure: failure}
			svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
			puts := store.puts
			_, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
			require.ErrorIs(t, err, literacy.ErrStoreUnavailable)
			require.Equal(t, puts, store.puts)
			require.True(t, store.sawDeadline)
		})
	}
	t.Run("empty_existing_object", func(t *testing.T) {
		_, store, svc, item := materialFixture(t)
		frozen, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
		require.NoError(t, err)
		key := "material-revisions/sha256/" + frozen.Items[0].Speech.SHA256 + ".mp3"
		store.objects[key] = []byte{}
		puts := store.puts
		_, err = svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
		require.Error(t, err)
		require.Equal(t, puts, store.puts)
		require.Empty(t, store.objects[key])
	})
}
func TestFreezeRejectsLargePNGDimensionsAndBytes(t *testing.T) {
	for _, mode := range []string{"dimensions", "bytes"} {
		t.Run(mode, func(t *testing.T) {
			_, store, svc, item := materialFixture(t)
			b := append([]byte(nil), store.objects[store.SenseKey(1)]...)
			if mode == "dimensions" {
				binary.BigEndian.PutUint32(b[16:20], 100000)
				binary.BigEndian.PutUint32(b[20:24], 100000)
				binary.BigEndian.PutUint32(b[29:33], crc32.ChecksumIEEE(b[12:29]))
			} else {
				b = append(b, make([]byte, 10*1024*1024)...)
			}
			store.objects[store.SenseKey(1)] = b
			_, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
			require.ErrorIs(t, err, literacy.ErrMaterialNotReady)
		})
	}
}
func TestFreezeHonorsCanceledContext(t *testing.T) {
	_, _, svc, item := materialFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.FreezeMaterials(ctx, []literacy.FreezeItem{item})
	require.ErrorIs(t, err, context.Canceled)
}

func TestFreezeDeadlineWhileWaitingForMaterialLock(t *testing.T) {
	gdb, base, _, item := materialFixture(t)
	store := &coordinatedStore{memStore: base, entered: make(chan struct{}), release: make(chan struct{}), pauseKey: base.SenseKey(1)}
	svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
	first := make(chan error, 1)
	go func() { _, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item}); first <- err }()
	<-store.entered
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	second := make(chan error, 1)
	go func() { _, err := svc.FreezeMaterials(ctx, []literacy.FreezeItem{item}); second <- err }()
	var timedOut bool
	select {
	case err := <-second:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(150 * time.Millisecond):
		timedOut = true
	}
	close(store.release)
	require.NoError(t, <-first)
	if timedOut {
		require.ErrorIs(t, <-second, context.DeadlineExceeded)
	}
	require.False(t, timedOut, "lock acquisition must honor the request deadline")
}

func TestFreezePreservesLegacyJPEGWithActualContentType(t *testing.T) {
	_, store, svc, item := materialFixture(t)
	var source bytes.Buffer
	require.NoError(t, jpeg.Encode(&source, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil))
	store.objects[store.SenseKey(1)] = append([]byte(nil), source.Bytes()...)
	frozen, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
	require.NoError(t, err)
	data, ct, err := svc.RevisionMedia(context.Background(), frozen.Items[0].RevisionID, "sense")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", ct)
	require.Equal(t, source.Bytes(), data)
	_, err = jpeg.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, source.Bytes(), store.objects[store.SenseKey(1)], "legacy JPEG object must remain untouched")
	again, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
	require.NoError(t, err)
	require.Equal(t, frozen, again)
}

func TestFreezeAcceptsBoundedLargeJPEG(t *testing.T) {
	_, store, svc, item := materialFixture(t)
	// A real legacy generator size: 16.9 MP, below the explicit 20 MP cap.
	var source bytes.Buffer
	require.NoError(t, jpeg.Encode(&source, image.NewGray(image.Rect(0, 0, 5504, 3072)), nil))
	store.objects[store.SenseKey(1)] = source.Bytes()
	frozen, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{item})
	require.NoError(t, err)
	data, ct, err := svc.RevisionMedia(context.Background(), frozen.Items[0].RevisionID, "sense")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", ct)
	require.Equal(t, source.Bytes(), data)
}
