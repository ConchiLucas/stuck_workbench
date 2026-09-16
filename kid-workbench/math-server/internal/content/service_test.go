package content

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadsDetailImageFromMaterialAdmin(t *testing.T) {
	file := strings.Repeat("e", 64) + ".png"
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/math/detail-image/"+file, r.URL.Path)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	t.Cleanup(upstream.Close)
	svc := NewService(upstream.URL)
	got, err := svc.Image(context.Background(), file)
	require.NoError(t, err)
	require.Equal(t, png, got)
	_, err = svc.Image(context.Background(), "../secret.png")
	require.Error(t, err)
}
