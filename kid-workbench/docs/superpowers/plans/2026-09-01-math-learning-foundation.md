# Math Learning Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the shared learning engine, shared schema, and content-admin publishing flow so standalone math practice can use module-specific mastery, immutable plan snapshots, and pre-generated question-level TTS.

**Architecture:** Keep cross-subject learning writes in `shared-go`, keep schema ownership in parent-dashboard migrations, and keep all TTS generation in content-admin. Math child services consume these foundations but do not generate content or audio.

**Tech Stack:** Go 1.26.1, GORM, PostgreSQL 16, SQLite tests, Gin, React 19, TypeScript, TanStack Query, MinIO, Testify

**Prerequisite:** Finish or checkpoint the active `shared-go` extraction so `go test ./...` passes in both `shared-go` and `parent-dashboard/backend` before starting this plan.

---

### Task 1: Add Module-Aware Math Skill Definitions

**Files:**
- Modify: `kid-workbench/shared-go/mastery/skills.go`
- Modify: `kid-workbench/shared-go/mastery/skills_test.go`

- [ ] **Step 1: Write failing module-skill tests**

Add table tests that assert exact ordered results and defensive copies:

```go
func TestSkillsForMathModules(t *testing.T) {
    require.Equal(t, []string{"calc", "story"}, mastery.SkillsFor("math", "add10"))
    require.Equal(t, []string{"calc", "story"}, mastery.SkillsFor("math", "sub10"))
    require.Equal(t, []string{"find", "name"}, mastery.SkillsFor("math", "shape"))
    require.Empty(t, mastery.SkillsFor("math", "unknown"))
}

func TestSkillsForReturnsCopy(t *testing.T) {
    got := mastery.SkillsFor("math", "add10")
    got[0] = "changed"
    require.Equal(t, []string{"calc", "story"}, mastery.SkillsFor("math", "add10"))
}

func TestMathQuestionCodesMapToSkills(t *testing.T) {
    for _, code := range []string{"calc", "story", "find", "name"} {
        require.Equal(t, code, mastery.SkillFromQuestionCode(code))
    }
}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `cd kid-workbench/shared-go && go test ./mastery -run 'TestSkillsForMathModules|TestSkillsForReturnsCopy|TestMathQuestionCodesMapToSkills' -count=1`

Expected: FAIL because `SkillsFor` and math skill constants do not exist.

- [ ] **Step 3: Implement the focused skill API**

Add constants and immutable ordered slices:

```go
const (
    SkillMathCalc  = "calc"
    SkillMathStory = "story"
    SkillMathFind  = "find"
    SkillMathName  = "name"
)

var MathArithmeticSkills = []string{SkillMathCalc, SkillMathStory}
var MathShapeSkills = []string{SkillMathFind, SkillMathName}

func SkillsFor(subjectCode, moduleCode string) []string {
    switch subjectCode {
    case "literacy":
        return append([]string{}, LiteracySkills...)
    case "pinyin":
        return append([]string{}, PinyinSkills...)
    case "math":
        switch moduleCode {
        case "add10", "sub10":
            return append([]string{}, MathArithmeticSkills...)
        case "shape":
            return append([]string{}, MathShapeSkills...)
        }
    }
    return nil
}

func SkillsForSubject(subjectCode string) []string {
    return SkillsFor(subjectCode, "")
}
```

Preserve `SkillsForSubject` for current callers. Extend `SkillFromQuestionCode` with the four math constants.

- [ ] **Step 4: Run mastery tests**

Run: `cd kid-workbench/shared-go && go test ./mastery -count=1`

Expected: PASS, including existing literacy and pinyin behavior.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/shared-go/mastery
git commit -m "feat: add module-aware math mastery skills"
```

### Task 2: Apply Module Skills in the Atomic Learning Service

**Files:**
- Modify: `kid-workbench/shared-go/learning/service.go`
- Modify: `kid-workbench/shared-go/learning/service_test.go`

- [ ] **Step 1: Add a failing two-skill rollup test**

Extend the test database modules table to include `code`, seed a math subject/module/KP with `calc` and `story` questions, then assert the first skill alone leaves KP mastery `learning`:

