package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/logic-server/internal/http"
	"github.com/conchi/logic-server/internal/quiz"
)

func TestHealthz(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
}

type stubQuiz struct{}

func (stubQuiz) Generate(_ context.Context, quizType string, _ []int64) (quiz.Question, error) {
	if quizType != "pattern" {
		return quiz.Question{}, quiz.ErrInvalidType
	}
	return quiz.Question{
		InstanceID: "q1", Type: "pattern", TargetID: 10, Stem: "下一个是哪个？",
		Options: []quiz.Option{{ID: 0, Emoji: "🔴"}}, AnswerIndex: 0,
		Visual: quiz.Visual{Kind: "seq", Items: []string{"🔴", "🔵"}},
	}, nil
}

func TestGenerateQuizRoute(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Quiz: stubQuiz{}})
	body, err := json.Marshal(map[string]any{"type": "pattern", "excludeTargetIds": []int64{1}})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/logic/quiz/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	bad, err := json.Marshal(map[string]any{"type": "blend"})
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/logic/quiz/generate", bytes.NewReader(bad))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestGenerateQuizUnavailable(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/logic/quiz/generate", bytes.NewReader([]byte(`{"type":"pattern"}`)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
