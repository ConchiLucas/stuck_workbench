# Pinyin Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go API for pinyin-app that reads content-admin-maintained pinyin content directly from PostgreSQL and MinIO and atomically writes plans, attempts, mastery, statistics, and rewards.

**Architecture:** `pinyin-server` is the only API used by pinyin-app. It never calls ports 19091 or 19081; it filters all catalog and question access to subject `pinyin`, serves existing MinIO objects through its own URLs, and reuses `shared-go` for learning writes.

**Tech Stack:** Go 1.26, Gin, GORM, PostgreSQL, MinIO Go SDK, Testify, Docker

**Prerequisites:** Complete `2026-09-01-shared-learning-engine-extraction.md` and `2026-09-01-shared-plan-option-order.md`. The latter is the only owner of the shared `plan_items.option_order` migration.

---

### Task 1: Scaffold Configuration, Database, and Health API

**Files:**
- Create: `kid-workbench/pinyin-server/go.mod`
- Create: `kid-workbench/pinyin-server/internal/config/config.go`
- Create: `kid-workbench/pinyin-server/internal/db/db.go`
- Create: `kid-workbench/pinyin-server/internal/http/router.go`
- Create: `kid-workbench/pinyin-server/internal/http/response.go`
- Create: `kid-workbench/pinyin-server/internal/http/router_test.go`
- Create: `kid-workbench/pinyin-server/cmd/server/main.go`

- [ ] **Step 1: Write the health-route test**

```go
func TestHealthz(t *testing.T) {
    r := httpapi.NewRouter(httpapi.Deps{})
    w := httptest.NewRecorder()
    r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
    require.Equal(t, http.StatusOK, w.Code)
    require.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}
```

- [ ] **Step 2: Run it to verify failure**

Run: `cd kid-workbench/pinyin-server && go test ./internal/http -run TestHealthz -count=1`

Expected: FAIL because the module and router do not exist.

- [ ] **Step 3: Create the module and configuration**

Use module `github.com/conchi/pinyin-server`, Go `1.26.1`, and a local replace for `github.com/conchi/study-learning => ../shared-go`. Load `APP_ADDR` with default `:19111` and the same `APP_DB_*` variables used by parent-dashboard.

- [ ] **Step 4: Implement the minimal router and server wiring**

```go
func NewRouter(deps Deps) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
    return r
}
```

Open PostgreSQL with GORM but do not run content-admin migrations from this service.

Use one response envelope for `/api/v1` routes:

```go
type APIError struct {
    Code string `json:"code"`
    Message string `json:"message"`
}

func writeData(c *gin.Context, status int, data any) {
    c.JSON(status, gin.H{"data": data, "error": nil})
}

func writeError(c *gin.Context, status int, code, message string) {
    c.JSON(status, gin.H{"data": nil, "error": APIError{Code: code, Message: message}})
}
```

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/pinyin-server && go mod tidy && go test ./... -count=1`

```bash
git add kid-workbench/pinyin-server
git commit -m "feat: scaffold pinyin server"
```

### Task 2: Add Subject-Scoped Catalog Queries

**Files:**
- Create: `kid-workbench/pinyin-server/internal/catalog/model.go`
- Create: `kid-workbench/pinyin-server/internal/catalog/repository.go`
- Create: `kid-workbench/pinyin-server/internal/catalog/repository_test.go`
- Create: `kid-workbench/pinyin-server/internal/http/handler_catalog.go`
- Modify: `kid-workbench/pinyin-server/internal/http/router.go`

- [ ] **Step 1: Write a cross-subject isolation test**

Seed one pinyin and one literacy knowledge point plus matching asset rows. Assert `ListModules` and `ListItems` return only the pinyin records.

- [ ] **Step 2: Run it to verify failure**

Run: `cd kid-workbench/pinyin-server && go test ./internal/catalog -run TestRepositoryOnlyReturnsPinyin -count=1`

Expected: FAIL because `catalog.Repository` is undefined.

- [ ] **Step 3: Implement focused DTOs and queries**

```go
type Item struct {
    KpID int64 `json:"kpId"`
    Letter string `json:"letter"`
    ModuleCode string `json:"moduleCode"`
    ModuleName string `json:"moduleName"`
    SoloText string `json:"soloText"`
    WordText string `json:"wordText"`
    HasSoloSpeech bool `json:"hasSoloSpeech"`
    HasWordSpeech bool `json:"hasWordSpeech"`
    HasGlyph bool `json:"hasGlyph"`
}
```

Every SQL query must join `knowledge_points -> modules -> subjects` and include `WHERE subjects.code = 'pinyin'`.
Item queries additionally left join `pinyin_assets ON pinyin_assets.kp_id = knowledge_points.id`; `pinyin_assets` remains read-only in this service.

- [ ] **Step 4: Add routes**

```http
GET /api/v1/pinyin/modules
GET /api/v1/pinyin/modules/:moduleCode/items
GET /api/v1/pinyin/items/:kpId
```

Return `{ "data": ..., "error": null }` on success, matching the existing kid client envelope.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/pinyin-server && go test ./internal/catalog ./internal/http -count=1`

