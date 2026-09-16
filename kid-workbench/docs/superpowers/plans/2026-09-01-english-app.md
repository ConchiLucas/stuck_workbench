# English App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an independent iPad-first React/PWA where a child can browse English topics, recognize words, complete listening practice, resume unfinished work, and view a simple result.

**Architecture:** `english-app` calls only `english-server` through `/api`. React Query owns server state, a small Zustand store owns the local child ID, a single audio controller enforces one active clip, and practice state preserves one `clientId` per pending answer until acknowledgement.

**Tech Stack:** React 19, TypeScript 6, Vite 8, React Router 7, TanStack Query 5, Zustand 5, Vitest, Testing Library, Playwright, Nginx, Docker

**Prerequisite:** Complete `2026-09-01-english-server.md` through its stable API and DTO tasks.

---

### Task 1: Scaffold the Independent App and Test Harness

**Files:**
- Create: `kid-workbench/english-app/package.json`
- Create: `kid-workbench/english-app/package-lock.json`
- Create: `kid-workbench/english-app/index.html`
- Create: `kid-workbench/english-app/tsconfig.json`
- Create: `kid-workbench/english-app/tsconfig.app.json`
- Create: `kid-workbench/english-app/tsconfig.node.json`
- Create: `kid-workbench/english-app/vite.config.ts`
- Create: `kid-workbench/english-app/src/main.tsx`
- Create: `kid-workbench/english-app/src/App.tsx`
- Create: `kid-workbench/english-app/src/App.test.tsx`
- Create: `kid-workbench/english-app/src/test/setup.ts`
- Create: `kid-workbench/english-app/src/styles/tokens.css`
- Create: `kid-workbench/english-app/src/styles/app.css`

- [ ] **Step 1: Create package scripts and dependencies**

Use production dependencies matching `literacy-app` except `hanzi-writer`. Add `vitest`, `jsdom`, `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, and `@playwright/test` as development dependencies.

```json
{
  "scripts": {
    "dev": "vite --host",
    "build": "tsc -b && vite build",
    "test": "vitest run",
    "e2e": "playwright test"
  }
}
```

- [ ] **Step 2: Write a failing route-shell test**

Assert `/` renders “今天学英语”, `/words` renders “单词地图”, and an unknown route returns to `/`.

- [ ] **Step 3: Run to verify failure**

Run: `cd kid-workbench/english-app && npm install && npm test -- --run src/App.test.tsx`

Expected: FAIL because routes and pages do not exist.

- [ ] **Step 4: Implement the minimal app shell**

Configure `QueryClientProvider`, `BrowserRouter`, safe-area layout, and lazy page boundaries. Set Vite port `19122` and proxy `/api` to `VITE_API_PROXY ?? 'http://localhost:19121'`.

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/english-app && npm test -- --run src/App.test.tsx && npm run build`

```bash
git add kid-workbench/english-app
git commit -m "feat: scaffold english learning app"
```

### Task 2: Add Typed API Contracts and Stable Client IDs

**Files:**
- Create: `kid-workbench/english-app/src/api/client.ts`
- Create: `kid-workbench/english-app/src/api/types.ts`
- Create: `kid-workbench/english-app/src/api/catalog.ts`
- Create: `kid-workbench/english-app/src/api/home.ts`
- Create: `kid-workbench/english-app/src/api/plans.ts`
- Create: `kid-workbench/english-app/src/api/progress.ts`
- Create: `kid-workbench/english-app/src/api/client.test.ts`
- Create: `kid-workbench/english-app/src/store/childStore.ts`
- Create: `kid-workbench/english-app/src/store/pendingAnswerStore.ts`
- Create: `kid-workbench/english-app/src/store/pendingAnswerStore.test.ts`

- [ ] **Step 1: Write envelope and client-ID tests**

