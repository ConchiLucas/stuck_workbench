package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/conchi/study-content-admin/internal/db"
	httpapi "github.com/conchi/study-content-admin/internal/http"
	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/conchi/study-content-admin/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestMaterialRoutesListReadinessAndValidateFreeze(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", ModuleName: "第一组"}).Error)
	router := httpapi.NewRouter(httpapi.Deps{Literacy: literacy.NewService(gdb, nil, nil, nil, nil, nil, nil)})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/generation-materials/literacy?moduleCode=g1", nil))
	require.Equal(t, 200, rec.Code)
	var result literacy.GenerationMaterialsResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Equal(t, "literacy", result.SubjectCode)
	require.Len(t, result.Items, 1)
	require.Equal(t, "第一组", result.Items[0].ModuleName)
	require.False(t, result.Items[0].Capabilities["sense_char"].Ready)
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"items":[]}`, 400}, {`{"items":[{"kpId":1,"sourceRevision":"stale"}]}`, 409}, {`{"items":[{"kpId":1,"sourceRevision":"` + result.Items[0].SourceRevision + `"}]}`, 422}} {
		rec = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/generation-materials/literacy/freeze", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/material-revisions/invalid/media/speech", nil))
	require.Equal(t, 404, rec.Code)
}

type materialHTTPStore struct{ objects map[string][]byte }

func (s *materialHTTPStore) GetBytes(ctx context.Context, key string) ([]byte, error) {
	b, ok := s.objects[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return b, nil
}
func (s *materialHTTPStore) GetBytesLimited(ctx context.Context, key string, limit int64) ([]byte, error) {
	b, e := s.GetBytes(ctx, key)
	if int64(len(b)) > limit {
		return nil, storage.ErrObjectTooLarge
	}
	return b, e
}
func (s *materialHTTPStore) GetPNG(ctx context.Context, key string) ([]byte, error) {
	return s.GetBytes(ctx, key)
}
func (s *materialHTTPStore) PutBytes(ctx context.Context, key string, b []byte, ct string) (string, error) {
	s.objects[key] = append([]byte(nil), b...)
	return key, nil
}
func (s *materialHTTPStore) PutPNG(ctx context.Context, key string, b []byte) (string, error) {
	return s.PutBytes(ctx, key, b, "image/png")
}
func (s *materialHTTPStore) GlyphKey(id int64) string  { return storage.GlyphObjectKey(id) }
func (s *materialHTTPStore) SenseKey(id int64) string  { return storage.SenseObjectKey(id) }
func (s *materialHTTPStore) SpeechKey(id int64) string { return storage.SpeechObjectKey(id) }

func TestMaterialMediaETagAndUnavailableErrors(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山", ModuleCode: "g1", SenseImageURL: "sense", SpeechAudioURL: "speech"}).Error)
	var pngData bytes.Buffer
	require.NoError(t, png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	speech, err := os.ReadFile("../literacy/testdata/tone.mp3")
	require.NoError(t, err)
	store := &materialHTTPStore{objects: map[string][]byte{storage.SenseObjectKey(1): pngData.Bytes(), storage.SpeechObjectKey(1): speech}}
	svc := literacy.NewService(gdb, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(context.Background(), "g1")
	require.NoError(t, err)
	frozen, err := svc.FreezeMaterials(context.Background(), []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision}})
	require.NoError(t, err)
	router := httpapi.NewRouter(httpapi.Deps{Literacy: svc})
	url := "/api/v1/material-revisions/" + frozen.Items[0].RevisionID + "/media/speech"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	require.Equal(t, 200, rec.Code)
	require.Equal(t, speech, rec.Body.Bytes())
	require.Equal(t, "audio/mpeg", rec.Header().Get("Content-Type"))
	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("If-None-Match", "W/"+etag)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, 304, rec.Code)
	require.Zero(t, rec.Body.Len())
	delete(store.objects, "material-revisions/sha256/"+frozen.Items[0].Speech.SHA256+".mp3")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	require.Equal(t, 503, rec.Code)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/generation-materials/literacy?moduleCode=unknown", nil))
	require.Equal(t, 404, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, strings.Replace(url, "speech", "unknown", 1), nil))
	require.Equal(t, 400, rec.Code)
}

func TestWritingTemplateRoutesRejectMalformedAndOversize(t *testing.T) {
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, g.Create(&literacy.Asset{KpID: 1, CharText: "一"}).Error)
	router := httpapi.NewRouter(httpapi.Deps{Literacy: literacy.NewService(g, nil, nil, nil, nil, nil, nil)})
	for _, tc := range []struct {
		body   string
		status int
	}{{`{}`, 400}, {strings.Repeat("x", 2<<20+1), 413}} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/literacy/chars/1/writing-template", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
	}
}

func TestWritingTemplateRoutesImportAndReadVersion(t *testing.T) {
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, g.Create(&literacy.Asset{KpID: 1, CharText: "一"}).Error)
	router := httpapi.NewRouter(httpapi.Deps{Literacy: literacy.NewService(g, nil, nil, nil, nil, nil, nil)})
	b, err := os.ReadFile("../literacy/testdata/writing-templates/一.json")
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/literacy/chars/1/writing-template", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	var detail literacy.WritingTemplateDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &detail))
	require.Equal(t, "valid", detail.ValidationStatus)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/literacy/chars/1/writing-template?version="+detail.Version, nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), "Arphic")
}
