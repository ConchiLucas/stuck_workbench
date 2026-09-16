# Science App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an independently deployable iPad-first React app for exploring published science concepts and completing science-only practice through `science-server`.

**Architecture:** `science-app` has one same-origin API dependency, stores only local child identity and pending answer IDs, and never calls content-admin, parent-dashboard, PostgreSQL or MinIO. It reuses interaction lessons from `literacy-app` but owns science-specific pages and visual language.

**Tech Stack:** React 19, TypeScript 6, Vite 8, React Router 7, TanStack Query 5, Zustand 5, Tailwind CSS 3, Vitest, Testing Library, Playwright

**Prerequisite:** Complete `2026-09-01-science-server.md` through Task 6 before integrating live API flows.

---

### Task 1: Scaffold the Independent App and Test Harness

**Files:**
- Create: `kid-workbench/science-app/package.json`
- Create: `kid-workbench/science-app/index.html`
- Create: `kid-workbench/science-app/tsconfig.json`
- Create: `kid-workbench/science-app/tsconfig.app.json`
- Create: `kid-workbench/science-app/vite.config.ts`
- Create: `kid-workbench/science-app/tailwind.config.js`
- Create: `kid-workbench/science-app/postcss.config.js`
- Create: `kid-workbench/science-app/src/main.tsx`
- Create: `kid-workbench/science-app/src/App.tsx`
- Create: `kid-workbench/science-app/src/App.test.tsx`
- Create: `kid-workbench/science-app/src/styles/index.css`

- [ ] **Step 1: Create scripts and dependencies**

Use the same major React/Vite/Query/Zustand/Tailwind versions as `literacy-app`. Add:

```json
{
  "scripts": {
    "dev": "vite --host",
    "build": "tsc -b && vite build",
    "test": "vitest run",
    "test:e2e": "playwright test",
    "preview": "vite preview"
  }
}
```

Set Vite port `19122` and proxy `/api` to `VITE_API_PROXY || 'http://localhost:19121'`.

- [ ] **Step 2: Write the failing route test**

```tsx
it('renders science home at root', () => {
  render(<App />, { wrapper: MemoryRouterWrapper('/') })
  expect(screen.getByRole('heading', { name: '今天想发现什么？' })).toBeInTheDocument()
})
```

- [ ] **Step 3: Run to verify failure**

Run: `cd kid-workbench/science-app && npm test -- App.test.tsx`

Expected: FAIL because providers and routes are absent.

- [ ] **Step 4: Add providers and route shells**

```tsx
<Routes>
  <Route element={<Shell />}>
    <Route index element={<ScienceHome />} />
    <Route path="explore" element={<ScienceMap />} />
    <Route path="concept/:kpId" element={<ConceptPage />} />
    <Route path="practice/:planId" element={<PracticePage />} />
    <Route path="practice/:planId/done" element={<ResultPage />} />
  </Route>
</Routes>
```

Use placeholder page components only long enough to make routing tests pass; do not copy Hanzi Writer or literacy-specific components.

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/science-app && npm test && npm run build`

```bash
git add kid-workbench/science-app
git commit -m "feat: scaffold independent science app"
```

### Task 2: Add API Contracts, Errors and Child State

**Files:**
- Create: `kid-workbench/science-app/src/api/client.ts`
- Create: `kid-workbench/science-app/src/api/client.test.ts`
- Create: `kid-workbench/science-app/src/api/types.ts`
- Create: `kid-workbench/science-app/src/api/science.ts`
- Create: `kid-workbench/science-app/src/api/plans.ts`
- Create: `kid-workbench/science-app/src/store/childStore.ts`

- [ ] **Step 1: Write envelope and error tests**

Test successful JSON, `404 asset_missing`, `503 database_unavailable`, non-JSON 500 responses, and request cancellation.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-app && npm test -- client.test.ts`

- [ ] **Step 3: Implement a single same-origin client**

```ts
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) { super(message) }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, body.code ?? 'request_failed', body.error ?? `请求失败 ${res.status}`)
  return (body.data ?? body) as T
}
```

All asset URLs remain relative. Do not define a second API base for `19091` or `19081`.

- [ ] **Step 4: Define exact domain contracts**

Add types matching science-server DTOs for `Home`, `ModuleProgress`, `ScienceItem`, `PlanDetail`, `AnswerResult`, and `FinishResult`. `Question` must not contain `answer` or `answerIndex` before submission.

Persist child ID with Zustand using default `1`. Persist a pending `clientId` per plan item and try until the request succeeds, then clear it.

