# Shared Learning Engine Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the existing mastery and atomic attempt-update behavior into a reusable local Go module without changing parent-dashboard behavior.

**Architecture:** Create `kid-workbench/shared-go` as `github.com/conchi/study-learning`. Move pure mastery rules first, then expose a focused `learning.Service.ApplyOne` that operates inside a caller-owned GORM transaction. Keep parent-dashboard HTTP and plan behavior unchanged through a thin compatibility wrapper.

**Tech Stack:** Go 1.26, GORM, PostgreSQL, SQLite test database, Testify

---

### Task 1: Lock Existing Learning Behavior

**Files:**
- Modify: `kid-workbench/parent-dashboard/backend/internal/service/service_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/service/plan_test.go`

- [ ] **Step 1: Add a failing transaction-rollback test**

Add `TestApplyOneRollsBackAllLearningWrites` that starts `repo.Tx`, calls `ApplyOne`, deliberately returns `errors.New("rollback")`, and then asserts zero matching rows in `attempts`, `mastery_states`, `mastery_skills`, `daily_stats`, and `flower_ledger`.

- [ ] **Step 2: Run the focused tests**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/service -run 'Test(ReportIsIdempotent|ApplyOneRollsBackAllLearningWrites|Plan)' -count=1`

Expected: the new rollback test passes against the current implementation; if it exposes an existing partial write, stop and fix that behavior before extraction.

- [ ] **Step 3: Add a pinyin skill aggregation regression test**

Seed one `inword` and one `listen` question for the same pinyin knowledge point, submit both correctly through `ApplyOne`, and assert two `mastery_skills` rows plus one rolled-up `mastery_states` row.

- [ ] **Step 4: Run the complete parent backend suite**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the baseline tests**

```bash
git add kid-workbench/parent-dashboard/backend/internal/service/service_test.go kid-workbench/parent-dashboard/backend/internal/service/plan_test.go
git commit -m "test: lock shared learning behavior"
```

### Task 2: Create the Shared Module and Move Mastery Rules

**Files:**
- Create: `kid-workbench/shared-go/go.mod`
- Create: `kid-workbench/shared-go/mastery/engine.go`
- Create: `kid-workbench/shared-go/mastery/engine_test.go`
- Create: `kid-workbench/shared-go/mastery/skills.go`
- Create: `kid-workbench/shared-go/mastery/skills_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/go.mod`
- Modify: all parent backend imports of `internal/mastery`
- Delete: `kid-workbench/parent-dashboard/backend/internal/mastery/engine.go`
- Delete: `kid-workbench/parent-dashboard/backend/internal/mastery/engine_test.go`
- Delete: `kid-workbench/parent-dashboard/backend/internal/mastery/skills.go`
- Delete: `kid-workbench/parent-dashboard/backend/internal/mastery/skills_test.go`

- [ ] **Step 1: Create the module file**

```go
module github.com/conchi/study-learning

go 1.26.1

require (
    github.com/stretchr/testify v1.12.1
    gorm.io/gorm v1.31.2
)
```

- [ ] **Step 2: Move mastery files without changing behavior**

Run from the repository root:

```bash
mkdir -p kid-workbench/shared-go/mastery
git mv kid-workbench/parent-dashboard/backend/internal/mastery/engine.go kid-workbench/shared-go/mastery/engine.go
git mv kid-workbench/parent-dashboard/backend/internal/mastery/engine_test.go kid-workbench/shared-go/mastery/engine_test.go
git mv kid-workbench/parent-dashboard/backend/internal/mastery/skills.go kid-workbench/shared-go/mastery/skills.go
git mv kid-workbench/parent-dashboard/backend/internal/mastery/skills_test.go kid-workbench/shared-go/mastery/skills_test.go
```

Keep `package mastery` unchanged.

- [ ] **Step 3: Wire the local module**

Add to `parent-dashboard/backend/go.mod`:

```go
require github.com/conchi/study-learning v0.0.0

replace github.com/conchi/study-learning => ../../shared-go
```

Replace `github.com/conchi/study-workbench/internal/mastery` imports with `github.com/conchi/study-learning/mastery`.

- [ ] **Step 4: Run both module test suites**

Run: `cd kid-workbench/shared-go && go mod tidy && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go mod tidy && go test ./... -count=1`

Expected: both PASS with unchanged mastery assertions.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/shared-go kid-workbench/parent-dashboard/backend
git commit -m "refactor: extract shared mastery engine"
```

### Task 3: Define Shared Learning Models

**Files:**
- Create: `kid-workbench/shared-go/model/model.go`
- Create: `kid-workbench/shared-go/model/model_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/model/model.go`

- [ ] **Step 1: Write model table-name tests**

```go
func TestSharedLearningTableNames(t *testing.T) {
    require.Equal(t, "attempts", (model.Attempt{}).TableName())
    require.Equal(t, "mastery_states", (model.MasteryState{}).TableName())
    require.Equal(t, "mastery_skills", (model.MasterySkill{}).TableName())
    require.Equal(t, "daily_stats", (model.DailyStat{}).TableName())
    require.Equal(t, "flower_ledger", (model.FlowerLedger{}).TableName())
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd kid-workbench/shared-go && go test ./model -count=1`

Expected: FAIL because the shared model package does not exist.

