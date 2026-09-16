# Shared Plan Option Order Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one shared, backward-compatible persisted option-order contract for every subject-specific learning server.

**Architecture:** The shared `plan_items` table owns display order because plans must survive refreshes and question edits. Subject servers read and write the same column; no subject creates a private alternative migration.

**Tech Stack:** Go 1.26, GORM, PostgreSQL, SQLite

---

### Task 1: Add and Verify the Shared Option Order

**Files:**
- Create: `kid-workbench/parent-dashboard/backend/internal/db/migrations/postgres/007_plan_option_order.sql`
- Create: `kid-workbench/parent-dashboard/backend/internal/db/migrations/sqlite/007_plan_option_order.sql`
- Modify: `kid-workbench/shared-go/model/model.go`
- Modify: `kid-workbench/shared-go/model/model_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/service/plan_test.go`

- [ ] **Step 1: Write the failing model and legacy-plan tests**

Assert `model.PlanItem` maps `OptionOrder` to `option_order`, and loading an existing plan with an empty value preserves the current identity ordering.

```go
func TestPlanItemHasPersistedOptionOrder(t *testing.T) {
    got := model.PlanItem{OptionOrder: "2,0,3,1"}
    require.Equal(t, "2,0,3,1", got.OptionOrder)
    require.Equal(t, "plan_items", got.TableName())
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/shared-go && go test ./model -run TestPlanItemHasPersistedOptionOrder -count=1`

Expected: FAIL because `OptionOrder` is absent.

- [ ] **Step 3: Add the migrations**

PostgreSQL:

```sql
ALTER TABLE plan_items
ADD COLUMN IF NOT EXISTS option_order VARCHAR(32) NOT NULL DEFAULT '';
```

SQLite:

```sql
ALTER TABLE plan_items
ADD COLUMN option_order TEXT NOT NULL DEFAULT '';
```

The migration runner applies numbered files once, so the SQLite statement is not wrapped in ignored errors.

- [ ] **Step 4: Add the shared model field**

```go
type PlanItem struct {
    // existing fields remain unchanged
    OptionOrder string `gorm:"column:option_order"`
}
```

Existing parent plans leave it empty. Subject servers interpret empty as identity order `0,1,2,3` and persist the value on their first read.

- [ ] **Step 5: Run all schema consumers**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/db ./internal/service -count=1`

Expected: PASS for new and migrated databases.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/shared-go/model kid-workbench/parent-dashboard/backend/internal/db/migrations kid-workbench/parent-dashboard/backend/internal/service/plan_test.go
git commit -m "feat: persist shared plan option order"
```