```go
func TestMathKnowledgePointRequiresBothModuleSkills(t *testing.T) {
    gdb := newMathLearningDB(t, "add10")
    svc := learning.NewService(mastery.Config{BaseMasterStreak: 1, MinAccuracy: 0.8})
    at := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

    applyCorrect(t, gdb, svc, 1, 10, 20, "calc-1", at)
    requireSkillStatus(t, gdb, 1, 10, "calc", "mastered")
    requireSkillStatus(t, gdb, 1, 10, "story", "not_started")
    requireKpStatus(t, gdb, 1, 10, "learning")

    applyCorrect(t, gdb, svc, 1, 10, 21, "story-1", at.Add(time.Minute))
    requireKpStatus(t, gdb, 1, 10, "mastered")
}
```

Add the equivalent `find`/`name` test for module `shape`.

- [ ] **Step 2: Run tests and verify failure**

Run: `cd kid-workbench/shared-go && go test ./learning -run 'TestMathKnowledgePointRequiresBothModuleSkills|TestMathShapeRequiresFindAndName' -count=1`

Expected: FAIL because the learning service only queries subject code and therefore treats math as single-row mastery.

- [ ] **Step 3: Query subject and module together**

Replace `subjectCodeForKp` with:

```go
type kpScope struct {
    SubjectCode string
    ModuleCode  string
}

func scopeForKp(tx *gorm.DB, kpID int64) (kpScope, error) {
    var scope kpScope
    err := tx.Raw(`
        SELECT s.code AS subject_code, m.code AS module_code
        FROM subjects s
        JOIN modules m ON m.subject_id = s.id
        JOIN knowledge_points kp ON kp.module_id = m.id
        WHERE kp.id = ?`, kpID).Scan(&scope).Error
    return scope, err
}
```

In `ApplyOne`, call:

```go
scope, err := scopeForKp(tx, in.KpID)
if err != nil { return StateDTO{}, false, err }
skillSet := mastery.SkillsFor(scope.SubjectCode, scope.ModuleCode)
```

- [ ] **Step 4: Run shared and parent tests**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Expected: both PASS.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/shared-go/learning
git commit -m "feat: roll up mastery by subject module"
```

### Task 3: Add Math Plan Metadata and Immutable Question Snapshots

**Files:**
- Create: `kid-workbench/parent-dashboard/backend/internal/db/migrations/postgres/012_math_plan_snapshot.sql`
- Create: `kid-workbench/parent-dashboard/backend/internal/db/migrations/sqlite/012_math_plan_snapshot.sql`
- Modify: `kid-workbench/parent-dashboard/backend/internal/db/db_test.go`
- Modify: `kid-workbench/shared-go/model/model.go`
- Modify: `kid-workbench/shared-go/model/model_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/model/model.go`

- [ ] **Step 1: Write failing migration and model tests**

Add migration assertions:

```go
for _, tc := range []struct{ table, column string }{
    {"plan_items", "question_snapshot"},
    {"study_plans", "plan_kind"},
    {"study_plans", "module_code"},
    {"study_plans", "stage_code"},
} {
    require.True(t, gdb.Migrator().HasColumn(tc.table, tc.column), "%s.%s", tc.table, tc.column)
}
```

Add a shared model test that creates and reloads a `PlanItem{QuestionSnapshot: ...}` without losing JSON text.

- [ ] **Step 2: Run tests and verify failure**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/db -run TestMigrateCreatesMathPlanColumns -count=1`

Run: `cd kid-workbench/shared-go && go test ./model -run TestPlanItemPersistsQuestionSnapshot -count=1`

Expected: FAIL because columns and fields do not exist.

- [ ] **Step 3: Add paired forward-only migrations**

PostgreSQL:

```sql
ALTER TABLE plan_items
ADD COLUMN IF NOT EXISTS question_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE study_plans
ADD COLUMN IF NOT EXISTS plan_kind VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE study_plans
ADD COLUMN IF NOT EXISTS module_code VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE study_plans
ADD COLUMN IF NOT EXISTS stage_code VARCHAR(32) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_study_plans_math_scope
ON study_plans(child_id, subject_code, plan_kind, module_code, stage_code);
```

SQLite:

```sql
ALTER TABLE plan_items ADD COLUMN question_snapshot TEXT NOT NULL DEFAULT '{}';
ALTER TABLE study_plans ADD COLUMN plan_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE study_plans ADD COLUMN module_code TEXT NOT NULL DEFAULT '';
ALTER TABLE study_plans ADD COLUMN stage_code TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_study_plans_math_scope
ON study_plans(child_id, subject_code, plan_kind, module_code, stage_code);
```

- [ ] **Step 4: Add model fields**

Add to `shared-go/model.PlanItem`:

```go
QuestionSnapshot string `gorm:"column:question_snapshot;type:json"`
```

Add to parent `model.StudyPlan`:

```go
PlanKind   string `gorm:"column:plan_kind"`
ModuleCode string `gorm:"column:module_code"`
StageCode  string `gorm:"column:stage_code"`
```

- [ ] **Step 5: Verify both dialects and existing behavior**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/db ./internal/model ./internal/service -count=1`

Expected: PASS and migration 012 is idempotent.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/shared-go/model kid-workbench/parent-dashboard/backend/internal/db kid-workbench/parent-dashboard/backend/internal/model/model.go
git commit -m "feat: add immutable math plan snapshots"
```

### Task 4: Generate Question-Level Math TTS in Content Admin

**Files:**
- Modify: `kid-workbench/content-admin/backend/internal/storage/minio.go`
- Modify: `kid-workbench/content-admin/backend/internal/math/service.go`
- Modify: `kid-workbench/content-admin/backend/internal/math/service_test.go`

- [ ] **Step 1: Write failing key and generation tests**

```go
func TestMathQuestionSpeechObjectKey(t *testing.T) {
    require.Equal(t, "math/questions/42.mp3", storage.MathQuestionSpeechObjectKey(42))
}

func TestGenerateQuestionSpeechUsesQuestionSpeechJSON(t *testing.T) {
    svc, fakeStore, fakeTTS := setupMathQuestionSpeechService(t)
    dto, err := svc.RegenerateQuestionSpeech(context.Background(), 42)
    require.NoError(t, err)
    require.Equal(t, "二加五等于几", fakeTTS.LastText)
    require.Equal(t, "math/questions/42.mp3", fakeStore.LastKey)
    require.Equal(t, "math/questions/42.mp3", dto.MediaURL)
}
```

Seed a non-math question and assert the method returns `gorm.ErrRecordNotFound`.

- [ ] **Step 2: Run tests and verify failure**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/math ./internal/storage -run 'TestMathQuestionSpeechObjectKey|TestGenerateQuestionSpeech' -count=1`

Expected: FAIL because the key and question-level methods do not exist.

- [ ] **Step 3: Add the stable object key and DTOs**

```go
func MathQuestionSpeechObjectKey(questionID int64) string {
    return fmt.Sprintf("math/questions/%d.mp3", questionID)
}

type QuestionSpeechDTO struct {
    QuestionID int64  `json:"questionId"`
    KpID       int64  `json:"kpId"`
    Code       string `json:"code"`
    SpeechText string `json:"speechText"`
    MediaURL   string `json:"mediaUrl"`
}

type QuestionSpeechStatus struct {
    ModuleCode string `json:"moduleCode"`
    Total      int    `json:"total"`
    Ready      int    `json:"ready"`
    Missing    int    `json:"missing"`
}
```

Parse `questions.speech` using `struct { Text string 'json:"text"' }`. Query through `questions -> knowledge_points -> modules -> subjects` with `subjects.code = 'math'`.

- [ ] **Step 4: Implement single and batch generation**

`RegenerateQuestionSpeech(ctx, questionID)` must synthesize `questions.speech.text`, store bytes at `math/questions/<questionID>.mp3`, and persist the returned full object key in `questions.media_url`.

`BatchGenerateQuestionSpeech(ctx, moduleCode)` must select only math questions in the module, skip only rows whose `media_url` matches `math/questions/%.mp3`, and regenerate empty, legacy URL, or other-key values. Process in deterministic KP/question order and return the existing `BatchResult` counts. `QuestionSpeechStatus(moduleCode)` counts ready only when `media_url` matches that stable key prefix; it returns total/ready/missing without reading MinIO.

- [ ] **Step 5: Run backend tests**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/math ./internal/storage ./internal/tts -count=1`

