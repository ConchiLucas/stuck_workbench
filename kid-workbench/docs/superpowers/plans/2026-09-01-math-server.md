# Math Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go API that serves math-app, reads published math content and TTS directly from PostgreSQL/MinIO, and atomically records plans, attempts, module skills, mastery, statistics, and rewards.

**Architecture:** `math-server` is the only API used by `math-app`. It is a separate Go module and process, never calls ports 19091 or 19081, scopes every business query to subject `math`, serves plan-scoped audio from MinIO, snapshots complete questions into plan items, and calls `shared-go/learning` inside its plan transaction.

**Tech Stack:** Go 1.26.1, Gin 1.12, GORM 1.31, PostgreSQL 16, SQLite tests, MinIO Go SDK, Testify, Docker

**Prerequisite:** Complete `2026-09-01-math-learning-foundation.md`. Migration 008, module-aware skills, `plan_items.question_snapshot`, and question-level `questions.media_url` are owned only by that plan.

---

### Task 1: Scaffold the Independent Service

**Files:**
- Create: `kid-workbench/math-server/go.mod`
- Create: `kid-workbench/math-server/internal/config/config.go`
- Create: `kid-workbench/math-server/internal/db/db.go`
- Create: `kid-workbench/math-server/internal/http/response.go`
- Create: `kid-workbench/math-server/internal/http/router.go`
- Create: `kid-workbench/math-server/internal/http/router_test.go`
- Create: `kid-workbench/math-server/cmd/server/main.go`

- [ ] **Step 1: Write failing liveness and envelope tests**

```go
func TestHealthz(t *testing.T) {
    router := httpapi.NewRouter(httpapi.Deps{})
    response := httptest.NewRecorder()
    router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
    require.Equal(t, http.StatusOK, response.Code)
    require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
}

func TestAPIErrorEnvelope(t *testing.T) {
    require.Equal(t, httpapi.APIError{Code: "child_not_found", Message: "孩子不存在"},
        httpapi.APIError{Code: "child_not_found", Message: "孩子不存在"})
}
```

