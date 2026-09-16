package quiz_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/conchi/pinyin-server/internal/config"
	serverdb "github.com/conchi/pinyin-server/internal/db"
	httpapi "github.com/conchi/pinyin-server/internal/http"
	"github.com/conchi/pinyin-server/internal/quiz"
	"github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincontract"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This test uses a pre-migrated, seeded disposable PostgreSQL database. It never
// creates schemas, edits source material, or binds a listening HTTP socket.
func TestPostgresFormalQuizIntegration(t *testing.T) {
	name := os.Getenv("PINYIN_TEST_DB")
	if name == "" {
		t.Skip("set PINYIN_TEST_DB to a prepared kid_pinyin_verify_* database")
	}
	require.Regexp(t, regexp.MustCompile(`^kid_pinyin_verify_[a-zA-Z0-9_]+$`), name, "refusing a database outside the integration-test namespace")
	cfg := config.Load()
	cfg.DB.Driver = "postgres"
	cfg.DB.DSN = "" // A raw APP_DSN must never override the explicitly gated test database.
	cfg.DB.Name = name
	database, err := serverdb.Open(cfg.DB)
	require.NoError(t, err)
	database = database.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	var actual string
	require.NoError(t, database.Raw("SELECT current_database()").Scan(&actual).Error)
	require.Equal(t, name, actual, "refusing to write to a different database")
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(12)
	sqlDB.SetMaxIdleConns(12)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	database = database.WithContext(ctx)
	child := model.Child{Name: fmt.Sprintf("拼音PG测试-%d", time.Now().UnixNano()), Grade: "测试", CreatedAt: time.Now()}
	other := model.Child{Name: fmt.Sprintf("拼音PG隔离-%d", time.Now().UnixNano()), Grade: "测试", CreatedAt: time.Now()}
	// The fixture seeds explicit IDs without necessarily advancing its sequence.
	// Allocate only these test IDs under a table lock; do not reset any sequence.
	require.NoError(t, database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE children IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		var maxID int64
		if err := tx.Model(&model.Child{}).Select("COALESCE(MAX(id),0)").Scan(&maxID).Error; err != nil {
			return err
		}
		child.ID = maxID + 1
		other.ID = maxID + 2
		if err := tx.Create(&child).Error; err != nil {
			return err
		}
		return tx.Create(&other).Error
	}))
	t.Logf("isolated PostgreSQL test children: %d, %d", child.ID, other.ID)
	// Keep these facts in the disposable test database for follow-up inspection.
	service := quiz.NewService(database, cfg.Mastery)
	router := httpapi.NewRouter(httpapi.Deps{Quiz: service, FormalQuiz: service})
	base := fmt.Sprintf("/api/v1/children/%d/pinyin/quiz", child.ID)
	request := func(method, path string, body any) (int, []byte) {
		encoded, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path, bytes.NewReader(encoded)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response.Code, response.Body.Bytes()
	}
	decode := func(body []byte, out any) {
		t.Helper()
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(body, &envelope))
		require.NotEmpty(t, envelope.Data)
		require.NoError(t, json.Unmarshal(envelope.Data, out))
	}
	generate := func(kind string, excluded []int64) pinyincontract.GeneratedQuestion {
		t.Helper()
		status, body := request(http.MethodPost, base+"/generate", map[string]any{"type": kind, "excludeTargetIds": excluded})
		require.Equal(t, http.StatusOK, status, string(body))
		require.NotContains(t, string(body), "answerIndex")
		require.NotContains(t, string(body), "answerOptionId")
		var q pinyincontract.GeneratedQuestion
		decode(body, &q)
		require.Equal(t, kind, q.Type)
		require.Len(t, q.Options, 4)
		require.NotEmpty(t, q.InstanceID)
		for _, option := range q.Options {
			require.NotEmpty(t, option.ID)
			require.NotEqual(t, fmt.Sprint(q.TargetID), option.ID)
		}
		if kind == "blend" {
			require.Empty(t, q.Visual.Syllable)
		}
		return q
	}
	questions := map[string]pinyincontract.GeneratedQuestion{}
	for _, kind := range []string{"listen", "inword", "shape", "blend"} {
		questions[kind] = generate(kind, nil)
	}
	var count int64
	require.NoError(t, database.Model(&model.Attempt{}).Where("child_id=?", child.ID).Count(&count).Error)
	require.Zero(t, count)

	// Start the same submission on independent pooled PostgreSQL connections.
	q := questions["blend"]
	var instance model.PinyinQuizInstance
	require.NoError(t, database.First(&instance, "id=?", q.InstanceID).Error)
	require.Equal(t, child.ID, instance.ChildID)
	var link model.PinyinSyllableLink
	require.NoError(t, database.First(&link, "asset_id=?", q.TargetID).Error)
	require.Equal(t, link.KpID, q.KpID)
	input := pinyincontract.AnswerRequest{ClientID: "pg-concurrent", OptionID: instance.AnswerOptionID, CostMs: 1800}
	type response struct {
		status int
		body   []byte
	}
	responses := make(chan response, 8)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, body := request(http.MethodPost, base+"/"+q.InstanceID+"/answer", input)
			responses <- response{status, body}
		}()
	}
	close(start)
	wg.Wait()
	close(responses)
	var accepted pinyincontract.AnswerResult
	for response := range responses {
		require.Equal(t, http.StatusOK, response.status, string(response.body))
		var current pinyincontract.AnswerResult
		decode(response.body, &current)
		if accepted.AttemptID == 0 {
			accepted = current
		} else {
			require.Equal(t, accepted, current)
		}
	}
	require.True(t, accepted.Correct)
	require.NoError(t, database.Model(&model.Attempt{}).Where("child_id=?", child.ID).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, database.Model(&model.PinyinAnswerReceipt{}).Where("child_id=?", child.ID).Count(&count).Error)
	require.EqualValues(t, 1, count)
	input.ClientID = "pg-conflict"
	status, body := request(http.MethodPost, base+"/"+q.InstanceID+"/answer", input)
	require.Equal(t, http.StatusConflict, status, string(body))
	status, body = request(http.MethodGet, fmt.Sprintf("/api/v1/children/%d/pinyin/quiz/%s", other.ID, q.InstanceID), nil)
	require.Equal(t, http.StatusNotFound, status, string(body))
	status, body = request(http.MethodGet, base+"/"+q.InstanceID, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var restored pinyincontract.InstanceSnapshot
	decode(body, &restored)
	require.Equal(t, &accepted, restored.AcceptedResult)
	require.Equal(t, q, restored.GeneratedQuestion)
	var receipt model.PinyinAnswerReceipt
	require.NoError(t, database.Where("child_id=? AND client_id=?", child.ID, "pg-concurrent").First(&receipt).Error)
	require.Equal(t, accepted.AttemptID, receipt.AttemptID)
	require.Equal(t, "blend", receipt.SkillCode)
	require.Equal(t, instance.AnswerOptionID, receipt.SelectedOptionID)
	var fact model.Attempt
	require.NoError(t, database.First(&fact, receipt.AttemptID).Error)
	require.Equal(t, q.KpID, fact.KpID)
	require.True(t, fact.IsCorrect)
	require.Nil(t, fact.QuestionID)
	require.NoError(t, database.Model(&model.PinyinQuizInstance{}).Where("id=?", q.InstanceID).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	status, body = request(http.MethodGet, base+"/"+q.InstanceID, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	decode(body, &restored)
	require.Equal(t, &accepted, restored.AcceptedResult)

	// Complete one real letter and one real syllable using the configured engine
	// threshold; each additional answer gets a fresh server-issued instance.
	var letterTargets, syllableTargets []int64
	require.NoError(t, database.Table("pinyin_assets").Pluck("kp_id", &letterTargets).Error)
	require.NoError(t, database.Table("pinyin_syllable_assets").Where("enabled=true").Pluck("id", &syllableTargets).Error)
	excluding := func(ids []int64, target int64) []int64 {
		out := []int64{}
		for _, id := range ids {
			if id != target {
				out = append(out, id)
			}
		}
		return out
	}
	needed := func(kpID int64) int {
		var kp model.KnowledgePoint
		require.NoError(t, database.First(&kp, kpID).Error)
		n := cfg.Mastery.BaseMasterStreak + max(1, kp.Difficulty)
		require.Greater(t, n, 0)
		require.LessOrEqual(t, n, 100)
		return n
	}
	acceptedCount := 1
	complete := func(kind string, target int64, excluded []int64, n int) pinyincontract.AnswerResult {
		t.Helper()
		var last pinyincontract.AnswerResult
		for i := 0; i < n; i++ {
			next := generate(kind, excluded)
			require.Equal(t, target, next.TargetID)
			var secret model.PinyinQuizInstance
			require.NoError(t, database.First(&secret, "id=?", next.InstanceID).Error)
			status, body := request(http.MethodPost, base+"/"+next.InstanceID+"/answer", pinyincontract.AnswerRequest{ClientID: fmt.Sprintf("pg-%s-%d", kind, i), OptionID: secret.AnswerOptionID, CostMs: 1200})
			require.Equal(t, http.StatusOK, status, string(body))
			decode(body, &last)
			require.True(t, last.Correct)
			acceptedCount++
		}
		return last
	}
	letter := questions["listen"]
	letterNeed := needed(letter.KpID)
	for _, kind := range []string{"listen", "inword", "shape"} {
		last := complete(kind, letter.TargetID, excluding(letterTargets, letter.TargetID), letterNeed)
		require.Equal(t, kind == "shape", last.Knowledge.NewlyMastered)
	}
	blendNeed := needed(q.KpID)
	if blendNeed > 1 {
		last := complete("blend", q.TargetID, excluding(syllableTargets, q.TargetID), blendNeed-1)
		require.True(t, last.Knowledge.NewlyMastered)
	} else {
		require.True(t, accepted.Knowledge.NewlyMastered)
	}
	var milestones []model.PinyinMasteryMilestone
	require.NoError(t, database.Where("child_id=? AND rule_version=2", child.ID).Find(&milestones).Error)
	require.Len(t, milestones, 2)
	for _, milestone := range milestones {
		require.Contains(t, []int64{letter.KpID, q.KpID}, milestone.KpID)
		var state model.MasteryState
		require.NoError(t, database.Where("child_id=? AND kp_id=?", child.ID, milestone.KpID).First(&state).Error)
		require.NotNil(t, state.MasteredAt)
		require.True(t, milestone.FirstCompletedAt.Equal(*state.MasteredAt))
	}
	var totals struct{ Attempts, Correct, Newly int }
	require.NoError(t, database.Table("daily_stats").Select("SUM(attempts) AS attempts,SUM(correct) AS correct,SUM(newly_mastered) AS newly").Where("child_id=?", child.ID).Scan(&totals).Error)
	require.Equal(t, acceptedCount, totals.Attempts)
	require.Equal(t, acceptedCount, totals.Correct)
	require.Equal(t, 2, totals.Newly)
	require.NoError(t, database.Model(&model.Attempt{}).Where("child_id=?", child.ID).Count(&count).Error)
	require.EqualValues(t, acceptedCount, count)
	require.NoError(t, database.Model(&model.PinyinAnswerReceipt{}).Where("child_id=?", child.ID).Count(&count).Error)
	require.EqualValues(t, acceptedCount, count)
	require.NoError(t, database.Model(&model.FlowerLedger{}).Where("child_id=? AND reason='mastered'", child.ID).Count(&count).Error)
	require.EqualValues(t, 2, count)
	// The physical FK must reject an orphan even outside the service validation.
	bad := model.PinyinQuizInstance{ID: fmt.Sprintf("invalid-fk-%d", time.Now().UnixNano()), ChildID: child.ID, KpID: -1, SkillCode: "listen", SnapshotVersion: 1, PublicSnapshot: "{}", AnswerOptionID: "x", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	err = database.Transaction(func(tx *gorm.DB) error { return tx.Create(&bad).Error })
	require.ErrorIs(t, err, gorm.ErrForeignKeyViolated)
	require.NoError(t, database.Model(&model.PinyinQuizInstance{}).Where("id=?", bad.ID).Count(&count).Error)
	require.Zero(t, count)
}
