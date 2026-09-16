# English Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go API for `english-app` that reads English course content and existing MinIO assets, creates stable English practice plans, and atomically records attempts, mastery, statistics, and rewards.

**Architecture:** `english-server` is the only API used by `english-app`. It scopes every query to subject `english`, serves existing assets through its own URLs, uses local models for plans, and delegates learning-state writes to `shared-go` inside the same database transaction.

**Tech Stack:** Go 1.26, Gin, GORM, PostgreSQL, SQLite tests, MinIO Go SDK, Testify, Docker

**Prerequisites:** Complete `2026-09-01-english-content-contract.md`; complete and commit `shared-go`, `006_plan_subject`, and `007_plan_option_order`.

---

### Task 1: Scaffold Configuration, Database, and Response Contract

**Files:**
- Create: `kid-workbench/english-server/go.mod`
- Create: `kid-workbench/english-server/internal/config/config.go`
- Create: `kid-workbench/english-server/internal/db/db.go`
- Create: `kid-workbench/english-server/internal/http/response.go`
- Create: `kid-workbench/english-server/internal/http/router.go`
- Create: `kid-workbench/english-server/internal/http/router_test.go`
- Create: `kid-workbench/english-server/cmd/server/main.go`

- [ ] **Step 1: Write failing health tests**

```go
func TestHealthz(t *testing.T) {
    router := httpapi.NewRouter(httpapi.Deps{})
    w := httptest.NewRecorder()
    router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
    require.Equal(t, http.StatusOK, w.Code)
    require.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}
```

Also test `/readyz` returns `503 dependency_unavailable` when no readiness checker is supplied.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/http -run 'TestHealthz|TestReadyz' -count=1`

Expected: FAIL because the module and router do not exist.

- [ ] **Step 3: Create module and configuration**

Use module `github.com/conchi/english-server`, Go `1.26.1`, and:

```go
replace github.com/conchi/study-learning => ../shared-go
```

Load `APP_ADDR` with default `:19121`, the existing `APP_DB_*` variables, `APP_MINIO_ENDPOINT`, `APP_MINIO_ACCESS_KEY`, `APP_MINIO_SECRET_KEY`, `APP_MINIO_BUCKET`, `APP_MINIO_BASE_PATH`, and `APP_MINIO_USE_SSL`.

- [ ] **Step 4: Implement response and health contracts**

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

`/healthz` checks only the process. `/readyz` calls bounded PostgreSQL and MinIO probes.

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/english-server && go mod tidy && go test ./... -count=1`

```bash
git add kid-workbench/english-server
git commit -m "feat: scaffold english server"
```

### Task 2: Add Subject-Scoped Catalog Queries

**Files:**
- Create: `kid-workbench/english-server/internal/catalog/model.go`
- Create: `kid-workbench/english-server/internal/catalog/repository.go`
- Create: `kid-workbench/english-server/internal/catalog/repository_test.go`
- Create: `kid-workbench/english-server/internal/http/handler_catalog.go`
- Modify: `kid-workbench/english-server/internal/http/router.go`

- [ ] **Step 1: Write cross-subject and payload tests**

Seed one English and one pinyin KP. Assert English queries exclude pinyin and parse `meaningZh`, while malformed optional metadata does not expose raw JSON.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/catalog -run TestRepositoryOnlyReturnsEnglish -count=1`

Expected: FAIL because the catalog package is undefined.

- [ ] **Step 3: Define focused DTOs**

```go
type Word struct {
    KpID int64 `json:"kpId"`
    Word string `json:"word"`
    MeaningZh string `json:"meaningZh"`
    Phonetic string `json:"phonetic,omitempty"`
    PartOfSpeech string `json:"partOfSpeech,omitempty"`
    Example string `json:"example,omitempty"`
    ExampleMeaningZh string `json:"exampleMeaningZh,omitempty"`
    ModuleCode string `json:"moduleCode"`
    ModuleName string `json:"moduleName"`
    HasGlyph bool `json:"hasGlyph"`
    HasSense bool `json:"hasSense"`
    HasSpeech bool `json:"hasSpeech"`
    OrderNo int `json:"orderNo"`
}
```

Every query joins `knowledge_points -> modules -> subjects`, includes `WHERE subjects.code = 'english'`, and left joins read-only `english_assets`.

- [ ] **Step 4: Add routes**

```http
GET /api/v1/english/modules
GET /api/v1/english/modules/:moduleCode/words
GET /api/v1/english/words/:kpId
```

Return `404 content_not_found` for unknown or non-English KPs.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-server && go test ./internal/catalog ./internal/http -count=1`

```bash
git add kid-workbench/english-server/internal/catalog kid-workbench/english-server/internal/http
git commit -m "feat: expose english catalog"
```

### Task 3: Serve Existing English Assets