Expose typed plan creation for all server modes:

```ts
export type PlanMode = 'daily' | 'module' | 'review'
export const createPlan = (childId: number, mode: PlanMode, moduleCode = '') =>
  request<PlanDetail>(`/children/${childId}/science/plans`, {
    method: 'POST',
    body: JSON.stringify({ mode, moduleCode }),
  })
```

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-app && npm test && npm run build`

```bash
git add kid-workbench/science-app/src/api kid-workbench/science-app/src/store
git commit -m "feat: add science app API contracts"
```

### Task 3: Build the Home and Science Map

**Files:**
- Create: `kid-workbench/science-app/src/features/home/ScienceHome.tsx`
- Create: `kid-workbench/science-app/src/features/home/ScienceHome.test.tsx`
- Create: `kid-workbench/science-app/src/features/explore/ScienceMap.tsx`
- Create: `kid-workbench/science-app/src/features/explore/ScienceMap.test.tsx`
- Create: `kid-workbench/science-app/src/components/TopicCard.tsx`
- Create: `kid-workbench/science-app/src/components/StatusMark.tsx`
- Modify: `kid-workbench/science-app/src/styles/index.css`

- [ ] **Step 1: Write loading, empty, error and populated tests**

Assert home shows today task/review/explored counts, the map groups all returned modules, and cards expose status without relying only on color.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-app && npm test -- ScienceHome.test.tsx ScienceMap.test.tsx`

- [ ] **Step 3: Implement the iPad-first structure**

Use `1024 × 768` as the main viewport, minimum 56px controls, safe-area padding and no hover-only actions. Keep the home hierarchy to two primary actions: “开始今日探索” and “科普地图”.

Map modules to science-specific labels without changing API codes:

```ts
export const TOPIC_LABELS: Record<string, string> = {
  animal: '动物世界', plant: '植物花园', weather: '天气观察站',
  space: '宇宙空间站', body: '身体实验室', traffic: '交通',
  eco: '环保', safety: '安全训练营', material: '材料', life: '生活常识',
}
```

- [ ] **Step 4: Implement recoverable states**

Loading uses fixed-size skeletons; empty content explains that no science content is published; errors have a visible retry button. Do not display admin actions or internal error details.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-app && npm test && npm run build`

```bash
git add kid-workbench/science-app/src/features/home kid-workbench/science-app/src/features/explore kid-workbench/science-app/src/components kid-workbench/science-app/src/styles
git commit -m "feat: add science home and topic map"
```

### Task 4: Build Knowledge Cards and Asset Fallbacks

**Files:**
- Create: `kid-workbench/science-app/src/features/concept/ConceptPage.tsx`
- Create: `kid-workbench/science-app/src/features/concept/ConceptPage.test.tsx`
- Create: `kid-workbench/science-app/src/audio/controller.ts`
- Create: `kid-workbench/science-app/src/audio/controller.test.ts`
- Create: `kid-workbench/science-app/src/components/ScienceImage.tsx`

- [ ] **Step 1: Write fallback and playback tests**

Test sense image success, broken image fallback to Emoji/title, absent summary/fun fact hiding empty blocks, one active audio element, and audio cleanup on route change.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-app && npm test -- ConceptPage.test.tsx controller.test.ts`

- [ ] **Step 3: Implement one audio controller**

```ts
export class AudioController {
  private current?: HTMLAudioElement
  async play(url: string) {
    this.stop()
    const audio = new Audio(url)
    this.current = audio
    await audio.play()
  }
  stop() {
    this.current?.pause()
    if (this.current) this.current.currentTime = 0
    this.current = undefined
  }
}
```

Audio starts only from an explicit tap. Route cleanup calls `stop()`.

- [ ] **Step 4: Implement the knowledge card**

Order content as sense image → title → summary → explanation → optional fun fact. Show “试一题” only when the API reports available published questions. A `404 asset_missing` changes only the affected media region.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-app && npm test && npm run build`

```bash
git add kid-workbench/science-app/src/features/concept kid-workbench/science-app/src/audio kid-workbench/science-app/src/components/ScienceImage.tsx
git commit -m "feat: add science knowledge cards"
```

### Task 5: Build the Two-Try Practice Flow

**Files:**
- Create: `kid-workbench/science-app/src/features/practice/PracticePage.tsx`
- Create: `kid-workbench/science-app/src/features/practice/PracticePage.test.tsx`
- Create: `kid-workbench/science-app/src/components/OptionButton.tsx`
- Create: `kid-workbench/science-app/src/components/ProgressDots.tsx`
- Create: `kid-workbench/science-app/src/hooks/usePendingAnswer.ts`
- Create: `kid-workbench/science-app/src/hooks/usePendingAnswer.test.ts`

- [ ] **Step 1: Write interaction tests**

Cover input lock, wrong first try, correct retry, exhausted retries, explanation display, duplicate-tap suppression, network retry with the same `clientId`, reload resume and safety-question explicit feedback.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-app && npm test -- PracticePage.test.tsx usePendingAnswer.test.ts`

