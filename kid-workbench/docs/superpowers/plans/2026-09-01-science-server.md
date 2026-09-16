# Science Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an independently deployable Go API that serves only published science content and records science learning without calling content-admin or parent-dashboard over HTTP.

**Architecture:** `science-server` reads subject-scoped catalog, question, progress and MinIO data directly, and delegates atomic learning updates to `shared-go`. It owns science plan composition and exposes a child-safe API with no answers before submission.

**Tech Stack:** Go 1.26, Gin, GORM, PostgreSQL, SQLite unit tests, MinIO Go SDK, shared `github.com/conchi/study-learning`

**Prerequisites:** Complete `2026-09-01-shared-learning-engine-extraction.md`, `2026-09-01-shared-plan-option-order.md`, and `2026-09-01-science-content-publishing.md` first. Do not duplicate transaction or shared schema code while those plans are in progress.

---

### Task 1: Scaffold the Independent Server

**Files:**
- Create: `kid-workbench/science-server/go.mod`
- Create: `kid-workbench/science-server/internal/config/config.go`
- Create: `kid-workbench/science-server/internal/db/db.go`
- Create: `kid-workbench/science-server/internal/http/router.go`
- Create: `kid-workbench/science-server/internal/http/router_test.go`
- Create: `kid-workbench/science-server/cmd/server/main.go`

- [ ] **Step 1: Write the failing health test**

```go
func TestHealthz(t *testing.T) {
    r := httpapi.NewRouter(httpapi.Deps{})
    w := httptest.NewRecorder()
    r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
    require.Equal(t, http.StatusOK, w.Code)
    require.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-server && go test ./internal/http -count=1`

Expected: FAIL because the module and router are absent.

- [ ] **Step 3: Create module and configuration**

Use module `github.com/conchi/study-science`, require Gin, GORM PostgreSQL/SQLite, MinIO and `github.com/conchi/study-learning v0.0.0`, and add:

```go
replace github.com/conchi/study-learning => ../shared-go
```

Configuration must support existing `APP_DB_*`, `APP_DSN`, `APP_ADDR` with default `:19121`, and `APP_MINIO_ENDPOINT`, `APP_MINIO_ACCESS_KEY`, `APP_MINIO_SECRET_KEY`, `APP_MINIO_BUCKET`, `APP_MINIO_USE_SSL`.

- [ ] **Step 4: Implement minimal health wiring**

`/healthz` reports process liveness only. Add `/readyz` that pings PostgreSQL and returns `503 database_unavailable` when unavailable; MinIO readiness is reported separately in the response and does not prevent text-only endpoints.

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/science-server && go test ./... -count=1`

```bash
git add kid-workbench/science-server
git commit -m "feat: scaffold independent science server"
```

### Task 2: Add Published, Subject-Scoped Catalog Queries

**Files:**
- Create: `kid-workbench/science-server/internal/catalog/model.go`
- Create: `kid-workbench/science-server/internal/catalog/service.go`
- Create: `kid-workbench/science-server/internal/catalog/service_test.go`
- Create: `kid-workbench/science-server/internal/http/handler_catalog.go`
- Modify: `kid-workbench/science-server/internal/http/router.go`

- [ ] **Step 1: Write isolation and publication tests**

Seed one published science KP, one draft science KP and one published literacy KP. Assert module and item endpoints return only the published science KP.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-server && go test ./internal/catalog -count=1`

Expected: FAIL because catalog service is absent.

- [ ] **Step 3: Define child-safe DTOs**

```go
type ModuleDTO struct {
    Code string `json:"code"`
    Name string `json:"name"`
    Order int `json:"order"`
    Total int `json:"total"`
}

type ItemDTO struct {
    KpID int64 `json:"kpId"`
    Title string `json:"title"`
    ModuleCode string `json:"moduleCode"`
    Difficulty int `json:"difficulty"`
    Summary string `json:"summary"`
    Explanation string `json:"explanation"`
    FunFact string `json:"funFact"`
    ContentVersion int `json:"contentVersion"`
    SenseImageURL string `json:"senseImageUrl,omitempty"`
    GlyphImageURL string `json:"glyphImageUrl,omitempty"`
    SpeechURL string `json:"speechUrl,omitempty"`
    HasPractice bool `json:"hasPractice"`
}
```

Every query must join `subjects`, `modules`, `knowledge_points`, and `science_assets`, with `subjects.code = 'science'` and `science_assets.review_status = 'published'` in SQL rather than filtering after loading.

- [ ] **Step 4: Add routes**

```http
GET /api/v1/science/modules
GET /api/v1/science/modules/:moduleCode/items
GET /api/v1/science/items/:kpId
```

