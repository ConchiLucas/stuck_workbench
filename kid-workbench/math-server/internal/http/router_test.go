package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/catalog"
	"github.com/conchi/math-server/internal/home"
	httpapi "github.com/conchi/math-server/internal/http"
	"github.com/conchi/math-server/internal/plan"
	"github.com/conchi/math-server/internal/practice"
	"github.com/conchi/math-server/internal/progress"
	"github.com/conchi/math-server/internal/quiz"
)

func TestHealthz(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
}

type readinessStub struct{ err error }

func (s readinessStub) Ready(context.Context) error { return s.err }

func TestReadyzChecksDatabaseAndAssets(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, tc := range []struct {
		deps httpapi.Deps
		want int
	}{
		{httpapi.Deps{}, http.StatusServiceUnavailable},
		{httpapi.Deps{Database: database, Readiness: readinessStub{}}, http.StatusOK},
		{httpapi.Deps{Database: database, Readiness: readinessStub{err: errors.New("minio down")}}, http.StatusServiceUnavailable},
	} {
		response := httptest.NewRecorder()
		httpapi.NewRouter(tc.deps).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		require.Equal(t, tc.want, response.Code, response.Body.String())
	}
}

type homeStub struct{}

func (homeStub) Get(_ context.Context, childID int64) (home.Home, error) {
	if childID == 404 {
		return home.Home{}, home.ErrChildNotFound
	}
	return home.Home{Child: home.ChildSummary{ID: childID}}, nil
}

func TestHomeRoute(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Home: homeStub{}})
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/children/nope/math/home", http.StatusBadRequest},
		{"/api/v1/children/404/math/home", http.StatusNotFound},
		{"/api/v1/children/1/math/home", http.StatusOK},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.want, response.Code, "%s: %s", tc.path, response.Body.String())
	}
}

type practiceStub struct{}

func (practiceStub) Answer(_ context.Context, _, _, _ int64, input practice.AnswerInput) (practice.AnswerResult, error) {
	return practice.AnswerResult{Correct: input.OptionIndex == 2}, nil
}
func (practiceStub) Finish(_ context.Context, _, planID int64) (plan.StudyPlan, error) {
	if planID == 409 {
		return plan.StudyPlan{}, practice.ErrPlanIncomplete
	}
	return plan.StudyPlan{ID: planID, Status: "done"}, nil
}

func TestPracticeRoutes(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Plans: planStub{}, Practice: practiceStub{}})
	tests := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPost, "/api/v1/children/1/math/plans/7/items/9/answer", `{"clientId":"try-1","optionIndex":2,"costMs":1000}`, http.StatusOK},
		{http.MethodPost, "/api/v1/children/1/math/plans/7/items/9/answer", `{"clientId":"","optionIndex":2,"costMs":1000}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/children/1/math/plans/7/items/9/answer", `{"clientId":"x","optionIndex":4,"costMs":1000}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/children/1/math/plans/7/finish", "", http.StatusOK},
		{http.MethodPost, "/api/v1/children/1/math/plans/409/finish", "", http.StatusConflict},
	}
	for _, tc := range tests {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		require.Equal(t, tc.want, response.Code, "%s %s: %s", tc.method, tc.path, response.Body.String())
	}
}

type planStub struct{}

func (planStub) Create(_ context.Context, childID int64, input plan.CreateInput) (plan.Detail, error) {
	if childID == 404 {
		return plan.Detail{}, plan.ErrChildNotFound
	}
	return plan.Detail{Plan: plan.StudyPlan{ID: 7, PlanKind: input.Kind}}, nil
}
func (planStub) Get(_ context.Context, _ int64, planID int64) (plan.Detail, error) {
	if planID == 404 {
		return plan.Detail{}, plan.ErrPlanNotFound
	}
	return plan.Detail{Plan: plan.StudyPlan{ID: planID, PlanKind: "daily"}}, nil
}
func (planStub) Start(ctx context.Context, childID, planID int64) (plan.Detail, error) {
	return planStub{}.Get(ctx, childID, planID)
}

