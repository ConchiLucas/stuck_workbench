package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/conchi/phrase-server/internal/plan"
	"github.com/conchi/phrase-server/internal/practice"
	"github.com/conchi/phrase-server/internal/testdb"
	"github.com/conchi/study-learning/mastery"
	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/phrase-server/internal/http"
)

func TestCreateTypePlanAndAnswer(t *testing.T) {
	db := testdb.Open(t, "phrase-http")
	router := httpapi.NewRouter(httpapi.Deps{
		Plans:    plan.NewService(db),
		Practice: practice.NewService(db, mastery.DefaultConfig()),
	})
	create := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/children/1/phrase/plans", bytes.NewBufferString(`{"mode":"type","questionCode":"listen_zh","count":2}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, createReq)
	require.Equal(t, http.StatusCreated, create.Code)
	var envelope struct {
		Data  plan.Detail
		Error *struct{ Code, Message string }
	}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &envelope))
	require.Nil(t, envelope.Error)
	require.Equal(t, "phrase", envelope.Data.Plan.SubjectCode)
	require.Len(t, envelope.Data.Items, 2)
	require.NotContains(t, create.Body.String(), `"answerIndex"`)
	require.NotContains(t, create.Body.String(), `"answer":`)
	item := envelope.Data.Items[0]
	require.Equal(t, "pending", item.Status)
	answer := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/children/1/phrase/plans/"+strconv.FormatInt(envelope.Data.Plan.ID, 10)+"/items/"+strconv.FormatInt(item.ID, 10)+"/answer", bytes.NewBufferString(`{"clientId":"try-1","optionIndex":0,"costMs":200}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(answer, req)
	require.Equal(t, http.StatusOK, answer.Code)
	var answered struct {
		Data struct {
			Status string `json:"status"`
			Tries  int    `json:"tries"`
		}
		Error *struct{ Code string }
	}
	require.NoError(t, json.Unmarshal(answer.Body.Bytes(), &answered))
	require.Nil(t, answered.Error)
	require.Equal(t, 1, answered.Data.Tries)
	require.Contains(t, []string{"correct", "wrong", "pending"}, answered.Data.Status)
	require.NotContains(t, answer.Body.String(), `"answerIndex"`)
}