Assert successful `{data,error:null}` unwrapping, stable error-code parsing, default child ID `1`, one UUID per `planId:itemId:try`, and removal only after explicit acknowledgement.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- --run src/api src/store`

Expected: FAIL because API and stores are undefined.

- [ ] **Step 3: Define exact practice DTOs**

```ts
export type MasteryStatus = 'not_started' | 'learning' | 'shaky' | 'review_due' | 'mastered'
export type QuestionOption =
  | { kpId: number; label: string }
  | { kpId: number; senseUrl: string }

export interface PlanQuestion {
  id: number
  code: 'listen' | 'picture'
  stem: string
  speechUrl: string
  options: QuestionOption[]
}

export interface AnswerResult {
  correct: boolean
  canRetry: boolean
  correctOptionIndex?: number
  tries: number
  itemStatus: 'pending' | 'correct' | 'wrong'
  planProgress: { done: number; total: number }
}
```

Do not include an answer field in pre-submit DTOs.

- [ ] **Step 4: Implement React Query hooks**

Use query keys containing `childId` and subject. Disable window-focus refetch for plan detail so option order and the current screen do not flash.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-app && npm test -- --run src/api src/store`

```bash
git add kid-workbench/english-app/src/api kid-workbench/english-app/src/store
git commit -m "feat: add english app data contracts"
```

### Task 3: Build the English Home Page

**Files:**
- Create: `kid-workbench/english-app/src/pages/HomePage.tsx`
- Create: `kid-workbench/english-app/src/pages/HomePage.test.tsx`
- Create: `kid-workbench/english-app/src/components/AppShell.tsx`
- Create: `kid-workbench/english-app/src/components/TaskCard.tsx`
- Create: `kid-workbench/english-app/src/components/StatPill.tsx`
- Modify: `kid-workbench/english-app/src/App.tsx`

- [ ] **Step 1: Write loading, error and action tests**

Assert loading skeleton; child-friendly retry; active plan shows “继续练习”; no plan shows “开始练习”; and exactly two primary entries exist: practice and word map.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- --run src/pages/HomePage.test.tsx`

Expected: FAIL because HomePage is undefined.

- [ ] **Step 3: Implement the iPad-first home layout**

Display child name, active/todo English task, review count, flowers, and two primary actions. Use minimum `56px` touch targets, safe-area padding, no hover-only actions, and no mastery percentages.

- [ ] **Step 4: Wire plan creation and resume**

Unlock audio from the same child gesture that starts/resumes practice. Navigate to the returned plan ID; do not ask the parent backend for a plan.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-app && npm test -- --run src/pages/HomePage.test.tsx src/components`

```bash
git add kid-workbench/english-app/src/pages/HomePage.tsx kid-workbench/english-app/src/pages/HomePage.test.tsx kid-workbench/english-app/src/components kid-workbench/english-app/src/App.tsx
git commit -m "feat: add english learning home"
```

### Task 4: Build Word Map and Recognition Pages

**Files:**
- Create: `kid-workbench/english-app/src/pages/WordMapPage.tsx`
- Create: `kid-workbench/english-app/src/pages/WordMapPage.test.tsx`
- Create: `kid-workbench/english-app/src/pages/WordPage.tsx`
- Create: `kid-workbench/english-app/src/pages/WordPage.test.tsx`
- Create: `kid-workbench/english-app/src/components/WordCard.tsx`
- Create: `kid-workbench/english-app/src/components/SenseImage.tsx`
- Create: `kid-workbench/english-app/src/components/StatusBadge.tsx`
- Modify: `kid-workbench/english-app/src/App.tsx`

- [ ] **Step 1: Write map and recognition tests**

Assert topics are ordered, every status has text plus color semantics, a missing sense image falls back to a spelling card, `meaningZh` is always shown, and absent optional phonetic/example sections are omitted.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- --run src/pages/WordMapPage.test.tsx src/pages/WordPage.test.tsx`

Expected: FAIL because pages and components are undefined.

- [ ] **Step 3: Implement word map**

Group words by module; render English, small sense image or spelling fallback, and mastery status. Clicking the card opens `/words/:kpId`; audio playback is a separate labelled control.

- [ ] **Step 4: Implement recognition page**

Render large spelling, reviewed Chinese meaning, optional phonetic/part of speech/example, sense image fallback, status, previous/next navigation, audio button, and a module-scoped “练一练” action.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-app && npm test -- --run src/pages/WordMapPage.test.tsx src/pages/WordPage.test.tsx && npm run build`