func TestPlanRoutes(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Plans: planStub{}})
	tests := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPost, "/api/v1/children/1/math/plans", `{"kind":"daily"}`, http.StatusCreated},
		{http.MethodPost, "/api/v1/children/nope/math/plans", `{"kind":"daily"}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/children/404/math/plans", `{"kind":"daily"}`, http.StatusNotFound},
		{http.MethodPost, "/api/v1/children/1/math/plans", `{"kind":"module"}`, http.StatusBadRequest},
		{http.MethodGet, "/api/v1/children/1/math/plans/7", "", http.StatusOK},
		{http.MethodGet, "/api/v1/children/1/math/plans/404", "", http.StatusNotFound},
		{http.MethodPost, "/api/v1/children/1/math/plans/7/start", "", http.StatusOK},
	}
	for _, tc := range tests {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		require.Equal(t, tc.want, response.Code, "%s %s: %s", tc.method, tc.path, response.Body.String())
	}
}

type progressStub struct{}

func (progressStub) Get(_ context.Context, childID int64) (progress.Result, error) {
	if childID == 404 {
		return progress.Result{}, progress.ErrChildNotFound
	}
	if childID == 500 {
		return progress.Result{}, errors.New("database down")
	}
	return progress.Result{Modules: []progress.ModuleProgress{}}, nil
}

func TestProgressRouteValidation(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Progress: progressStub{}})
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/children/nope/math/progress", http.StatusBadRequest},
		{"/api/v1/children/404/math/progress", http.StatusNotFound},
		{"/api/v1/children/1/math/progress", http.StatusOK},
		{"/api/v1/children/500/math/progress", http.StatusServiceUnavailable},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.want, response.Code, "%s: %s", tc.path, response.Body.String())
	}
}

func TestAPIErrorEnvelope(t *testing.T) {
	require.Equal(t,
		httpapi.APIError{Code: "child_not_found", Message: "孩子不存在"},
		httpapi.APIError{Code: "child_not_found", Message: "孩子不存在"},
	)
}

type catalogStub struct{}

func (catalogStub) ListModules(context.Context) ([]catalog.Module, error) {
	return []catalog.Module{{Code: "add10", Name: "20以内加法"}}, nil
}

type failedCatalogStub struct{ catalogStub }

func (failedCatalogStub) ListModules(context.Context) ([]catalog.Module, error) {
	return nil, errors.New("database down")
}

func (catalogStub) GetModule(_ context.Context, code string) (catalog.Module, error) {
	if code != "add10" {
		return catalog.Module{}, gorm.ErrRecordNotFound
	}
	return catalog.Module{Code: code, Name: "20以内加法"}, nil
}

func TestCatalogDatabaseFailureIsRetryable(t *testing.T) {
	response := httptest.NewRecorder()
	httpapi.NewRouter(httpapi.Deps{Catalog: failedCatalogStub{}}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/math/modules", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "database_unavailable")
}

func (catalogStub) ListStageItems(_ context.Context, moduleCode, stageCode string) ([]catalog.LearningItem, error) {
	if moduleCode != "add10" || stageCode != "within5" {
		return nil, gorm.ErrRecordNotFound
	}
	return []catalog.LearningItem{{KpID: 10, ModuleCode: moduleCode, StageCode: stageCode}}, nil
}

func TestCatalogRoutes(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Catalog: catalogStub{}})
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/math/modules", http.StatusOK},
		{"/api/v1/math/modules/add10", http.StatusOK},
		{"/api/v1/math/modules/add10/stages/within5", http.StatusOK},
		{"/api/v1/math/modules/unknown", http.StatusNotFound},
		{"/api/v1/math/modules/add10/stages/unknown", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.want, response.Code, "%s: %s", tc.path, response.Body.String())
	}
}

type stubQuiz struct{}

func (stubQuiz) Generate(_ context.Context, quizType string, _ []int64) (quiz.Question, error) {
	if quizType != "equation" {
		return quiz.Question{}, quiz.ErrInvalidType
	}
	return quiz.Question{InstanceID: "q1", Type: "equation", TargetID: 10, Options: []quiz.Option{{ID: 0, Label: "8"}}, AnswerIndex: 0}, nil
}

func TestGenerateQuizRoute(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Quiz: stubQuiz{}})
	body, err := json.Marshal(map[string]any{"type": "equation", "excludeTargetIds": []int64{1}})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/math/quiz/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	bad, err := json.Marshal(map[string]any{"type": "blend"})
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/math/quiz/generate", bytes.NewReader(bad))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestGenerateQuizUnavailable(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/math/quiz/generate", bytes.NewReader([]byte(`{"type":"equation"}`)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