**Files:**
- Create: `kid-workbench/english-server/internal/asset/keys.go`
- Create: `kid-workbench/english-server/internal/asset/store.go`
- Create: `kid-workbench/english-server/internal/asset/service.go`
- Create: `kid-workbench/english-server/internal/asset/service_test.go`
- Create: `kid-workbench/english-server/internal/http/handler_asset.go`
- Modify: `kid-workbench/english-server/internal/http/router.go`
- Modify: `kid-workbench/english-server/cmd/server/main.go`

- [ ] **Step 1: Write key and error-mapping tests**

```go
func TestKeys(t *testing.T) {
    require.Equal(t, "english/glyphs/42.png", GlyphKey(42))
    require.Equal(t, "english/senses/42.png", SenseKey(42))
    require.Equal(t, "english/speech/42.mp3", SpeechKey(42))
}
```

Assert absent objects map to `ErrMissing`; transport failures map to `ErrUnavailable`.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/asset -count=1`

Expected: FAIL because the asset package does not exist.

- [ ] **Step 3: Implement a read-only MinIO boundary**

Expose only `Get(ctx, key) ([]byte, error)` and `Ping(ctx) error`. Do not add put, generation, TTS, or content-admin HTTP methods.

- [ ] **Step 4: Add routes and headers**

```http
GET /api/v1/english/words/:kpId/glyph.png
GET /api/v1/english/words/:kpId/sense.png
GET /api/v1/english/words/:kpId/speech.mp3
```

Validate the KP belongs to English before reading MinIO. Return `404 asset_missing`, `503 asset_unavailable`, correct MIME type, and `Cache-Control: public, max-age=86400`.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-server && go test ./internal/asset ./internal/http -count=1`

```bash
git add kid-workbench/english-server/internal/asset kid-workbench/english-server/internal/http kid-workbench/english-server/cmd/server/main.go
git commit -m "feat: serve english learning assets"
```

### Task 4: Add English Progress and Home Aggregation

**Files:**
- Create: `kid-workbench/english-server/internal/progress/service.go`
- Create: `kid-workbench/english-server/internal/progress/service_test.go`
- Create: `kid-workbench/english-server/internal/home/service.go`
- Create: `kid-workbench/english-server/internal/home/service_test.go`
- Create: `kid-workbench/english-server/internal/http/handler_progress.go`
- Create: `kid-workbench/english-server/internal/http/handler_home.go`
- Modify: `kid-workbench/english-server/internal/http/router.go`

- [ ] **Step 1: Write failing progress tests**

Seed `listen` and `picture` mastery rows plus unrelated pinyin rows. Assert English progress returns both ordered skills, missing rows as `not_started`, module summaries, and no pinyin data.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/progress ./internal/home -count=1`

Expected: FAIL because services are undefined.

- [ ] **Step 3: Implement response models**

```go
type SkillProgress struct { Code string `json:"code"`; Status string `json:"status"` }
type WordProgress struct {
    KpID int64 `json:"kpId"`
    Word string `json:"word"`
    MeaningZh string `json:"meaningZh"`
    Status string `json:"status"`
    Skills []SkillProgress `json:"skills"`
}
```

Validate the child before querying. Home aggregates child, active/todo English plans, review count, English mastery counts, and flowers in one service call.

- [ ] **Step 4: Add endpoints**

```http
GET /api/v1/children/:childId/english/progress
GET /api/v1/children/:childId/english/home
```

Return `404 child_not_found` for unknown children.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-server && go test ./internal/progress ./internal/home ./internal/http -count=1`

```bash
git add kid-workbench/english-server/internal/progress kid-workbench/english-server/internal/home kid-workbench/english-server/internal/http
git commit -m "feat: expose english home and progress"
```

### Task 5: Create Stable English Plans

**Files:**
- Create: `kid-workbench/english-server/internal/plan/model.go`
- Create: `kid-workbench/english-server/internal/plan/service.go`
- Create: `kid-workbench/english-server/internal/plan/service_test.go`
- Create: `kid-workbench/english-server/internal/http/handler_plan.go`
- Modify: `kid-workbench/english-server/internal/http/router.go`

- [ ] **Step 1: Write plan eligibility and stability tests**

Cover `today`, `review`, and `module`; subject isolation; bucket priority; missing speech exclusion; picture exclusion unless all four sense assets exist; and identical question/option order after reload.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/plan -count=1`

Expected: FAIL because plan service is undefined.

- [ ] **Step 3: Define local plan models and public input**

```go
type StudyPlan struct {
    ID int64 `gorm:"primaryKey"`
    ChildID int64
    PlanDate string
    SeqNo int
    SubjectCode string
    Status string
    TargetCount int
    DoneCount int
    CorrectCount int
    Stars int
    DurationSec int
}

