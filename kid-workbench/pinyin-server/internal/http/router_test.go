package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/conchi/study-learning/pinyincontract"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/pinyin-server/internal/http"
	"github.com/conchi/pinyin-server/internal/quiz"
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
	if quizType != "listen" {
		return quiz.Question{}, quiz.ErrInvalidType
	}
	return quiz.Question{
		InstanceID: "quiz-1", Type: "listen", TargetID: 4,
		Options: []quiz.Option{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}, AnswerIndex: 3,
	}, nil
}

func TestGenerateQuizRoute(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Quiz: stubQuiz{}})

	body, err := json.Marshal(map[string]any{"type": "listen", "excludeTargetIds": []int64{1, 2, 3}})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/pinyin/quiz/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	var envelope struct {
		Data  quiz.Question `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Nil(t, envelope.Error)
	require.Equal(t, "listen", envelope.Data.Type)
	require.Equal(t, int64(4), envelope.Data.TargetID)

	bad, err := json.Marshal(map[string]any{"type": "unknown"})
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/pinyin/quiz/generate", bytes.NewReader(bad))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func (stubQuiz) GenerateForChild(_ context.Context, childID int64, kind string, _ []int64) (pinyincontract.GeneratedQuestion, error) {
	if childID != 1 {
		return pinyincontract.GeneratedQuestion{}, quiz.ErrNotFound
	}
	if kind != "shape" {
		return pinyincontract.GeneratedQuestion{}, quiz.ErrInvalidType
	}
	return pinyincontract.GeneratedQuestion{InstanceID: "formal", Type: kind, KpID: 100, TargetID: 100, Options: []pinyincontract.Option{{ID: "100"}}}, nil
}
func (stubQuiz) Answer(_ context.Context, childID int64, id string, in pinyincontract.AnswerRequest) (pinyincontract.AnswerResult, error) {
	if childID != 1 {
		return pinyincontract.AnswerResult{}, quiz.ErrNotFound
	}
	switch in.OptionID {
	case "invalid":
		return pinyincontract.AnswerResult{}, quiz.ErrInvalidRequest
	case "conflict":
		return pinyincontract.AnswerResult{}, quiz.ErrConflict
	case "expired":
		return pinyincontract.AnswerResult{}, quiz.ErrExpired
	case "unavailable":
		return pinyincontract.AnswerResult{}, pinyincatalog.ErrUnavailable
	}
	return pinyincontract.AnswerResult{InstanceID: id, SelectedOptionID: in.OptionID, Correct: true}, nil
}
func (stubQuiz) GetInstance(_ context.Context, childID int64, id string) (pinyincontract.InstanceSnapshot, error) {
	if childID != 1 {
		return pinyincontract.InstanceSnapshot{}, quiz.ErrNotFound
	}
	return pinyincontract.InstanceSnapshot{GeneratedQuestion: pinyincontract.GeneratedQuestion{InstanceID: id}}, nil
}
func TestFormalQuizRoutes(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Quiz: stubQuiz{}, FormalQuiz: stubQuiz{}})
	cases := []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/api/v1/children/1/pinyin/quiz/generate", `{"type":"shape"}`, 200},
		{"POST", "/api/v1/children/2/pinyin/quiz/generate", `{"type":"shape"}`, 404},
		{"POST", "/api/v1/children/1/pinyin/quiz/generate", `{"type":"unknown"}`, 400},
		{"GET", "/api/v1/children/1/pinyin/quiz/formal", "", 200},
		{"GET", "/api/v1/children/2/pinyin/quiz/formal", "", 404},
		{"GET", "/api/v1/children/0/pinyin/quiz/formal", "", 400},
		{"POST", "/api/v1/children/1/pinyin/quiz/formal/answer", `{"clientId":"one","optionId":"100","costMs":0}`, 200},
		{"POST", "/api/v1/children/1/pinyin/quiz/formal/answer", `{"optionId":100}`, 400},
		{"POST", "/api/v1/children/1/pinyin/quiz/formal/answer", `{"optionId":"invalid"}`, 400},
		{"POST", "/api/v1/children/1/pinyin/quiz/formal/answer", `{"optionId":"conflict"}`, 409},
		{"POST", "/api/v1/children/1/pinyin/quiz/formal/answer", `{"optionId":"expired"}`, 410},
		{"POST", "/api/v1/children/1/pinyin/quiz/formal/answer", `{"optionId":"unavailable"}`, 503},
	}
	for _, tc := range cases {
		t.Run(tc.method+tc.path+tc.body, func(t *testing.T) {
			r := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(r, req)
			require.Equal(t, tc.status, r.Code, r.Body.String())
			if tc.method == "POST" && strings.HasSuffix(tc.path, "generate") && tc.status == 200 {
				require.NotContains(t, r.Body.String(), "answerIndex")
				require.Contains(t, r.Body.String(), `"id":"100"`)
			}
		})
	}
}