Return service-local asset paths with `?v=<contentVersion>`; never return stored `localhost:19091` URLs.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-server && go test ./internal/catalog ./internal/http -count=1`

```bash
git add kid-workbench/science-server/internal/catalog kid-workbench/science-server/internal/http
git commit -m "feat: expose published science catalog"
```

### Task 3: Serve Existing Science Assets Read-Only

**Files:**
- Create: `kid-workbench/science-server/internal/asset/store.go`
- Create: `kid-workbench/science-server/internal/asset/store_test.go`
- Create: `kid-workbench/science-server/internal/http/handler_asset.go`
- Modify: `kid-workbench/science-server/internal/http/router.go`

- [ ] **Step 1: Write key, scope and missing-object tests**

Verify exact keys:

```go
require.Equal(t, "science/senses/42.png", asset.SenseKey(42))
require.Equal(t, "science/glyphs/42.png", asset.GlyphKey(42))
require.Equal(t, "science/speech/42.mp3", asset.SpeechKey(42))
```

Assert unpublished or non-science KPs return `404 item_not_found`, missing objects return `404 asset_missing`, and store failures return `503 asset_unavailable`.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-server && go test ./internal/asset ./internal/http -run Asset -count=1`

Expected: FAIL because asset service is absent.

- [ ] **Step 3: Implement the read-only store**

Define an interface usable by tests:

```go
type Reader interface {
    Get(ctx context.Context, key string) ([]byte, error)
}

func SenseKey(kpID int64) string  { return fmt.Sprintf("science/senses/%d.png", kpID) }
func GlyphKey(kpID int64) string  { return fmt.Sprintf("science/glyphs/%d.png", kpID) }
func SpeechKey(kpID int64) string { return fmt.Sprintf("science/speech/%d.mp3", kpID) }
```

The production implementation uses MinIO `GetObject` and never exposes write, generate or delete methods.

- [ ] **Step 4: Add routes and cache headers**

```http
GET /api/v1/science/items/:kpId/sense.png
GET /api/v1/science/items/:kpId/glyph.png
GET /api/v1/science/items/:kpId/speech.mp3
```

Set correct MIME types, `Cache-Control: public, max-age=86400`, and `ETag` derived from KP, asset kind and content version.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-server && go test ./internal/asset ./internal/http -count=1`

```bash
git add kid-workbench/science-server/internal/asset kid-workbench/science-server/internal/http
git commit -m "feat: serve science assets read only"
```

### Task 4: Add Science Progress and Home Summary

**Files:**
- Create: `kid-workbench/science-server/internal/progress/model.go`
- Create: `kid-workbench/science-server/internal/progress/service.go`
- Create: `kid-workbench/science-server/internal/progress/service_test.go`
- Create: `kid-workbench/science-server/internal/http/handler_progress.go`
- Modify: `kid-workbench/science-server/internal/http/router.go`

- [ ] **Step 1: Write progress aggregation tests**

Seed published/draft science KPs plus literacy state. Assert progress includes only published science content, returns the `recognize` skill, and derives `review_due` from a mastered state whose `due_at` is past.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-server && go test ./internal/progress -count=1`

- [ ] **Step 3: Implement responses**

```go
type ItemProgress struct {
    KpID int64 `json:"kpId"`
    Title string `json:"title"`
    Status string `json:"status"`
    RecognizeStatus string `json:"recognizeStatus"`
}

type ModuleProgress struct {
    Code string `json:"code"`
    Name string `json:"name"`
    Total int `json:"total"`
    Mastered int `json:"mastered"`
    ReviewDue int `json:"reviewDue"`
    Items []ItemProgress `json:"items"`
}
```

Validate the child exists before querying progress. Missing skill/state rows map to `not_started`.

- [ ] **Step 4: Add endpoints**

```http
GET /api/v1/children/:childId/science/progress
GET /api/v1/children/:childId/science/home
```

Home returns active/today plan summary, review-due count, explored count and module summaries.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-server && go test ./internal/progress ./internal/http -count=1`

```bash
git add kid-workbench/science-server/internal/progress kid-workbench/science-server/internal/http
git commit -m "feat: expose science learning progress"
```

### Task 5: Create Stable Science Plans

**Files:**
- Create: `kid-workbench/science-server/internal/plan/model.go`
- Create: `kid-workbench/science-server/internal/plan/service.go`
- Create: `kid-workbench/science-server/internal/plan/service_test.go`
- Create: `kid-workbench/science-server/internal/http/handler_plan.go`
- Modify: `kid-workbench/science-server/internal/http/router.go`

- [ ] **Step 1: Write plan scope and stability tests**

Assert daily, module and review plans:

- set `subject_code = 'science'`;
- contain only published science `recognize` questions;
- contain each KP at most once;
- preserve question IDs and option order on repeated reads;
- reject unknown/non-science module codes.
- restrict review plans to wrong or review-due science KPs.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-server && go test ./internal/plan -count=1`

- [ ] **Step 3: Verify the shared option-order prerequisite**

Run: `cd kid-workbench/shared-go && go test ./model -run TestPlanItemHasPersistedOptionOrder -count=1`

Expected: PASS. Do not create another migration in `science-server`.

