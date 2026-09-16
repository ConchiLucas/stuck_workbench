package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/math-server/internal/asset"
	httpapi "github.com/conchi/math-server/internal/http"
)

type audioStub struct{ err error }

func (s audioStub) QuestionAudio(_ context.Context, key string) ([]byte, error) {
	if key != "math/questions/42.mp3" {
		return nil, fmt.Errorf("unexpected key %s", key)
	}
	if s.err != nil {
		return nil, s.err
	}
	return []byte("mp3"), nil
}

func audioDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
CREATE TABLE children(id INTEGER PRIMARY KEY);
CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
CREATE TABLE modules(id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT);
CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY, module_id INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, media_url TEXT);
CREATE TABLE study_plans(id INTEGER PRIMARY KEY, child_id INTEGER, subject_code TEXT, plan_kind TEXT);
CREATE TABLE plan_items(id INTEGER PRIMARY KEY, plan_id INTEGER, kp_id INTEGER, question_snapshot TEXT);
INSERT INTO children VALUES(1),(2);
INSERT INTO subjects VALUES(1,'math'),(2,'pinyin');
INSERT INTO modules VALUES(1,1,'add10'),(2,2,'initials');
INSERT INTO knowledge_points VALUES(10,1),(20,2);
INSERT INTO questions VALUES(42,10,'calc','math/questions/42.mp3'),(43,20,'calc','math/questions/43.mp3');
INSERT INTO study_plans VALUES(7,1,'math','daily'),(8,1,'pinyin','daily');
INSERT INTO plan_items VALUES(9,7,10,'{"audioObjectKey":"math/questions/42.mp3"}'),(10,8,20,'{"audioObjectKey":"math/questions/43.mp3"}');
`).Error)
	return db
}

func TestPlanItemAudioIsChildAndPlanScoped(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Database: audioDB(t), Assets: audioStub{}})
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/children/1/math/plans/7/items/9/audio.mp3", http.StatusOK},
		{"/api/v1/children/2/math/plans/7/items/9/audio.mp3", http.StatusNotFound},
		{"/api/v1/children/1/math/plans/8/items/10/audio.mp3", http.StatusNotFound},
		{"/api/v1/children/1/math/plans/7/items/99/audio.mp3", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.want, response.Code, "%s: %s", tc.path, response.Body.String())
		if tc.want == http.StatusOK {
			require.Equal(t, "audio/mpeg", response.Header().Get("Content-Type"))
			require.Equal(t, "private, max-age=86400", response.Header().Get("Cache-Control"))
		}
	}
}

func TestLearningAudioIsKpAndCodeScoped(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Database: audioDB(t), Assets: audioStub{}})
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/children/1/math/items/10/audio/calc.mp3", http.StatusOK},
		{"/api/v1/children/1/math/items/20/audio/calc.mp3", http.StatusNotFound},
		{"/api/v1/children/1/math/items/10/audio/unknown.mp3", http.StatusNotFound},
		{"/api/v1/children/99/math/items/10/audio/calc.mp3", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.want, response.Code, "%s: %s", tc.path, response.Body.String())
	}
}

func TestAudioStorageFailureIsRetryable(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Database: audioDB(t), Assets: audioStub{err: asset.ErrUnavailable}})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/children/1/math/plans/7/items/9/audio.mp3", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "audio_unavailable")
}

func TestAudioDatabaseFailureIsRetryable(t *testing.T) {
	database := audioDB(t)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	router := httpapi.NewRouter(httpapi.Deps{Database: database, Assets: audioStub{}})
	for _, path := range []string{
		"/api/v1/children/1/math/items/10/audio/calc.mp3",
		"/api/v1/children/1/math/plans/7/items/9/audio.mp3",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusServiceUnavailable, response.Code)
		require.Contains(t, response.Body.String(), "database_unavailable")
	}
}
