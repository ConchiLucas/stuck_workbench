package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/english-server/internal/http"
	"github.com/conchi/english-server/internal/quiz"
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

type stubQuiz struct{}

func (stubQuiz) Generate(_ context.Context, quizType string, _ []int64) (quiz.Question, error) {
	if quizType != "listen" {
		return quiz.Question{}, quiz.ErrInvalidType
	}
	return quiz.Question{InstanceID: "q1", Type: "listen", TargetID: 101, Options: []quiz.Option{{ID: 101}}, AnswerIndex: 0}, nil
}

func TestGenerateQuizRoute(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Quiz: stubQuiz{}})
	body, err := json.Marshal(map[string]any{"type": "listen", "excludeTargetIds": []int64{1}})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/english/quiz/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	bad, err := json.Marshal(map[string]any{"type": "blend"})
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/english/quiz/generate", bytes.NewReader(bad))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestGenerateQuizUnavailable(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/english/quiz/generate", bytes.NewReader([]byte(`{"type":"listen"}`)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
