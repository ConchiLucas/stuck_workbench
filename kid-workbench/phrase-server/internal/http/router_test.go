package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/phrase-server/internal/http"
)

func TestHealthz(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
}

func TestReadyzWithoutChecker(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.JSONEq(t, `{"data":null,"error":{"code":"dependency_unavailable","message":"服务依赖尚未就绪"}}`, response.Body.String())
}