```bash
git add kid-workbench/pinyin-server/internal/catalog kid-workbench/pinyin-server/internal/http
git commit -m "feat: expose pinyin catalog"
```

### Task 3: Serve Existing MinIO Assets

**Files:**
- Create: `kid-workbench/pinyin-server/internal/asset/store.go`
- Create: `kid-workbench/pinyin-server/internal/asset/keys.go`
- Create: `kid-workbench/pinyin-server/internal/asset/service.go`
- Create: `kid-workbench/pinyin-server/internal/asset/service_test.go`
- Create: `kid-workbench/pinyin-server/internal/http/handler_asset.go`
- Modify: `kid-workbench/pinyin-server/internal/config/config.go`
- Modify: `kid-workbench/pinyin-server/internal/http/router.go`

- [ ] **Step 1: Write key and missing-object tests**

```go
func TestKeys(t *testing.T) {
    require.Equal(t, "pinyin/glyphs/42.png", asset.GlyphKey(42))
    require.Equal(t, "pinyin/speech/42/solo.mp3", asset.SpeechKey(42, "solo"))
    require.Equal(t, "pinyin/speech/42/word.mp3", asset.SpeechKey(42, "word"))
}
```

Also assert an absent object maps to `asset.ErrMissing`, not an internal error.

- [ ] **Step 2: Run tests to verify failure**

Run: `cd kid-workbench/pinyin-server && go test ./internal/asset -count=1`

Expected: FAIL because the asset package does not exist.

- [ ] **Step 3: Implement read-only object access**

Configure endpoint, access key, secret key, bucket, base path, and TLS from `APP_MINIO_*`. Do not add `PutObject` or TTS methods to this package.

- [ ] **Step 4: Add routes and MIME/cache headers**

```http
GET /api/v1/pinyin/items/:kpId/glyph.png
GET /api/v1/pinyin/items/:kpId/speech/:kind.mp3
```

Validate `kind` as `solo|word`. Return `404` with code `asset_missing`; return `Content-Type: image/png` or `audio/mpeg` and `Cache-Control: public, max-age=86400` on success.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/pinyin-server && go test ./internal/asset ./internal/http -count=1`

```bash
git add kid-workbench/pinyin-server/internal/asset kid-workbench/pinyin-server/internal/http kid-workbench/pinyin-server/internal/config
git commit -m "feat: serve pinyin learning assets"
```

### Task 4: Add Child Progress Queries

**Files:**
- Create: `kid-workbench/pinyin-server/internal/progress/service.go`
- Create: `kid-workbench/pinyin-server/internal/progress/service_test.go`
- Create: `kid-workbench/pinyin-server/internal/http/handler_progress.go`
- Modify: `kid-workbench/pinyin-server/internal/http/router.go`

- [ ] **Step 1: Write progress aggregation tests**

Seed pinyin skill rows for `inword` and `listen` and unrelated literacy rows. Assert the response contains only pinyin items, both ordered skills, and the rolled-up status.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-server && go test ./internal/progress -count=1`