```bash
git add kid-workbench/english-app/src/pages kid-workbench/english-app/src/components kid-workbench/english-app/src/App.tsx
git commit -m "feat: add english word recognition"
```

### Task 5: Add One-at-a-Time Audio and Preloading

**Files:**
- Create: `kid-workbench/english-app/src/audio/AudioController.ts`
- Create: `kid-workbench/english-app/src/audio/AudioController.test.ts`
- Create: `kid-workbench/english-app/src/audio/preload.ts`
- Create: `kid-workbench/english-app/src/audio/preload.test.ts`
- Create: `kid-workbench/english-app/src/hooks/useEnglishAudio.ts`
- Create: `kid-workbench/english-app/src/components/AudioButton.tsx`
- Modify: `kid-workbench/english-app/src/pages/HomePage.tsx`
- Modify: `kid-workbench/english-app/src/pages/WordMapPage.tsx`
- Modify: `kid-workbench/english-app/src/pages/WordPage.tsx`

- [ ] **Step 1: Write controller lifecycle tests**

Assert playing a second clip pauses the first, route cleanup stops playback, `NotAllowedError` becomes an unlock-required state, `404 asset_missing` becomes missing, and network failures remain retryable.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- --run src/audio`

Expected: FAIL because audio modules are undefined.

- [ ] **Step 3: Implement the single controller**

```ts
export class AudioController {
  private current?: HTMLAudioElement
  async play(src: string) {
    this.stop()
    const response = await fetch(src)
    if (!response.ok) throw new AudioHTTPError(response.status)
    const objectURL = URL.createObjectURL(await response.blob())
    this.current = new Audio(objectURL)
    await this.current.play()
  }
  stop() {
    this.current?.pause()
    this.current = undefined
  }
}
```

Revoke the object URL on stop/end, classify `404` as missing and `503` as unavailable from the fetch response, and add a one-time user-gesture unlock. Keep `NotAllowedError` from `play()` as the separate unlock-required state.

- [ ] **Step 4: Implement bounded preload**

Preload only current and next question audio plus current/next images. Dispose stale `Audio` and `Image` references when the item or plan changes.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-app && npm test -- --run src/audio src/pages/WordPage.test.tsx`

```bash
git add kid-workbench/english-app/src/audio kid-workbench/english-app/src/hooks kid-workbench/english-app/src/components/AudioButton.tsx kid-workbench/english-app/src/pages
git commit -m "feat: add english audio playback"
```

### Task 6: Build the Two-Try Practice Flow

**Files:**
- Create: `kid-workbench/english-app/src/pages/PracticePage.tsx`
- Create: `kid-workbench/english-app/src/pages/PracticePage.test.tsx`
- Create: `kid-workbench/english-app/src/components/PracticeProgress.tsx`
- Create: `kid-workbench/english-app/src/components/WordOption.tsx`
- Create: `kid-workbench/english-app/src/components/PictureOption.tsx`
- Create: `kid-workbench/english-app/src/components/ExitConfirm.tsx`
- Modify: `kid-workbench/english-app/src/App.tsx`

- [ ] **Step 1: Write complete interaction tests**