- [ ] **Step 2: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/http -count=1`

Expected: FAIL because the module and router do not exist.

- [ ] **Step 3: Create module and configuration**

Use module `github.com/conchi/math-server`, Go `1.26.1`, and:

```go
replace github.com/conchi/study-learning => ../shared-go
```

Match the live pinyin-server database and MinIO configuration fields, changing only the default address to `:19141`. Load `APP_DB_*`, `APP_DSN`, `APP_MINIO_*`, and mastery settings.

- [ ] **Step 4: Implement DB opening, envelope, and router**

Use `gorm.Config{TranslateError: true}` for PostgreSQL and SQLite. Do not run schema migrations from math-server.

```go
type APIError struct {
    Code    string `json:"code"`
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

Run: `cd kid-workbench/math-server && go mod tidy && go test ./... -count=1`

```bash
git add kid-workbench/math-server
git commit -m "feat: scaffold math server"
```

### Task 2: Add Math Catalog and Exclusive Stages

**Files:**
- Create: `kid-workbench/math-server/internal/catalog/model.go`
- Create: `kid-workbench/math-server/internal/catalog/stage.go`
- Create: `kid-workbench/math-server/internal/catalog/stage_test.go`
- Create: `kid-workbench/math-server/internal/catalog/repository.go`
- Create: `kid-workbench/math-server/internal/catalog/repository_test.go`
- Create: `kid-workbench/math-server/internal/http/handler_catalog.go`
- Modify: `kid-workbench/math-server/internal/http/router.go`

- [ ] **Step 1: Write failing stage boundary tests**

```go
func TestArithmeticStage(t *testing.T) {
    tests := []struct{ module string; a, b int; want string }{
        {"add10", 2, 3, "within5"}, {"add10", 4, 6, "within10"},
        {"add10", 8, 7, "within20"}, {"sub10", 5, 2, "within5"},
        {"sub10", 10, 4, "within10"}, {"sub10", 20, 8, "within20"},
    }
    for _, tc := range tests {
        require.Equal(t, tc.want, catalog.StageFor(tc.module, tc.a, tc.b))
    }
    require.Equal(t, "basic-shapes", catalog.StageFor("shape", 0, 0))
}
```

- [ ] **Step 2: Write a cross-subject repository test**

Seed math and literacy subjects, similarly named modules, and math KPs. Assert only math modules/items are returned and raw `knowledge_points.payload` is not present in serialized DTOs.

- [ ] **Step 3: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/catalog -count=1`

Expected: FAIL because catalog types and functions do not exist.

- [ ] **Step 4: Define child-safe catalog DTOs**

```go
type Module struct {
    Code      string  `json:"code"`
    Name      string  `json:"name"`
    OrderNo   int     `json:"orderNo"`
    ItemCount int     `json:"itemCount"`
    Stages    []Stage `json:"stages"`
}

type Stage struct {
    Code      string `json:"code"`
    Name      string `json:"name"`
    ItemCount int    `json:"itemCount"`
}

type LearningItem struct {
    KpID       int64  `json:"kpId"`
    Title      string `json:"title"`
    ModuleCode string `json:"moduleCode"`
    StageCode  string `json:"stageCode"`
    Kind       string `json:"kind"`
    A          int    `json:"a,omitempty"`
    B          int    `json:"b,omitempty"`
    Shape      string `json:"shape,omitempty"`
}
```

Parse payload in the repository and emit only these validated fields. Reject unknown kinds instead of exposing raw JSON.

- [ ] **Step 5: Add exact routes**

```http
GET /api/v1/math/modules
GET /api/v1/math/modules/:moduleCode
GET /api/v1/math/modules/:moduleCode/stages/:stageCode
```

Valid modules are `add10|sub10|shape`; valid stages are determined by the module. Unknown combinations return `404 catalog_not_found`.

- [ ] **Step 6: Test and commit**

Run: `cd kid-workbench/math-server && go test ./internal/catalog ./internal/http -count=1`

```bash
git add kid-workbench/math-server/internal/catalog kid-workbench/math-server/internal/http
git commit -m "feat: expose staged math catalog"
```

### Task 3: Add Child-Scoped Progress

**Files:**
- Create: `kid-workbench/math-server/internal/progress/service.go`
- Create: `kid-workbench/math-server/internal/progress/service_test.go`
- Create: `kid-workbench/math-server/internal/http/handler_progress.go`
- Modify: `kid-workbench/math-server/internal/http/router.go`

- [ ] **Step 1: Write failing module-dependent progress tests**

Seed one `add10` KP with only `calc` mastered, one `shape` KP with `find` and `name` mastered, and unrelated pinyin rows. Assert:

```go
require.Equal(t, "learning", result.Modules[0].Stages[0].Items[0].Status)
require.Equal(t, []string{"calc", "story"}, skillCodes(result.Modules[0].Stages[0].Items[0]))
require.Equal(t, "mastered", shapeItem.Status)
require.Equal(t, []string{"find", "name"}, skillCodes(shapeItem))
```

- [ ] **Step 2: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/progress -count=1`

Expected: FAIL because progress service does not exist.

- [ ] **Step 3: Implement progress DTOs and query**

```go
type SkillProgress struct {
    Code   string `json:"code"`
    Status string `json:"status"`
}

type ItemProgress struct {
    KpID      int64           `json:"kpId"`
    Title     string          `json:"title"`
    Status    string          `json:"status"`
    Skills    []SkillProgress `json:"skills"`
}
```

Validate the child exists. Load only math KPs and their `mastery_states`/`mastery_skills`; synthesize missing required skills as `not_started` using `mastery.SkillsFor("math", moduleCode)`.

- [ ] **Step 4: Add endpoint and error mapping**

```http
GET /api/v1/children/:childId/math/progress
```

Unknown child returns `404 child_not_found`. Invalid child ID returns `400 invalid_child_id`.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/math-server && go test ./internal/progress ./internal/http -count=1`

```bash
git add kid-workbench/math-server/internal/progress kid-workbench/math-server/internal/http
git commit -m "feat: expose math learning progress"
```

### Task 4: Add Read-Only MinIO Access

**Files:**
- Create: `kid-workbench/math-server/internal/asset/store.go`
- Create: `kid-workbench/math-server/internal/asset/service.go`
- Create: `kid-workbench/math-server/internal/asset/service_test.go`

- [ ] **Step 1: Write read and availability tests**

```go
func TestQuestionAudioReadsSnapshotKey(t *testing.T) {
    reader := &fakeReader{data: []byte("mp3")}
    service := asset.NewService(reader)
    got, err := service.QuestionAudio(context.Background(), "math/questions/42.mp3")
    require.NoError(t, err)
    require.Equal(t, []byte("mp3"), got)
    require.Equal(t, "math/questions/42.mp3", reader.key)
}
```

Reject keys outside `math/questions/` with `asset.ErrInvalidKey`. Map MinIO 404 to `asset.ErrUnavailable` only for transient serving; the app contract does not define an audio-missing fallback.

- [ ] **Step 2: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/asset -count=1`

Expected: FAIL because asset package does not exist.

- [ ] **Step 3: Implement a read-only store**

Use the same MinIO config and client pattern as live `pinyin-server/internal/asset`, but expose only:

```go
type Reader interface {
    Get(context.Context, string) ([]byte, error)
}

type Service struct { reader Reader }

func (s *Service) QuestionAudio(ctx context.Context, key string) ([]byte, error)
```

Do not add upload, TTS, presign, or public URL methods.

- [ ] **Step 4: Test and commit**

Run: `cd kid-workbench/math-server && go test ./internal/asset -count=1`

```bash
git add kid-workbench/math-server/internal/asset
git commit -m "feat: read math question audio"
```

### Task 5: Create Stable Daily, Module, and Review Plans

**Files:**
- Create: `kid-workbench/math-server/internal/plan/model.go`
- Create: `kid-workbench/math-server/internal/plan/snapshot.go`
- Create: `kid-workbench/math-server/internal/plan/snapshot_test.go`
- Create: `kid-workbench/math-server/internal/plan/candidates.go`
- Create: `kid-workbench/math-server/internal/plan/candidates_test.go`
- Create: `kid-workbench/math-server/internal/plan/service.go`
- Create: `kid-workbench/math-server/internal/plan/service_test.go`
- Create: `kid-workbench/math-server/internal/http/handler_plan.go`
- Modify: `kid-workbench/math-server/internal/http/router.go`

- [ ] **Step 1: Write failing snapshot secrecy and stability tests**

Create a plan, mutate the original `questions` row, reload the plan, and assert stem/options/audio key remain the original values. Marshal the public DTO and assert it does not contain `answerIndex`, `audioObjectKey`, or raw payload.

- [ ] **Step 2: Write failing plan-kind and quota tests**

Assert:

- repeated `daily` creation on the same child/date returns the same plan;
- a full daily plan contains 4 `add10`, 4 `sub10`, and 2 `shape` items;
- module plan contains only the requested module/stage;
- review plan contains only due, shaky, or recently wrong math KPs and may contain fewer than 10;
- all items have question codes valid for their module.

- [ ] **Step 3: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/plan -count=1`

Expected: FAIL because plan package does not exist.

- [ ] **Step 4: Define private snapshot and public DTOs**

```go
type QuestionSnapshot struct {
    QuestionID    int64           `json:"questionId"`
    Code          string          `json:"code"`
    Stem          string          `json:"stem"`
    Options       json.RawMessage `json:"options"`
    AnswerIndex   int             `json:"answerIndex"`
    Visual        Visual          `json:"visual"`
    AudioObjectKey string         `json:"audioObjectKey"`
}

type QuestionDTO struct {
    QuestionID int64           `json:"questionId"`
    Code       string          `json:"code"`
    Stem       string          `json:"stem"`
    Options    json.RawMessage `json:"options"`
    Visual     Visual          `json:"visual"`
    AudioURL   string          `json:"audioUrl"`
}

type Visual struct {
    Kind       string `json:"kind"`
    A          int    `json:"a,omitempty"`
    B          int    `json:"b,omitempty"`
    Operator   string `json:"operator,omitempty"`
    LeftCount  int    `json:"leftCount,omitempty"`
    RightCount int    `json:"rightCount,omitempty"`
    Object     string `json:"object,omitempty"`
    Shape      string `json:"shape,omitempty"`
}
```

Parse and validate four options plus `questions.answer.index` before snapshot creation. Normalize stored question data by code: `calc` emits `kind=equation` with `a/b/operator` and discards count hints; `story` emits `kind=add|sub` with `leftCount/rightCount` and maps `🍎` to `apple` and `🍓` to `strawberry`; `find` emits `kind=none`; `name` emits `kind=shape` with the controlled SVG key. Reject unrecognized objects/shapes. Never pass stored `questions.visual` through without validation.

- [ ] **Step 5: Implement deterministic candidate selection**

Candidate SQL joins `questions -> knowledge_points -> modules -> subjects`, requires `s.code='math'` and `q.media_url LIKE 'math/questions/%.mp3'`, and left joins mastery by `q.code`. Assign priority `review_due`, `shaky`, `learning`, then `new`; use KP/question IDs as deterministic tie-breakers after a plan-specific seeded shuffle.

Daily quotas are `add10=4`, `sub10=4`, `shape=2`; backfill shortages from remaining math candidates. Within each quota, alternate the module's two skills when candidates exist. Module mode filters `moduleCode` and the exclusive stage function. Review mode selects no new items and returns at most 10.

- [ ] **Step 6: Persist plans and snapshots atomically**

```go
type CreateInput struct {
    Kind       string `json:"kind" binding:"required,oneof=daily module review"`
    ModuleCode string `json:"moduleCode"`
    StageCode  string `json:"stageCode"`
}
```

Set `study_plans.subject_code='math'` plus scope fields. Use the next `seq_no` for non-daily plans. Insert each selected question's serialized snapshot into `plan_items.question_snapshot`. Never read live question fields when returning an existing math-server plan.

- [ ] **Step 7: Add routes**

```http
POST /api/v1/children/:childId/math/plans
GET  /api/v1/children/:childId/math/plans/:planId
POST /api/v1/children/:childId/math/plans/:planId/start
```

Require matching `child_id`, `subject_code='math'`, and non-empty `plan_kind`; otherwise return 404.

- [ ] **Step 8: Run and commit**

Run: `cd kid-workbench/math-server && go test ./internal/plan ./internal/http -count=1`

```bash
git add kid-workbench/math-server/internal/plan kid-workbench/math-server/internal/http
git commit -m "feat: create stable math plans"
```

### Task 6: Serve Controlled Learning and Plan-Scoped Audio

**Files:**
- Create: `kid-workbench/math-server/internal/http/handler_audio.go`
- Modify: `kid-workbench/math-server/internal/http/router.go`
- Modify: `kid-workbench/math-server/internal/http/router_test.go`

- [ ] **Step 1: Write authorization and MIME tests**

Assert a matching child/plan/item returns `audio/mpeg`; another child, another plan, a non-math plan, and an item outside the plan all return 404. For learning audio, assert a valid child/math KP/code succeeds while another subject, unknown code, or code belonging to another KP returns 404. Assert storage errors return `503 audio_unavailable`.

- [ ] **Step 2: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/http -run TestPlanItemAudio -count=1`

Expected: FAIL with 404 because the route is absent.

- [ ] **Step 3: Implement the scoped route**

```http
GET /api/v1/children/:childId/math/items/:kpId/audio/:code.mp3
GET /api/v1/children/:childId/math/plans/:planId/items/:itemId/audio.mp3
```

For learning audio, validate the child, math KP, and one of `calc|story|find|name`, then load `questions.media_url` through the matching `kp_id + code`. For plan audio, load the matching plan item and private snapshot. In both cases pass only the server-loaded object key to `asset.Service` and return:

```text
Content-Type: audio/mpeg
Cache-Control: private, max-age=86400
```

Do not accept a question ID or object key from request parameters.

- [ ] **Step 4: Test and commit**

Run: `cd kid-workbench/math-server && go test ./internal/http ./internal/asset -count=1`

```bash
git add kid-workbench/math-server/internal/http
git commit -m "feat: serve plan-scoped math audio"
```

### Task 7: Implement Two-Try Idempotent Answering and Finish

**Files:**
- Create: `kid-workbench/math-server/internal/practice/model.go`
- Create: `kid-workbench/math-server/internal/practice/service.go`
- Create: `kid-workbench/math-server/internal/practice/service_test.go`
- Modify: `kid-workbench/math-server/internal/http/handler_plan.go`

- [ ] **Step 1: Write failing two-try, replay, and rollback tests**

Submit first wrong, duplicate the same `clientId`, then submit a correct second try. Assert exactly two attempts, `tries=2`, item `correct`, one daily-stat increment per unique attempt, and no duplicate reward. Add a forced plan-update failure and assert attempts/mastery/stat writes roll back.

- [ ] **Step 2: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/practice -count=1`

Expected: FAIL because practice service does not exist.

- [ ] **Step 3: Define exact contract**

```go
type AnswerInput struct {
    ClientID   string `json:"clientId" binding:"required"`
    OptionIndex int   `json:"optionIndex" binding:"min=0,max=3"`
    CostMs      int   `json:"costMs" binding:"min=0,max=3600000"`
}

type AnswerResult struct {
    Correct     bool              `json:"correct"`
    AnswerIndex int               `json:"answerIndex"`
    CanRetry    bool              `json:"canRetry"`
    Tries       int               `json:"tries"`
    Status      string            `json:"status"`
    Mastery     learning.StateDTO `json:"mastery"`
    Plan        PlanSummary       `json:"plan"`
}
```

- [ ] **Step 4: Implement transactional answer logic**

Before applying, query `attempts` by `child_id + client_id`; if found, reconstruct the same response without another write. Otherwise load and validate the plan/item, parse the private snapshot, compare `optionIndex` with `AnswerIndex`, and call:

```go
state, applied, err := sharedLearning.ApplyOne(tx, childID, learning.AttemptInput{
    ClientID: in.ClientID, KpID: item.KpID, QuestionID: &item.QuestionID,
    IsCorrect: correct, CostMs: in.CostMs, Source: mastery.SourceQuiz,
})
```

In the same transaction append `Picks`, increment `Tries`, leave status pending after the first wrong, and finish the item after correct or second wrong. Recalculate plan counts from plan items.

- [ ] **Step 5: Implement finish semantics and routes**

```http
POST /api/v1/children/:childId/math/plans/:planId/items/:itemId/answer
POST /api/v1/children/:childId/math/plans/:planId/finish
```

Incomplete finish returns `409 plan_incomplete`; completed finish is replayable. Calculate stars from `correct/target`: ratio at least 0.9 gives 3, at least 0.7 gives 2, otherwise 1. A completed plan awards `1 + stars` flowers exactly once with `reason=plan_done`, `ref_type=study_plan`, and `ref_id=planID`.

- [ ] **Step 6: Run and commit**

Run: `cd kid-workbench/math-server && go test ./internal/practice ./internal/http ./... -count=1`

```bash
git add kid-workbench/math-server/internal/practice kid-workbench/math-server/internal/http
git commit -m "feat: record math practice atomically"
```

### Task 8: Add Home Summary and Review Entry Points

**Files:**
- Create: `kid-workbench/math-server/internal/home/service.go`
- Create: `kid-workbench/math-server/internal/home/service_test.go`
- Create: `kid-workbench/math-server/internal/http/handler_home.go`
- Modify: `kid-workbench/math-server/internal/http/router.go`

- [ ] **Step 1: Write failing subject-isolated home tests**

Seed active math and literacy plans plus due mastery from both subjects. Assert home returns only the math plan, math due count, child flowers, and three module summaries.

- [ ] **Step 2: Run and verify failure**

Run: `cd kid-workbench/math-server && go test ./internal/home -count=1`

Expected: FAIL because home service does not exist.

- [ ] **Step 3: Implement the response and endpoint**

```go
type Home struct {
    Child   ChildSummary    `json:"child"`
    Plan    *PlanSummary    `json:"plan"`
    DueCount int            `json:"dueCount"`
    Modules []ModuleSummary `json:"modules"`
}
```

```http
GET /api/v1/children/:childId/math/home
```

An active plan is the newest pending/doing math-server plan for the child. Module totals and mastered counts derive only from subject math.

- [ ] **Step 4: Test and commit**

Run: `cd kid-workbench/math-server && go test ./internal/home ./internal/http -count=1`

```bash
git add kid-workbench/math-server/internal/home kid-workbench/math-server/internal/http
git commit -m "feat: add math child home summary"
```

### Task 9: Add Docker, Readiness, and Cross-System Acceptance

**Files:**
- Create: `kid-workbench/math-server/Dockerfile`
- Create: `kid-workbench/math-server/README.md`
- Create: `kid-workbench/math-server/scripts/acceptance.sh`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Add readiness contract**

Add `/readyz` that pings PostgreSQL and performs a bounded MinIO bucket check. Return 200 only when both are reachable; keep `/healthz` as process liveness.

- [ ] **Step 2: Add Docker build and Compose service**

Build with context `kid-workbench` so `shared-go` remains available for the local replace. Expose `19141`, reuse `*db_env`, pass `APP_MINIO_*`, depend on healthy PostgreSQL, join `vibedeploy-shared`, and do not depend on `backend`, `content-admin`, or `pinyin-server`.

- [ ] **Step 3: Write a bounded acceptance script**

The script must create one module-scoped math plan for child `1`, submit one answer using a unique `clientId`, and query only the created plan/attempt plus matching mastery rows. It must not delete or reset existing data and must not print credentials.

- [ ] **Step 4: Run all backend checks**

Run: `cd kid-workbench/math-server && go test ./... -count=1`

Run: `cd kid-workbench && docker compose config --quiet`

Run: `cd kid-workbench && docker compose build math-server`

Expected: PASS.

- [ ] **Step 5: Run cross-system acceptance**

Start PostgreSQL, seeded content, MinIO connectivity, and math-server. Run the acceptance script, then verify parent-dashboard queries show the resulting math skill/KP state. Stop content-admin and parent backend, repeat plan loading/answering, and inspect logs to confirm no requests to ports 19091 or 19081.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/math-server kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "build: add standalone math server"
```