Expected: FAIL because `progress.Service` is undefined.

- [ ] **Step 3: Implement the response model**

```go
type ItemProgress struct {
    KpID int64 `json:"kpId"`
    Letter string `json:"letter"`
    Status string `json:"status"`
    Skills []SkillProgress `json:"skills"`
}
```

Missing skill rows must appear as `not_started`; validate the child exists before querying.

- [ ] **Step 4: Add endpoint**

```http
GET /api/v1/children/:childId/pinyin/progress
```

Return `404 child_not_found` for an unknown child.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/pinyin-server && go test ./internal/progress ./internal/http -count=1`

```bash
git add kid-workbench/pinyin-server/internal/progress kid-workbench/pinyin-server/internal/http
git commit -m "feat: expose pinyin progress"
```

### Task 5: Create Stable Pinyin Plans

**Files:**
- Create: `kid-workbench/pinyin-server/internal/plan/model.go`
- Create: `kid-workbench/pinyin-server/internal/plan/service.go`
- Create: `kid-workbench/pinyin-server/internal/plan/service_test.go`
- Create: `kid-workbench/pinyin-server/internal/http/handler_plan.go`
- Modify: `kid-workbench/pinyin-server/internal/http/router.go`

- [ ] **Step 1: Write plan stability tests**

Assert a created plan has `subject_code = 'pinyin'`, contains only `inword|listen` questions, excludes `listen` for `eng`, and returns identical question IDs and option ordering on repeated reads.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-server && go test ./internal/plan -run TestCreatePlanIsStable -count=1`

Expected: FAIL because plan service is undefined.

- [ ] **Step 3: Verify the shared option-order prerequisite**

Run: `cd kid-workbench/shared-go && go test ./model -run TestPlanItemHasPersistedOptionOrder -count=1`

Expected: PASS. Do not create another migration in `pinyin-server`.

- [ ] **Step 4: Implement plan creation and safe question DTOs**

Read eligible `inword|listen` rows from `questions` joined to pinyin knowledge points. Store selected `question_id` and the shuffled original option indexes as a comma-separated value such as `"2,0,3,1"` in `plan_items.option_order`. Return stem, visual, speech URL and reordered options, but omit the stored answer.

When loading an older plan whose `option_order` is empty, use identity order `"0,1,2,3"` and persist it before returning the plan. Never reshuffle a non-empty order.

- [ ] **Step 5: Add plan routes**

```http
POST /api/v1/children/:childId/pinyin/plans
GET  /api/v1/children/:childId/pinyin/plans/:planId
POST /api/v1/children/:childId/pinyin/plans/:planId/start
```

Reject access when `plan.child_id` or `plan.subject_code` does not match the route.

- [ ] **Step 6: Test migrations and service behavior**

Run: `cd kid-workbench/pinyin-server && go test ./internal/plan ./internal/http -count=1`

Run a PostgreSQL integration test that creates, reloads, and compares a plan.

- [ ] **Step 7: Commit**

```bash
git add kid-workbench/pinyin-server
git commit -m "feat: create stable pinyin plans"
```

### Task 6: Implement Server-Side Answering

**Files:**
- Create: `kid-workbench/pinyin-server/internal/practice/service.go`
- Create: `kid-workbench/pinyin-server/internal/practice/service_test.go`
- Modify: `kid-workbench/pinyin-server/internal/http/handler_plan.go`

- [ ] **Step 1: Write two-try and idempotency tests**

Submit a wrong answer, retry correctly, then repeat the second request with the same `clientId`. Assert two attempts, one completed plan item, one correct item, no duplicated daily stat, and no duplicated reward.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-server && go test ./internal/practice -count=1`

Expected: FAIL because `practice.Service.Answer` is undefined.

- [ ] **Step 3: Implement the request and response contract**

```go
type AnswerInput struct {
    ClientID string `json:"clientId" binding:"required"`
    OptionIndex int `json:"optionIndex" binding:"min=0,max=3"`
    CostMs int `json:"costMs" binding:"min=0,max=3600000"`
}

