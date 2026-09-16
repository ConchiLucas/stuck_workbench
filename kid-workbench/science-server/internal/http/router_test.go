package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/study-science/internal/http"
	"github.com/conchi/study-science/internal/quiz"
)

func TestHealthz(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
}

func TestReadyzReportsMissingDatabase(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "database_unavailable")
}

type stubQuiz struct{}

func (stubQuiz) Generate(_ context.Context, quizType string, _ []int64) (quiz.Question, error) {
	if quizType != "choice" {
		return quiz.Question{}, quiz.ErrInvalidType
	}
	return quiz.Question{InstanceID: "q1", Type: "choice", TargetID: 10, Options: []quiz.Option{{ID: 0, Label: "鸭子"}}, AnswerIndex: 0}, nil
}

func TestGenerateQuizRoute(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Quiz: stubQuiz{}})
	body, err := json.Marshal(map[string]any{"type": "choice", "excludeTargetIds": []int64{1}})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/science/quiz/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	bad, err := json.Marshal(map[string]any{"type": "match"})
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/science/quiz/generate", bytes.NewReader(bad))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestGenerateQuizUnavailable(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/science/quiz/generate", bytes.NewReader([]byte(`{"type":"choice"}`)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
