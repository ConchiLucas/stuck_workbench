package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/poem-server/internal/http"
)

func TestHealthz(t *testing.T) {
	router := httpapi.NewRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
}

func TestListPavilions(t *testing.T) {
	router := httpapi.NewRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/poem/pavilions", nil))
	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Data []struct {
			Code     string `json:"code"`
			KidTitle string `json:"kidTitle"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Len(t, body.Data, 7)
	require.Equal(t, "moon", body.Data[0].Code)
	require.Equal(t, "望月", body.Data[0].KidTitle)
	require.Equal(t, "scroll", body.Data[6].Code)
}

func TestMoonClues(t *testing.T) {
	router := httpapi.NewRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/poem/pavilions/moon/clues", nil))
	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Data []struct {
			Line     string `json:"line"`
			AnswerID string `json:"answerId"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, "床前明月光", body.Data[0].Line)
	require.Equal(t, "moon", body.Data[0].AnswerID)
}

func TestUnknownPavilion(t *testing.T) {
	router := httpapi.NewRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/poem/pavilions/zoo/clues", nil))
	require.Equal(t, http.StatusNotFound, response.Code)
}

func TestPlanAnswerLoop(t *testing.T) {
	router := httpapi.NewRouter()
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/children/1/poem/plans", bytes.NewBufferString(`{"pavilionCode":"scroll"}`)))
	require.Equal(t, http.StatusOK, create.Code)
	var created struct {
		Data struct {
			ID    int64 `json:"id"`
			Items []struct {
				ID     int64  `json:"id"`
				ClueID string `json:"clueId"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &created))
	require.NotZero(t, created.Data.ID)
	last := created.Data.Items[len(created.Data.Items)-1]
	require.Equal(t, "scroll-bird", last.ClueID)

	start := httptest.NewRecorder()
	router.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/api/v1/children/1/poem/plans/"+strconv.FormatInt(created.Data.ID, 10)+"/start", nil))
	require.Equal(t, http.StatusOK, start.Code)

	answer := httptest.NewRecorder()
	router.ServeHTTP(answer, httptest.NewRequest(http.MethodPost, "/api/v1/children/1/poem/plans/"+strconv.FormatInt(created.Data.ID, 10)+"/items/"+strconv.FormatInt(last.ID, 10)+"/answer", bytes.NewBufferString(`{"answerId":"still","clientId":"scroll-bird-1"}`)))
	require.Equal(t, http.StatusOK, answer.Code)
	var answered struct {
		Data struct {
			Correct bool `json:"correct"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(answer.Body.Bytes(), &answered))
	require.True(t, answered.Data.Correct)
}