type CreateInput struct {
    Mode string `json:"mode" binding:"required,oneof=today review module"`
    ModuleCode string `json:"moduleCode"`
}
```

Use shared `model.PlanItem`. Store a stable permutation in `plan_items.option_order` and never re-randomize while reading a plan.

- [ ] **Step 4: Implement eligibility and DTO conversion**

Require speech for both English question types. For `picture`, parse option `kpId` values and verify every referenced `english_assets.sense_image_url` is non-empty. Convert asset references to `english-server` URLs. Omit the database `answer` from DTOs.

- [ ] **Step 5: Add plan routes**

```http
POST /api/v1/children/:childId/english/plans
GET  /api/v1/children/:childId/english/plans/:planId
POST /api/v1/children/:childId/english/plans/:planId/start
POST /api/v1/children/:childId/english/plans/:planId/finish
```

Return `409 no_eligible_questions` when no complete English questions qualify.

`finish` returns completed/correct counts, stars, newly earned flowers, and weak words derived from wrong plan items:

```go
type FinishResult struct {
    Plan PlanDTO `json:"plan"`
    Stars int `json:"stars"`
    Flowers int `json:"flowers"`
    WeakWords []WeakWord `json:"weakWords"`
}

type WeakWord struct {
    KpID int64 `json:"kpId"`
    Word string `json:"word"`
    MeaningZh string `json:"meaningZh"`
}
```

Keep all weak words in the API result; the app limits display to six.

- [ ] **Step 6: Test and commit**

Run: `cd kid-workbench/english-server && go test ./internal/plan ./internal/http -count=1`

```bash
git add kid-workbench/english-server/internal/plan kid-workbench/english-server/internal/http
git commit -m "feat: create stable english plans"
```

### Task 6: Add Idempotent Two-Try Answering

**Files:**
- Create: `kid-workbench/english-server/internal/practice/service.go`
- Create: `kid-workbench/english-server/internal/practice/service_test.go`
- Create: `kid-workbench/english-server/internal/http/handler_answer.go`
- Modify: `kid-workbench/english-server/internal/http/router.go`

- [ ] **Step 1: Write transaction and retry tests**

Cover correct first answer, wrong then correct, two wrong answers, invalid option, foreign plan, completed plan, duplicate `clientId`, English skill updates, daily stats, flower reward, and forced rollback after learning-state application.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/practice -count=1`

Expected: FAIL because practice service is undefined.

- [ ] **Step 3: Define request and persisted pick result**

```go
type AnswerInput struct {
    ClientID string `json:"clientId" binding:"required"`
    OptionIndex int `json:"optionIndex" binding:"min=0"`
    CostMs int `json:"costMs" binding:"min=0"`
}

type Pick struct {
    ClientID string `json:"clientId"`
    OptionIndex int `json:"optionIndex"`
    Correct bool `json:"correct"`
}
```

Persist picks in `plan_items.picks`. Before applying a new attempt, lock the item row and return the stored result if `clientId` is already present.

- [ ] **Step 4: Implement one atomic answer transaction**

Inside one `gorm.Transaction`: validate child/English plan/item; decode fixed option order and answer; append the pick; call `learning.Service.ApplyOne`; update tries/status/cost; update plan counts; and preserve the first wrong attempt even if the second is correct.

- [ ] **Step 5: Add answer endpoint**

```http
POST /api/v1/children/:childId/english/plans/:planId/items/:itemId/answer
```

Return the correct option index only when correct or tries are exhausted.

- [ ] **Step 6: Test and commit**

Run: `cd kid-workbench/english-server && go test ./internal/practice ./internal/plan ./internal/http -count=1`

```bash
git add kid-workbench/english-server/internal/practice kid-workbench/english-server/internal/http
git commit -m "feat: record english practice attempts"
```

### Task 7: Add Docker and Workspace Deployment

**Files:**
- Create: `kid-workbench/english-server/Dockerfile`
- Create: `kid-workbench/english-server/README.md`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Add a Docker build using workspace context**

Use `/src/english-server` and `/src/shared-go` so the local replace resolves. Build a static binary and use `alpine` plus `wget` for health checks.

- [ ] **Step 2: Add Compose service**

Expose `19121:19121`, pass the shared database environment and MinIO read credentials, depend on healthy PostgreSQL but not on content-admin or parent backend, and use `/healthz` for process health.

- [ ] **Step 3: Run deployment checks**

Run: `cd kid-workbench && docker compose config --quiet`

Run: `cd kid-workbench && docker compose build english-server`

Expected: both PASS.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/english-server/Dockerfile kid-workbench/english-server/README.md kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "build: add english server to workspace"
```

### Task 8: Final Server Verification

**Files:**
- Create: `kid-workbench/english-server/internal/integration/english_flow_test.go`

- [ ] **Step 1: Add PostgreSQL integration flow**

Against an isolated test database, seed one English module with complete assets and questions, create a plan, reload it, submit wrong then correct with stable `clientId` values, finish it, and assert attempts, English skill mastery, rolled-up mastery, daily stats, and plan counts.

- [ ] **Step 2: Run all Go checks**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Run: `cd kid-workbench/english-server && go test ./... -count=1`

Expected: all PASS.

- [ ] **Step 3: Verify the runtime boundary**

Start PostgreSQL, MinIO and `english-server`, then stop content-admin and parent backend. Fetch an already-generated word, its speech and sense image, and complete a plan.

Expected: no request targets ports `19091` or `19081`.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/english-server/internal/integration
git commit -m "test: verify standalone english server"
```