Expected: PASS; existing KP-level TTS remains unchanged for admin previews.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/content-admin/backend/internal/storage kid-workbench/content-admin/backend/internal/math
git commit -m "feat: generate math question audio"
```

### Task 5: Expose Question Audio Publishing in Content Admin

**Files:**
- Modify: `kid-workbench/content-admin/backend/internal/http/handler_math.go`
- Modify: `kid-workbench/content-admin/backend/internal/http/router.go`
- Modify: `kid-workbench/content-admin/backend/internal/http/router_test.go`
- Modify: `kid-workbench/content-admin/frontend/src/api/math.ts`
- Modify: `kid-workbench/content-admin/frontend/src/api/mathTypes.ts`
- Modify: `kid-workbench/content-admin/frontend/src/features/math/MathPage.tsx`

- [ ] **Step 1: Write failing HTTP contract tests**

Assert these routes are registered and validate `moduleCode`/`questionId`:

```http
GET  /api/v1/math/question-speech/status?moduleCode=add10
POST /api/v1/math/question-speech/batch?moduleCode=add10
POST /api/v1/math/questions/:questionId/speech
```

Expected response types are `QuestionSpeechStatus`, `BatchResult`, and `QuestionSpeechDTO`.

- [ ] **Step 2: Run and verify failure**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/http -run TestMathQuestionSpeechRoutes -count=1`

Expected: FAIL with 404 because routes are absent.

- [ ] **Step 3: Implement handlers and routes**

Use existing math handler error mapping. Empty `moduleCode` returns 400. Unknown/non-math question returns 404. TTS/config/storage failures return 503 using the same classification as current math speech endpoints.

- [ ] **Step 4: Add typed frontend calls**

```ts
export interface MathQuestionSpeechStatus {
  moduleCode: string
  total: number
  ready: number
  missing: number
}

export async function getQuestionSpeechStatus(moduleCode: string) {
  const query = new URLSearchParams({ moduleCode })
  return parseJSON<MathQuestionSpeechStatus>(await fetch(`/api/v1/math/question-speech/status?${query}`))
}

export async function batchGenerateQuestionSpeech(moduleCode: string) {
  const query = new URLSearchParams({ moduleCode })
  return parseJSON<MathBatchResult>(await fetch(`/api/v1/math/question-speech/batch?${query}`, { method: 'POST' }))
}
```

- [ ] **Step 5: Add per-module publishing controls**

In each math group header, show `题目音频 ready/total` and a button labelled `生成本组题目音频`. Invalidate `['math', 'question-speech']` after generation. Keep existing KP-level “读音” controls because they serve admin asset previews.

- [ ] **Step 6: Run all content-admin checks**

Run: `cd kid-workbench/content-admin/backend && go test ./... -count=1`

Run: `cd kid-workbench/content-admin/frontend && npm run build`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add kid-workbench/content-admin/backend/internal/http kid-workbench/content-admin/frontend/src/api kid-workbench/content-admin/frontend/src/features/math/MathPage.tsx
git commit -m "feat: publish math question audio"
```

### Task 6: Verify Foundation Contracts End to End

**Files:**
- Modify: `kid-workbench/content-admin/README.md`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Run foundation verification**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Run: `cd kid-workbench/content-admin/backend && go test ./... -count=1`

Run: `cd kid-workbench/content-admin/frontend && npm run build`

Expected: all PASS.

- [ ] **Step 2: Verify database contract**

Run migrations against a disposable PostgreSQL database, seed questions, batch-generate one math module with fake TTS/storage adapters, and assert:

```sql
SELECT COUNT(*) FROM questions q
JOIN knowledge_points kp ON kp.id=q.kp_id
JOIN modules m ON m.id=kp.module_id
JOIN subjects s ON s.id=m.subject_id
WHERE s.code='math' AND m.code='add10' AND q.media_url LIKE 'math/questions/%.mp3';
```

Expected: count equals the number of published `add10` question variants.

- [ ] **Step 3: Document the ownership boundary**

Document that `math_assets.speech_audio_url` remains a KP/admin-preview asset while `questions.media_url` is the mandatory child-practice audio object key. Document migration 012 and module-aware skill rules.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/content-admin/README.md kid-workbench/README.md
git commit -m "docs: document math learning foundations"
```
