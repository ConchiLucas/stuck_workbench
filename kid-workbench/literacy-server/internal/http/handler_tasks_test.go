package http_test

import (
	api "github.com/conchi/literacy-server/internal/http"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRevisionMediaProxyOnlyAcceptsFrozenReferences(t *testing.T) {
	hash := strings.Repeat("a", 64)
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, "/api/v1/material-revisions/"+hash+"/media/speech", r.URL.Path)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte("ID3frozen"))
	}))
	defer upstream.Close()
	t.Setenv("APP_CONTENT_ADMIN_URL", upstream.URL)
	router := api.NewRouter(api.Deps{})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/literacy/material-revisions/"+hash+"/media/speech", nil))
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "ID3frozen", rec.Body.String())
	require.Contains(t, rec.Header().Get("Cache-Control"), "immutable")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/literacy/material-revisions/localhost/media/speech", nil))
	require.Equal(t, 400, rec.Code)
	require.Equal(t, 1, requests)
}

func TestRevisionMediaPreservesJPEGContentType(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte{0xff, 0xd8, 0xff, 0xd9})
	}))
	defer upstream.Close()
	t.Setenv("APP_CONTENT_ADMIN_URL", upstream.URL)
	rec := httptest.NewRecorder()
	api.NewRouter(api.Deps{}).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/literacy/material-revisions/"+strings.Repeat("b", 64)+"/media/sense", nil))
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
}
