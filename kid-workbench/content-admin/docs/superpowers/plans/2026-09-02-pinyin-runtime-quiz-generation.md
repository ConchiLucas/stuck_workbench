# Pinyin Runtime Quiz Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate a fresh database-backed preview question whenever an administrator enters one of the four pinyin quiz types.

**Architecture:** Add curated blend syllables to the shared database and a generator inside the content-admin pinyin service. Expose one POST endpoint that returns a complete immutable preview payload. The React page requests that endpoint on every card entry and sends recently used target IDs to reduce repetition.

**Tech Stack:** Go, Gin, GORM, PostgreSQL/SQLite, React, TypeScript, TanStack Query, Vitest.

---

### Task 1: Database-backed generator

**Files:**
- Modify: `backend/internal/db/db.go`
- Create: `backend/internal/pinyin/quiz.go`
- Create: `backend/internal/pinyin/quiz_test.go`

- [ ] Write tests for listen, inword, shape, blend, unique four-option output, and excluded targets.
- [ ] Run `go test ./internal/pinyin -run Quiz -count=1` and confirm the missing generator fails.
- [ ] Add `pinyin_syllable_assets`, seed valid syllables idempotently, and implement `GenerateQuiz`.
- [ ] Run the focused tests and confirm they pass.

### Task 2: HTTP preview endpoint

**Files:**
- Modify: `backend/internal/http/handler_pinyin.go`
- Modify: `backend/internal/http/router.go`
- Modify: `backend/internal/http/router_test.go`

- [ ] Write a failing route test for `POST /api/v1/pinyin/quiz/generate`.
- [ ] Parse `type` and `excludeTargetIds`, validate the type, and call the pinyin generator.
- [ ] Run `go test ./internal/http -count=1` and confirm the endpoint contract passes.

### Task 3: React runtime preview

**Files:**
- Modify: `frontend/src/api/pinyin.ts`
- Modify: `frontend/src/api/pinyinTypes.ts`
- Modify: `frontend/src/features/pinyin/PinyinPage.tsx`
- Modify: `frontend/src/features/pinyin/PinyinPage.test.tsx`
- Modify: `frontend/src/styles/global.css`

- [ ] Write a failing test proving each card makes a generation request on entry and a second entry requests a new question with exclusions.
- [ ] Add API payload types and the generation call.
- [ ] Replace local real examples and fixed mocks with one renderer driven by the returned payload.
- [ ] Keep shape/blend options audio-only and add loading/error/retry states.
- [ ] Run the focused test and confirm it passes.

### Task 4: Verification and local deployment

**Files:**
- Verify all files above; do not modify unrelated dirty-worktree files.

- [ ] Run `go test ./...` in `content-admin/backend`.
- [ ] Run `npm test && npm run build` in `content-admin/frontend`.
- [ ] Rebuild the standalone content-admin container.
- [ ] Use the live page to enter all four types twice and verify changed targets, valid controls, and hidden shape/blend labels.