- [ ] **Step 3: Implement the answer state machine**

```ts
export type PracticePhase = 'asking' | 'submitting' | 'retry' | 'correct' | 'revealed'
```

Generate one UUID before submission and reuse it until the request returns successfully. Lock options while submitting and for the feedback transition. Never calculate correctness in the browser.

- [ ] **Step 4: Implement iPad layout and feedback**

Use one question per screen with a fixed media/stem region and a two-column option grid. Fix media dimensions to prevent layout shift. After correct or exhausted attempts, show `explanation`; safety questions must state the correct action before advancing.

- [ ] **Step 5: Preload bounded assets**

Preload only current and next question images/audio. Cancel or discard preloads when leaving the plan; do not download the entire topic.

- [ ] **Step 6: Test and commit**

Run: `cd kid-workbench/science-app && npm test && npm run build`

```bash
git add kid-workbench/science-app/src/features/practice kid-workbench/science-app/src/components kid-workbench/science-app/src/hooks
git commit -m "feat: add science practice flow"
```

### Task 6: Add Result, Resume and PWA Behavior

**Files:**
- Create: `kid-workbench/science-app/src/features/result/ResultPage.tsx`
- Create: `kid-workbench/science-app/src/features/result/ResultPage.test.tsx`
- Create: `kid-workbench/science-app/public/manifest.webmanifest`
- Create: `kid-workbench/science-app/public/science-icon.svg`
- Create: `kid-workbench/science-app/public/science-icon-180.png`
- Modify: `kid-workbench/science-app/index.html`
- Modify: `kid-workbench/science-app/src/App.tsx`

- [ ] **Step 1: Write result and resume tests**

Assert result shows new discoveries, correct/completed counts, stars, flowers and review items; completed plans redirect to result; incomplete plans resume the first pending item.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/science-app && npm test -- ResultPage.test.tsx`

- [ ] **Step 3: Implement result actions**

Provide exactly “回到首页”, “复习错题”, and “继续这个主题” when a module is known. “复习错题” creates a `review` plan; “继续这个主题” creates a `module` plan with the completed plan's module code. Do not show percentages or answer keys for unrelated questions.

- [ ] **Step 4: Add PWA metadata**

Set standalone display, landscape orientation, theme/background colors, iPad touch icon and safe-area viewport metadata. Respect `prefers-reduced-motion` in celebration styles.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/science-app && npm test && npm run build`

```bash
git add kid-workbench/science-app
git commit -m "feat: finish science app learning loop"
```

### Task 7: Add Docker, Compose and iPad Acceptance

**Files:**
- Create: `kid-workbench/science-app/Dockerfile`
- Create: `kid-workbench/science-app/nginx.conf`
- Create: `kid-workbench/science-app/README.md`
- Create: `kid-workbench/science-app/playwright.config.ts`
- Create: `kid-workbench/science-app/e2e/science-flow.spec.ts`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Add the production proxy**

Serve the Vite build with Nginx and proxy only `/api/` to `http://science-server:19121`. Add SPA fallback. Do not define routes to content-admin or parent-dashboard.

- [ ] **Step 2: Add Compose service**

Expose `19122:80`, depend only on `science-server` health, and keep the app in the default network. It must not receive database or MinIO credentials.

- [ ] **Step 3: Write the iPad acceptance flow**

At Playwright viewport `1024 × 768`, verify home → topic map → knowledge card → practice → wrong retry → finish → result. Assert primary touch targets are at least 56px and the page has no horizontal overflow.

- [ ] **Step 4: Run all checks**

Run: `cd kid-workbench/science-app && npm test && npm run build && npm run test:e2e`

Run: `cd kid-workbench && docker compose config`

Run: `cd kid-workbench && docker build -f science-app/Dockerfile .`

- [ ] **Step 5: Verify runtime independence**

With a published concept and existing asset, stop only content-admin and parent-dashboard. Verify `http://localhost:19122` still loads the concept and completes a science-server plan. Restart stopped services afterward; preserve volumes.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/science-app kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "build: deploy independent science app"
```
