# Science Content Publishing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make science content explicitly reviewable and publishable, expose child-safe presentation fields, and replace duplicate `fact1`/`fact2` rows with one real `recognize` question per knowledge point.

**Architecture:** `content-admin` remains the only writer of `science_assets` presentation and publication fields. Published questions remain in the shared `questions` table; `science-server` will later join them to `science_assets.review_status = 'published'`.

**Tech Stack:** Go 1.26, Gin, GORM, PostgreSQL, SQLite tests, React 19, TypeScript, TanStack Query

---

### Task 1: Add the Science Publication Schema

**Files:**
- Modify: `kid-workbench/content-admin/backend/internal/db/db.go`
- Modify: `kid-workbench/content-admin/backend/internal/science/service.go`
- Create: `kid-workbench/content-admin/backend/internal/science/publication_test.go`

- [ ] **Step 1: Write the failing migration test**

Create a SQLite database with the old `science_assets` columns, call `db.Migrate`, and assert the new columns exist and the existing row becomes published:

```go
func TestMigrateAddsSciencePublicationColumns(t *testing.T) {
    gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
    require.NoError(t, err)
    require.NoError(t, gdb.Exec(`CREATE TABLE science_assets (
      kp_id INTEGER PRIMARY KEY, title TEXT NOT NULL,
      needs_sense_image INTEGER NOT NULL,
      module_code TEXT NOT NULL DEFAULT '', module_name TEXT NOT NULL DEFAULT '',
      module_order INTEGER NOT NULL DEFAULT 0, kp_order INTEGER NOT NULL DEFAULT 0,
      needs_sense_image_override INTEGER,
      glyph_image_url TEXT NOT NULL DEFAULT '', sense_image_url TEXT NOT NULL DEFAULT '',
      speech_audio_url TEXT NOT NULL DEFAULT '',
      synced_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
    )`).Error)
    require.NoError(t, gdb.Exec(`INSERT INTO science_assets
      (kp_id,title,needs_sense_image,synced_at,updated_at)
      VALUES (1,'冬眠',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`).Error)

    require.NoError(t, db.Migrate(gdb))

    var got struct {
        ReviewStatus string
        ContentVersion int
    }
    require.NoError(t, gdb.Table("science_assets").First(&got, "kp_id = 1").Error)
    require.Equal(t, "published", got.ReviewStatus)
    require.Equal(t, 1, got.ContentVersion)
}
```

- [ ] **Step 2: Run it to verify failure**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/science -run TestMigrateAddsSciencePublicationColumns -count=1`

Expected: FAIL because the columns do not exist.

- [ ] **Step 3: Add backward-compatible columns**

After creating `science_assets`, add dialect-specific idempotent migrations for:

```sql
summary TEXT NOT NULL DEFAULT ''
explanation TEXT NOT NULL DEFAULT ''
fun_fact TEXT NOT NULL DEFAULT ''
review_status VARCHAR(16) NOT NULL DEFAULT 'published'
reviewed_at TIMESTAMPTZ
content_version INT NOT NULL DEFAULT 1
```

For SQLite, inspect `PRAGMA table_info(science_assets)` before each `ALTER TABLE`; do not ignore arbitrary migration errors. Extend `science.Asset` with matching GORM and JSON fields:

```go
Summary        string     `gorm:"column:summary" json:"summary"`
Explanation    string     `gorm:"column:explanation" json:"explanation"`
FunFact        string     `gorm:"column:fun_fact" json:"funFact"`
ReviewStatus   string     `gorm:"column:review_status" json:"reviewStatus"`
ReviewedAt     *time.Time `gorm:"column:reviewed_at" json:"reviewedAt"`
ContentVersion int        `gorm:"column:content_version" json:"contentVersion"`
```

- [ ] **Step 4: Run migration tests**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/db ./internal/science -count=1`

Expected: PASS for a fresh schema and an upgraded old schema.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/content-admin/backend/internal/db/db.go kid-workbench/content-admin/backend/internal/science
git commit -m "feat: add science publication metadata"
```

### Task 2: Parse Presentation Content During Science Sync

**Files:**
- Create: `kid-workbench/content-admin/backend/internal/science/content.go`
- Create: `kid-workbench/content-admin/backend/internal/science/content_test.go`
- Modify: `kid-workbench/content-admin/backend/internal/science/service.go`

- [ ] **Step 1: Write parser and sync-preservation tests**

Cover both the target and legacy payloads:

```go
func TestParseContentSupportsTargetPayload(t *testing.T) {
    got := ParseContent(`{"summary":"过冬的方法","explanation":"节省能量","funFact":"心跳变慢"}`)
    require.Equal(t, "过冬的方法", got.Summary)
    require.Equal(t, "节省能量", got.Explanation)
    require.Equal(t, "心跳变慢", got.FunFact)
}

func TestParseContentAcceptsLegacyFactPayload(t *testing.T) {
    got := ParseContent(`{"kind":"fact","q":"谁会冬眠？","a":"熊"}`)
    require.Empty(t, got.Summary)
    require.Empty(t, got.Explanation)
}
```

Add a service test that syncs an already reviewed row and verifies sync updates catalog fields without resetting `review_status`, `reviewed_at`, or `content_version`.

- [ ] **Step 2: Run tests to verify failure**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/science -run 'TestParseContent|TestSyncPreservesPublication' -count=1`

Expected: FAIL because `ParseContent` and publication-preserving sync are absent.

- [ ] **Step 3: Implement the parser**

```go
type Presentation struct {
    Summary     string `json:"summary"`
    Explanation string `json:"explanation"`
    FunFact     string `json:"funFact"`
}

func ParseContent(raw string) Presentation {
    var out Presentation
    if json.Unmarshal([]byte(raw), &out) != nil {
        return Presentation{}
    }
    out.Summary = strings.TrimSpace(out.Summary)
    out.Explanation = strings.TrimSpace(out.Explanation)
    out.FunFact = strings.TrimSpace(out.FunFact)
    return out
}
```

Select `kp.payload` in `Sync`. New assets must set `ReviewStatus: "draft"` and `ContentVersion: 1`; existing assets keep publication fields. Only increment `content_version` when presentation content changes, and reset changed published content to `draft` so it must be reviewed again.

- [ ] **Step 4: Verify sync behavior**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/science -count=1`

Expected: PASS including new, unchanged, and changed published records.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/content-admin/backend/internal/science
git commit -m "feat: sync science presentation content"
```

### Task 3: Add Explicit Review and Publish APIs

**Files:**
- Modify: `kid-workbench/content-admin/backend/internal/science/service.go`
- Modify: `kid-workbench/content-admin/backend/internal/http/handler_science.go`
- Modify: `kid-workbench/content-admin/backend/internal/http/router.go`
- Modify: `kid-workbench/content-admin/backend/internal/http/router_test.go`

- [ ] **Step 1: Write route contract tests**

Test these transitions:

```text
draft --review--> reviewed --publish--> published
published --content edit--> draft
```

Verify direct `draft -> published` returns `409 review_required`, and publishing without at least one question returns `409 no_published_question`.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/http -run TestSciencePublication -count=1`

Expected: FAIL with 404 routes.

- [ ] **Step 3: Implement service transitions**

Add focused methods:

```go
type ContentPatch struct {
    Summary     string `json:"summary"`
    Explanation string `json:"explanation"`
    FunFact     string `json:"funFact"`
}

func (s *Service) PatchContent(kpID int64, in ContentPatch) (ItemDTO, error)
func (s *Service) Review(kpID int64, at time.Time) (ItemDTO, error)
func (s *Service) Publish(kpID int64) (ItemDTO, error)
func (s *Service) Unpublish(kpID int64) (ItemDTO, error)
```

`PatchContent` trims fields, increments `content_version` only on change, clears `reviewed_at`, and sets `review_status = 'draft'`. `Review` sets `reviewed`; `Publish` requires `reviewed` and a `questions.code = 'recognize'` row for the KP; `Unpublish` returns to `reviewed`.

- [ ] **Step 4: Mount the routes**

```http
PATCH /api/v1/science/items/:kpId/content
POST  /api/v1/science/items/:kpId/review
POST  /api/v1/science/items/:kpId/publish
POST  /api/v1/science/items/:kpId/unpublish
```

Return stable JSON errors with `code` and `error`; do not return raw SQL errors.

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/content-admin/backend && go test ./internal/science ./internal/http -count=1`

```bash
git add kid-workbench/content-admin/backend/internal/science kid-workbench/content-admin/backend/internal/http
git commit -m "feat: review and publish science content"
```

### Task 4: Normalize Science Questions to One Recognize Variant

**Files:**
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/science.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/quiz_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/seed/questions.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/seed/questions_test.go`
- Modify: `kid-workbench/shared-go/mastery/skills.go`
- Modify: `kid-workbench/shared-go/mastery/skills_test.go`

- [ ] **Step 1: Replace the duplicate-question regression test**

```go
func TestScienceProducesOneRecognizeQuestion(t *testing.T) {
    kp := Kp{ID: 900, Title: "冬眠", SubjectCode: "science", ModuleCode: "animal",
        Payload: `{"kind":"fact","q":"谁会冬眠？","a":"熊","wrong":["燕子","蝴蝶","鸭子"]}`}
    specs := Generate(kp)
    require.Len(t, specs, 1)
    require.Equal(t, "recognize", specs[0].Code)
}
```

Add a seed test asserting legacy `fact1` and `fact2` rows are removed for science KPs after reseeding.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz ./internal/seed -run Science -count=1`

Expected: FAIL because two variants are still generated.

- [ ] **Step 3: Generate only `recognize`**

Return one deterministic `Spec{Code: "recognize"}` from `scienceSpecs`. In `seed.Questions`, delete obsolete science `fact1`/`fact2` rows inside the same transaction before upserting generated questions; never delete non-science rows.

Register the science skill in the shared module:

```go
const SkillScienceRecognize = "recognize"
var ScienceSkills = []string{SkillScienceRecognize}
```

Make `SkillsForSubject("science")` return `ScienceSkills` and map `recognize` in `SkillFromQuestionCode`.

- [ ] **Step 4: Run all affected tests**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz ./internal/seed ./internal/service -count=1`

Expected: PASS, with exactly one science question per KP.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/shared-go/mastery kid-workbench/parent-dashboard/backend/internal/quiz kid-workbench/parent-dashboard/backend/internal/seed
git commit -m "feat: normalize science recognition questions"
```

### Task 5: Add Publication Controls to Content Admin

**Files:**
- Modify: `kid-workbench/content-admin/frontend/src/api/scienceTypes.ts`
- Modify: `kid-workbench/content-admin/frontend/src/api/science.ts`
- Modify: `kid-workbench/content-admin/frontend/src/features/science/SciencePage.tsx`
- Create: `kid-workbench/content-admin/frontend/src/features/science/SciencePage.test.tsx`
- Modify: `kid-workbench/content-admin/frontend/package.json`

- [ ] **Step 1: Add the frontend test harness and failing tests**

Add Vitest, jsdom, Testing Library and a `test` script. Test that a draft card shows “待审核”, a reviewed card allows publish, and changing explanation resets a published card to draft.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/content-admin/frontend && npm test -- SciencePage.test.tsx`

Expected: FAIL because the controls and API functions do not exist.

- [ ] **Step 3: Extend API contracts**

```ts
export type ScienceReviewStatus = 'draft' | 'reviewed' | 'published'

export interface ScienceItem {
  // existing fields stay unchanged
  summary: string
  explanation: string
  funFact: string
  reviewStatus: ScienceReviewStatus
  reviewedAt: string | null
  contentVersion: number
}
```

Add `patchScienceContent`, `reviewScienceItem`, `publishScienceItem`, and `unpublishScienceItem` functions using the routes from Task 3.

- [ ] **Step 4: Implement focused controls**

Keep asset generation controls unchanged. Add a detail editor for summary, explanation and fun fact; display status and version; show only valid transition actions. On mutation success invalidate `['science']` queries and render server error codes as actionable Chinese messages.

- [ ] **Step 5: Verify frontend and backend**

Run: `cd kid-workbench/content-admin/frontend && npm test && npm run build`

Run: `cd kid-workbench/content-admin/backend && go test ./... -count=1`

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/content-admin/frontend
git commit -m "feat: manage science publication status"
```

### Task 6: Content Publication Acceptance

**Files:**
- Create: `kid-workbench/content-admin/backend/scripts/verify_science_publication.sql`
- Modify: `kid-workbench/content-admin/README.md`

- [ ] **Step 1: Add a bounded verification query**

The script must return counts only and perform no writes:

```sql
SELECT review_status, COUNT(*)
FROM science_assets
GROUP BY review_status
ORDER BY review_status;

SELECT COUNT(*) AS published_without_question
FROM science_assets sa
WHERE sa.review_status = 'published'
  AND NOT EXISTS (SELECT 1 FROM questions q WHERE q.kp_id = sa.kp_id);
```

- [ ] **Step 2: Document the workflow**

Document sync → edit → review → publish, the migration behavior for existing content, and the rule that new synced content starts as draft.

- [ ] **Step 3: Run final checks**

Run: `cd kid-workbench/content-admin/backend && go test ./... -count=1`

Run: `cd kid-workbench/content-admin/frontend && npm test && npm run build`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz ./internal/seed -count=1`

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/content-admin/README.md kid-workbench/content-admin/backend/scripts/verify_science_publication.sql
git commit -m "docs: verify science publication workflow"
```