- [ ] **Step 3: Add only transaction-owned models**

Move the exact current definitions of `Child`, `KnowledgePoint`, `Question`, `Attempt`, `MasteryState`, `MasterySkill`, `DailyStat`, and `FlowerLedger` into `shared-go/model/model.go`. Preserve field names, GORM tags, JSON tags, and table names exactly.

- [ ] **Step 4: Alias shared types from the parent model package**

In `parent-dashboard/backend/internal/model/model.go`, replace the moved definitions with aliases:

```go
type Child = learningmodel.Child
type KnowledgePoint = learningmodel.KnowledgePoint
type Question = learningmodel.Question
type Attempt = learningmodel.Attempt
type MasteryState = learningmodel.MasteryState
type MasterySkill = learningmodel.MasterySkill
type DailyStat = learningmodel.DailyStat
type FlowerLedger = learningmodel.FlowerLedger
```

Keep parent-specific `User`, `Subject`, `Module`, `StudyPlan`, `PlanItem`, `DailyTask`, and `Reward` definitions local.

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

```bash
git add kid-workbench/shared-go/model kid-workbench/parent-dashboard/backend/internal/model
git commit -m "refactor: share learning data models"
```

### Task 4: Extract Atomic Attempt Application

**Files:**
- Create: `kid-workbench/shared-go/learning/repository.go`
- Create: `kid-workbench/shared-go/learning/service.go`
- Create: `kid-workbench/shared-go/learning/service_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/service/attempt.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/repo/repo.go`

- [ ] **Step 1: Write the shared idempotency test**

Create a SQLite test schema with one child, one pinyin knowledge point, one `inword` question, and all transaction-owned tables. Call `ApplyOne` twice with `ClientID: "pinyin-plan-1-item-1-try-1"`; assert one attempt and one increment in daily stats.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd kid-workbench/shared-go && go test ./learning -run TestApplyOneIsIdempotent -count=1`

Expected: FAIL because `learning.NewService` is undefined.

- [ ] **Step 3: Define the public API**

```go
type AttemptInput struct {
    ClientID string
    KpID int64
    QuestionID *int64
    IsCorrect bool
    CostMs int
    Source string
    At time.Time
}

type StateDTO struct {
    KpID int64 `json:"kp_id"`
    Status string `json:"status"`
    Attempts int `json:"attempts"`
    Correct int `json:"correct"`
    DueAt *time.Time `json:"due_at"`
}

func (s *Service) ApplyOne(tx *gorm.DB, childID int64, in AttemptInput) (StateDTO, bool, error)
```

Move the current `ApplyOne`, skill aggregation, rollup, model conversion, daily-stat update, and reward write behavior into this package. Preserve the current unique-violation idempotency behavior.

- [ ] **Step 4: Make parent AttemptService delegate**

Keep the parent public methods `Report`, `MarkMastered`, and `UndoMark`. Add a `shared *learning.Service` field and delegate `ApplyOne` with explicit input/output conversion so existing HTTP and plan code does not change.

- [ ] **Step 5: Run rollback, idempotency, skill and plan tests**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/service ./internal/http -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/shared-go/learning kid-workbench/parent-dashboard/backend/internal/service/attempt.go kid-workbench/parent-dashboard/backend/internal/repo/repo.go
git commit -m "refactor: share atomic attempt updates"
```

### Task 5: Make Docker Builds Resolve the Shared Module

**Files:**
- Modify: `kid-workbench/parent-dashboard/backend/Dockerfile`
- Modify: `kid-workbench/parent-dashboard/backend/Makefile`

- [ ] **Step 1: Add a Docker build regression check**

Run: `cd kid-workbench && docker build -f parent-dashboard/backend/Dockerfile --target backend .`

Expected before modification: FAIL because `../../shared-go` is unavailable at the module download step.

- [ ] **Step 2: Copy shared module metadata before download**

In both backend and seed build stages, copy these paths before `go mod download`:

```dockerfile
COPY shared-go/go.mod shared-go/go.sum* /shared-go/
COPY parent-dashboard/backend/go.mod parent-dashboard/backend/go.sum ./
RUN go mod download
COPY shared-go/ /shared-go/
COPY parent-dashboard/backend/ ./
```

The working directory must keep the relative replace target `../../shared-go` valid; use `/src/parent-dashboard/backend` with the shared module at `/src/shared-go`.

- [ ] **Step 3: Run local and Docker verification**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Run: `cd kid-workbench && docker build -f parent-dashboard/backend/Dockerfile --target backend .`

Expected: both PASS.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/parent-dashboard/backend/Dockerfile kid-workbench/parent-dashboard/backend/Makefile
git commit -m "build: include shared learning module"
```

### Task 6: Final Extraction Verification

**Files:**
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Document the module boundary**

Add `shared-go/` to the directory tree and state that it contains shared mastery and atomic learning-write rules, not HTTP handlers or subject content rules.

- [ ] **Step 2: Verify no parent imports remain**

Run: `rg 'study-workbench/internal/mastery' kid-workbench`

Expected: no matches.

- [ ] **Step 3: Run all relevant checks**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Run: `cd kid-workbench && docker compose config --quiet`

Expected: all commands succeed.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/README.md
git commit -m "docs: describe shared learning module"
```