type AnswerResult struct {
    Correct bool `json:"correct"`
    AnswerIndex int `json:"answerIndex"`
    CanRetry bool `json:"canRetry"`
    Tries int `json:"tries"`
    Status string `json:"status"`
    Mastery learning.StateDTO `json:"mastery"`
}
```

Resolve the correct stored option through `option_order`, call shared `ApplyOne` inside the same GORM transaction, and update `plan_items` and `study_plans` before commit.
For a displayed `OptionIndex`, parse `option_order`, map it back to the original question option index, and compare that original index with `questions.answer`. Pass the request `clientId` unchanged into `attempts.client_id`.

- [ ] **Step 4: Add answer and finish endpoints**

```http
POST /api/v1/children/:childId/pinyin/plans/:planId/items/:itemId/answer
POST /api/v1/children/:childId/pinyin/plans/:planId/finish
```

Finishing an incomplete plan returns `409 plan_incomplete`; finishing an already completed plan replays the existing result.

- [ ] **Step 5: Run transaction tests**

Run: `cd kid-workbench/pinyin-server && go test ./internal/practice ./internal/http -count=1`

Expected: PASS including rollback, retry and replay cases.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/pinyin-server/internal/practice kid-workbench/pinyin-server/internal/http
git commit -m "feat: record pinyin practice results"
```

### Task 7: Add Home Summary, Docker, and Workspace Wiring

**Files:**
- Create: `kid-workbench/pinyin-server/internal/home/service.go`
- Create: `kid-workbench/pinyin-server/internal/home/service_test.go`
- Create: `kid-workbench/pinyin-server/internal/http/handler_home.go`
- Create: `kid-workbench/pinyin-server/Dockerfile`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Write home summary test**

Seed an active pinyin plan, one due pinyin item and unrelated literacy data. Assert home reports the pinyin plan and exactly one due item.

- [ ] **Step 2: Implement home endpoint**

```http
GET /api/v1/children/:childId/pinyin/home
```

Return child summary, current plan, due count, and per-module mastered/total counts.

- [ ] **Step 3: Add Docker build**

Use a Go build stage with `shared-go` available at the relative replace path and a minimal Alpine runtime. Expose `19111` and add `GET /healthz` healthcheck.

- [ ] **Step 4: Add Compose service**

Add `pinyin-server` on port `19111`, reuse `*db_env`, pass `APP_MINIO_*`, depend on healthy PostgreSQL, and join `vibedeploy-shared`. Do not add dependencies on `backend` or `content-admin`.

- [ ] **Step 5: Verify all backend behavior**

Run: `cd kid-workbench/pinyin-server && go test ./... -count=1`

Run: `cd kid-workbench && docker compose config --quiet`

Run: `cd kid-workbench && docker compose build pinyin-server`

Expected: all succeed.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/pinyin-server kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "build: add pinyin server to workspace"
```

### Task 8: Cross-System Database Acceptance

**Files:**
- Create: `kid-workbench/pinyin-server/scripts/acceptance.sh`
- Modify: `kid-workbench/pinyin-server/README.md`

- [ ] **Step 1: Write the bounded acceptance script**

The script must: verify pinyin assets exist; create a test pinyin plan for child `1`; submit one answer with a unique client ID; query `attempts`, `mastery_skills`, `mastery_states`, and `daily_stats`; and print row counts without deleting existing data.

- [ ] **Step 2: Run acceptance against the local stack**

Run: `cd kid-workbench && docker compose up --build -d postgres seed content-admin pinyin-server`

Run: `cd kid-workbench && ./pinyin-server/scripts/acceptance.sh`

Expected: one new attempt and matching pinyin mastery rows; no HTTP requests to ports 19091 or 19081 from pinyin-server logs.

- [ ] **Step 3: Verify parent dashboard reads the result**

Run the existing parent dashboard service test/query for child `1`, subject `pinyin`, and assert the submitted knowledge point no longer appears as `not_started`.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/pinyin-server/scripts kid-workbench/pinyin-server/README.md
git commit -m "test: add pinyin server acceptance flow"
```