- [ ] **Step 4: Implement safe plan DTOs**

```go
type CreateInput struct {
    Mode string `json:"mode" binding:"required,oneof=daily module review"`
    ModuleCode string `json:"moduleCode"`
}

type QuestionDTO struct {
    ID int64 `json:"id"`
    Stem string `json:"stem"`
    Options []OptionDTO `json:"options"`
    SenseImageURL string `json:"senseImageUrl,omitempty"`
    SpeechURL string `json:"speechUrl,omitempty"`
}
```

Store a comma-separated original-index permutation such as `2,0,3,1`. Never serialize `questions.answer`. Older plan rows with empty order use and persist identity order `0,1,2,3`.

`review` selects only the child's wrong or review-due published science KPs, prioritizing the most recent wrong attempts. `moduleCode` must be empty for `daily` and `review`, and required for `module`.

- [ ] **Step 5: Add routes**

```http
POST /api/v1/children/:childId/science/plans
GET  /api/v1/children/:childId/science/plans/:planId
POST /api/v1/children/:childId/science/plans/:planId/start
```

Reject access unless both `child_id` and `subject_code = 'science'` match.

- [ ] **Step 6: Test and commit**

Run: `cd kid-workbench/science-server && go test ./internal/plan ./internal/http -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/db ./internal/service -count=1`

```bash
git add kid-workbench/science-server
git commit -m "feat: create stable science plans"
```

### Task 6: Implement Atomic Answering and Finish

**Files:**
- Create: `kid-workbench/science-server/internal/practice/service.go`
- Create: `kid-workbench/science-server/internal/practice/service_test.go`
- Modify: `kid-workbench/science-server/internal/http/handler_plan.go`

- [ ] **Step 1: Write retry, explanation and idempotency tests**

Submit one wrong answer, retry correctly, repeat the second request with the same `clientId`, then finish. Assert two attempts, one completed item, one correct plan item, no duplicate stats/reward, and explanation returned only after correct or exhausted attempts.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-server && go test ./internal/practice -count=1`

- [ ] **Step 3: Implement contracts**

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
    Explanation string `json:"explanation,omitempty"`
    Mastery learning.StateDTO `json:"mastery"`
}
```

Map displayed indexes through `option_order`, call shared `learning.Service.ApplyOne` inside the same GORM transaction, then update plan item and plan counters before commit. Pass `clientId` unchanged.

- [ ] **Step 4: Add endpoints**

```http
POST /api/v1/children/:childId/science/plans/:planId/items/:itemId/answer
POST /api/v1/children/:childId/science/plans/:planId/finish
```

Finishing incomplete plans returns `409 plan_incomplete`; finishing a completed plan replays its existing result.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-server && go test ./internal/practice ./internal/http -count=1`

```bash
git add kid-workbench/science-server/internal/practice kid-workbench/science-server/internal/http
git commit -m "feat: record science practice results"
```

### Task 7: Add Docker and Workspace Integration

**Files:**
- Create: `kid-workbench/science-server/Dockerfile`
- Create: `kid-workbench/science-server/Makefile`
- Create: `kid-workbench/science-server/README.md`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Add a multi-stage server image**

Build from the `kid-workbench` context so the image can copy both `science-server` and `shared-go`. Run as a non-root user and expose `19121`.

- [ ] **Step 2: Add Compose service**

Add `science-server` on `19121:19121`, reuse `db_env`, configure MinIO read-only credentials through environment variables, and depend on PostgreSQL health plus seed completion. Do not add `content-admin` or `backend` as dependencies.

- [ ] **Step 3: Verify build and runtime boundary**

Run: `cd kid-workbench && docker build -f science-server/Dockerfile .`

Run: `cd kid-workbench && docker compose config`

Run: `cd kid-workbench/science-server && go test ./... -count=1`

Expected: build succeeds; Compose has no `science-server -> content-admin/backend` dependency.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/science-server kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "build: deploy independent science server"
```

### Task 8: Cross-System Acceptance

**Files:**
- Create: `kid-workbench/science-server/scripts/acceptance.sh`
- Modify: `kid-workbench/science-server/README.md`

- [ ] **Step 1: Add a bounded acceptance script**

The script must create or reuse one science plan through `19121`, answer its first item with a UUID client ID, and query counts for that exact plan/child. It must not truncate shared tables or regenerate assets.

- [ ] **Step 2: Run with content-admin available**

Verify catalog, asset, plan, answer and finish endpoints.

- [ ] **Step 3: Stop only content-admin and rerun reads**

Verify published knowledge cards, existing MinIO assets and plan answering continue through `science-server`. Restart content-admin afterward; do not remove volumes.

- [ ] **Step 4: Verify parent dashboard visibility**

Query the parent API for the same child and KP; assert it displays the science attempt and updated mastery without calling science-server.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/science-server/scripts/acceptance.sh kid-workbench/science-server/README.md
git commit -m "test: cover science server integration"
```
