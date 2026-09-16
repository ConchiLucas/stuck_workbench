package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/study-learning/mathcontent"
	httpapi "github.com/conchi/math-server/internal/http"
	"github.com/stretchr/testify/require"
)

type contentStub struct {
	images map[string][]byte
}

func (contentStub) List(context.Context) (mathcontent.Catalog, error) {
	return mathcontent.Catalog{SchemaVersion: 1}, nil
}
func (contentStub) Audio(context.Context, string) ([]byte, error) {
	return nil, errors.New("missing")
}
func (s contentStub) Image(_ context.Context, file string) ([]byte, error) {
	if data, ok := s.images[file]; ok {
		return data, nil
	}
	return nil, errors.New("missing")
}

func TestServesImmutableDetailImage(t *testing.T) {
	file := strings.Repeat("d", 64) + ".png"
	png := []byte{0x89, 0x50, 0x4e, 0x47}
	router := httpapi.NewRouter(httpapi.Deps{Content: contentStub{images: map[string][]byte{file: png}}})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/math/detail-image/"+file, nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "image/png", response.Header().Get("Content-Type"))
	require.Equal(t, png, response.Body.Bytes())
}