Cover hidden answer before submission, automatic audio after unlock, manual replay, option lock while submitting, one retry after first wrong, advance after correct, reveal after second wrong, persistent `clientId` on network retry, resume first pending item, completed-plan redirect, and exit confirmation.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- --run src/pages/PracticePage.test.tsx`

Expected: FAIL because practice UI is undefined.

- [ ] **Step 3: Implement one-screen landscape layout**

Render progress/header above a centered replay control and a two-by-two option grid. Prevent document scroll while answering. Use text tiles for `listen` and image tiles with spelling fallback for `picture`.

- [ ] **Step 4: Implement stable submission state**

Measure `costMs` from item visibility, obtain a persistent UUID for the current try, keep it until acknowledged, update local item status without refetch flicker, and invalidate home/progress after acknowledgement.

- [ ] **Step 5: Implement feedback timing**

Lock immediately. Hold correct feedback about 900ms, retry feedback about 1500ms, and final reveal about 2400ms. Stop audio before feedback or navigation.

- [ ] **Step 6: Test and commit**

Run: `cd kid-workbench/english-app && npm test -- --run src/pages/PracticePage.test.tsx src/components`

```bash
git add kid-workbench/english-app/src/pages/PracticePage.tsx kid-workbench/english-app/src/pages/PracticePage.test.tsx kid-workbench/english-app/src/components kid-workbench/english-app/src/App.tsx
git commit -m "feat: add english listening practice"
```

### Task 7: Add Result and PWA Behavior

**Files:**
- Create: `kid-workbench/english-app/src/pages/ResultPage.tsx`
- Create: `kid-workbench/english-app/src/pages/ResultPage.test.tsx`
- Create: `kid-workbench/english-app/src/components/StarRow.tsx`
- Create: `kid-workbench/english-app/src/components/WeakWordList.tsx`
- Create: `kid-workbench/english-app/public/manifest.webmanifest`
- Create: `kid-workbench/english-app/public/english-icon.svg`
- Create: `kid-workbench/english-app/public/english-icon-180.png`
- Modify: `kid-workbench/english-app/index.html`
- Modify: `kid-workbench/english-app/src/App.tsx`

- [ ] **Step 1: Write result and accessibility tests**

Assert completed/correct counts, stars, flowers, at most six weak words, exactly “复习错词” and “回到首页” actions, no percentages, and meaningful labels for image/audio controls.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- --run src/pages/ResultPage.test.tsx`

Expected: FAIL because result components are undefined.

- [ ] **Step 3: Implement result page**

Fetch/reuse finished plan data, show child-friendly outcome, create a `review` plan from the review action, and navigate home without creating a new plan.

- [ ] **Step 4: Add PWA metadata**

Use `display: standalone`, `orientation: landscape`, matching theme/background colors, Apple touch icon metadata, and no offline answer queue in version one.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/english-app && npm test -- --run && npm run build`

```bash
git add kid-workbench/english-app
git commit -m "feat: finish english learning flow"
```

### Task 8: Add Docker and iPad Acceptance

**Files:**
- Create: `kid-workbench/english-app/Dockerfile`
- Create: `kid-workbench/english-app/nginx.conf`
- Create: `kid-workbench/english-app/playwright.config.ts`
- Create: `kid-workbench/english-app/e2e/english-flow.spec.ts`
- Create: `kid-workbench/english-app/README.md`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Add production proxy**

```nginx
location /api/ {
    proxy_pass http://english-server:19121;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}

location / {
    try_files $uri $uri/ /index.html;
}
```

- [ ] **Step 2: Add Compose service**

Expose `19122:80`, depend only on healthy `english-server`, and do not depend on content-admin or parent backend.

- [ ] **Step 3: Write iPad flow**

At `1024×768`, open home, enter word map, open and play one word, return home, create/resume a plan, answer through completion, and assert the result. Assert primary controls render at least `56px` high and the practice page has no vertical scroll.

- [ ] **Step 4: Run all checks**

Run: `cd kid-workbench/english-app && npm test -- --run`

Run: `cd kid-workbench/english-app && npm run build`

Run: `cd kid-workbench && docker compose config --quiet && docker compose build english-app`

Run: `cd kid-workbench/english-app && npm run e2e`

Expected: all PASS.

- [ ] **Step 5: Verify service independence**

Stop content-admin and parent backend while keeping PostgreSQL, MinIO, `english-server` and `english-app` running. Reload an existing word, play audio, complete practice, and confirm browser requests target only port `19122` with `/api` proxied internally.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/english-app kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "build: add english app to workspace"
```
