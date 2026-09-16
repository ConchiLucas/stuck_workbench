# Pinyin App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an iPad-landscape PWA where a child can explore pinyin, hear prepared audio, complete stable two-try exercises, resume interrupted plans, and see a simple result.

**Architecture:** Create a standalone React application that talks only to pinyin-server through `/api/v1`. TanStack Query owns server state, Zustand stores the local child ID and pending answer client IDs, and one audio controller guarantees single-track playback with cleanup on navigation.

**Tech Stack:** React 19, TypeScript, Vite 8, React Router 7, TanStack Query 5, Zustand 5, Tailwind CSS 3, Vitest, Testing Library, Playwright, Nginx, Docker

---

### Task 1: Scaffold the App and Test Harness

**Files:**
- Create: `kid-workbench/pinyin-app/package.json`
- Create: `kid-workbench/pinyin-app/tsconfig.json`
- Create: `kid-workbench/pinyin-app/tsconfig.app.json`
- Create: `kid-workbench/pinyin-app/tsconfig.node.json`
- Create: `kid-workbench/pinyin-app/vite.config.ts`
- Create: `kid-workbench/pinyin-app/index.html`
- Create: `kid-workbench/pinyin-app/src/main.tsx`
- Create: `kid-workbench/pinyin-app/src/App.tsx`
- Create: `kid-workbench/pinyin-app/src/App.test.tsx`
- Create: `kid-workbench/pinyin-app/src/test/setup.ts`

- [ ] **Step 1: Create package scripts and dependencies**

Use scripts `dev`, `build`, `test`, `test:watch`, and `e2e`. Match literacy-app production dependencies, omit `hanzi-writer`, and add Vitest, jsdom, Testing Library, user-event and Playwright as dev dependencies.

- [ ] **Step 2: Write the failing route test**

```tsx
it('renders the pinyin home route', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('heading', { name: '今天学拼音' })).toBeInTheDocument()
})
```

- [ ] **Step 3: Run it to verify failure**

Run: `cd kid-workbench/pinyin-app && npm install && npm test -- --run src/App.test.tsx`

Expected: FAIL because `App` and the home page are not implemented.

- [ ] **Step 4: Add minimal providers and routes**

```tsx
export function AppRoutes() {
  return (
    <Routes>
      <Route element={<Shell />}>
        <Route index element={<HomePage />} />
        <Route path="map" element={<MapPage />} />
        <Route path="learn/:kpId" element={<LearnPage />} />
        <Route path="practice/:planId" element={<PracticePage />} />
        <Route path="practice/:planId/result" element={<ResultPage />} />
      </Route>
    </Routes>
  )
}

<QueryClientProvider client={queryClient}>
  <BrowserRouter>
    <AppRoutes />
  </BrowserRouter>
</QueryClientProvider>
```

- [ ] **Step 5: Run and commit**

Run: `cd kid-workbench/pinyin-app && npm test -- --run && npm run build`

```bash
git add kid-workbench/pinyin-app
git commit -m "feat: scaffold pinyin app"
```

### Task 2: Add API Contracts and Child State

**Files:**
- Create: `kid-workbench/pinyin-app/src/api/client.ts`
- Create: `kid-workbench/pinyin-app/src/api/types.ts`
- Create: `kid-workbench/pinyin-app/src/api/pinyin.ts`
- Create: `kid-workbench/pinyin-app/src/api/pinyin.test.ts`
- Create: `kid-workbench/pinyin-app/src/store/childStore.ts`
- Create: `kid-workbench/pinyin-app/src/store/pendingAnswerStore.ts`

- [ ] **Step 1: Write client envelope and error tests**

Mock fetch and assert a successful `{data,error:null}` response is unwrapped, while `{data:null,error:{code,message}}` throws an `ApiError` preserving HTTP status and code.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/api/pinyin.test.ts`

Expected: FAIL because the API client is undefined.

- [ ] **Step 3: Implement the client**

```ts
export class ApiError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) {
    super(message)
  }
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
}
```

Define exact TypeScript types matching pinyin-server home, progress, item, plan, question, answer and result DTOs.

- [ ] **Step 4: Add stores**

Default `childId` to `1`. Store one UUID per `planId:itemId:tryNumber` until the server acknowledges it; a retry after a network error must reuse the same UUID.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/api src/store`

```bash
git add kid-workbench/pinyin-app/src/api kid-workbench/pinyin-app/src/store
git commit -m "feat: add pinyin app data contracts"
```

### Task 3: Build Home and Pinyin Map

**Files:**
- Create: `kid-workbench/pinyin-app/src/pages/HomePage.tsx`
- Create: `kid-workbench/pinyin-app/src/pages/HomePage.test.tsx`
- Create: `kid-workbench/pinyin-app/src/pages/MapPage.tsx`
- Create: `kid-workbench/pinyin-app/src/pages/MapPage.test.tsx`
- Create: `kid-workbench/pinyin-app/src/components/StatusBadge.tsx`
- Create: `kid-workbench/pinyin-app/src/components/ModuleProgress.tsx`
- Create: `kid-workbench/pinyin-app/src/styles/tokens.css`
- Create: `kid-workbench/pinyin-app/src/styles/app.css`

- [ ] **Step 1: Write loading, empty and populated tests**

For home, assert an active plan produces “继续练习”; without a plan it produces “开始练习”. For the map, assert pinyin items are grouped into 声母 and 韵母 and each status has both text and color semantics.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/pages/HomePage.test.tsx src/pages/MapPage.test.tsx`

Expected: FAIL because the pages are placeholders.

- [ ] **Step 3: Implement iPad-first layout**

Use a minimum `56px` touch target, no hover-only affordances, safe-area padding, and a `1024×768` primary layout. Home has exactly two primary actions: practice and pinyin map.

- [ ] **Step 4: Implement data states**

Show a skeleton while loading, a child-friendly retry card for API failures, and “还没有拼音内容” for an empty catalog. Do not display raw error codes to the child.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/pages`

```bash
git add kid-workbench/pinyin-app/src/pages kid-workbench/pinyin-app/src/components kid-workbench/pinyin-app/src/styles
git commit -m "feat: add pinyin home and map"
```

### Task 4: Build Single-Audio Playback and Learn Page

**Files:**
- Create: `kid-workbench/pinyin-app/src/audio/controller.ts`
- Create: `kid-workbench/pinyin-app/src/audio/controller.test.ts`
- Create: `kid-workbench/pinyin-app/src/hooks/usePinyinAudio.ts`
- Create: `kid-workbench/pinyin-app/src/pages/LearnPage.tsx`
- Create: `kid-workbench/pinyin-app/src/pages/LearnPage.test.tsx`
- Create: `kid-workbench/pinyin-app/src/components/FourLineGrid.tsx`

- [ ] **Step 1: Write audio cleanup tests**

Assert starting a second clip pauses the first, leaving the page pauses playback, and failed playback returns a retryable state.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/audio src/pages/LearnPage.test.tsx`

Expected: FAIL because the controller and page are undefined.

- [ ] **Step 3: Implement one shared controller**

```ts
class AudioController {
  private current?: HTMLAudioElement
  async play(src: string) {
    this.stop()
    this.current = new Audio(src)
    await this.current.play()
  }
  stop() {
    this.current?.pause()
    this.current = undefined
  }
}
```

Add event cleanup and explicit `ended`/`error` state handling around this minimal shape.

- [ ] **Step 4: Implement learn page rules**

Render the pinyin in a four-line grid, separate solo and example-word buttons, hide solo content when `soloText` is empty, and fall back to text when glyph or audio returns `asset_missing`.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/audio src/pages/LearnPage.test.tsx && npm run build`

```bash
git add kid-workbench/pinyin-app/src/audio kid-workbench/pinyin-app/src/hooks kid-workbench/pinyin-app/src/pages/LearnPage.tsx kid-workbench/pinyin-app/src/components/FourLineGrid.tsx
git commit -m "feat: add pinyin recognition experience"
```

### Task 5: Build Two-Try Practice Flow

**Files:**
- Create: `kid-workbench/pinyin-app/src/pages/PracticePage.tsx`
- Create: `kid-workbench/pinyin-app/src/pages/PracticePage.test.tsx`
- Create: `kid-workbench/pinyin-app/src/components/QuestionStem.tsx`
- Create: `kid-workbench/pinyin-app/src/components/PinyinOption.tsx`
- Create: `kid-workbench/pinyin-app/src/components/PracticeProgress.tsx`
- Create: `kid-workbench/pinyin-app/src/hooks/useNextAudioPreload.ts`

- [ ] **Step 1: Write interaction tests**

Cover: server answer hidden before submission; option locks while pending; wrong first answer enables one retry; correct answer advances; second wrong answer advances; network retry reuses the same `clientId`; completed plan redirects to result.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/pages/PracticePage.test.tsx`

Expected: FAIL because practice components are undefined.

- [ ] **Step 3: Implement the fixed three-column layout**

At iPad landscape width, render stem on the left and four choices in two columns on the right. Keep one question per screen and prevent vertical page scroll during answering.

- [ ] **Step 4: Implement stable answer submission**

Measure elapsed time from question visibility, get the persistent client ID from `pendingAnswerStore`, submit `optionIndex` and `costMs`, clear the key only after acknowledgement, and invalidate plan/home/progress queries.

- [ ] **Step 5: Add audio preloading**

Preload only current and next question audio using `new Audio(url).preload = 'auto'`. Cancel stale preload references when plan or item changes.

- [ ] **Step 6: Test and commit**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/pages/PracticePage.test.tsx src/components`

```bash
git add kid-workbench/pinyin-app/src/pages/PracticePage.tsx kid-workbench/pinyin-app/src/components kid-workbench/pinyin-app/src/hooks/useNextAudioPreload.ts
git commit -m "feat: add pinyin practice flow"
```

### Task 6: Add Result, Resume, and PWA Behavior

**Files:**
- Create: `kid-workbench/pinyin-app/src/pages/ResultPage.tsx`
- Create: `kid-workbench/pinyin-app/src/pages/ResultPage.test.tsx`
- Create: `kid-workbench/pinyin-app/src/components/StarRow.tsx`
- Create: `kid-workbench/pinyin-app/public/manifest.webmanifest`
- Create: `kid-workbench/pinyin-app/public/pinyin-icon.svg`
- Create: `kid-workbench/pinyin-app/public/pinyin-icon-180.png`
- Modify: `kid-workbench/pinyin-app/index.html`

- [ ] **Step 1: Write result and resume tests**

Assert result shows completed/correct counts, stars, flowers and weak pinyin; assert loading an unfinished plan resumes the first pending item instead of creating a new plan.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/pages/ResultPage.test.tsx`

Expected: FAIL because result and resume behavior are undefined.

- [ ] **Step 3: Implement result actions**

Provide exactly “回到首页” and “复习错题”. Do not expose percentages or raw mastery algorithm values.

- [ ] **Step 4: Add PWA metadata**

Set display to `standalone`, orientation to `landscape`, theme/background colors matching app tokens, and include Apple touch icon metadata. Do not add offline answer submission in the first version.

- [ ] **Step 5: Test and commit**

Run: `cd kid-workbench/pinyin-app && npm test -- --run && npm run build`

```bash
git add kid-workbench/pinyin-app
git commit -m "feat: finish pinyin app learning flow"
```

### Task 7: Add Docker, Compose, and Browser Acceptance

**Files:**
- Create: `kid-workbench/pinyin-app/Dockerfile`
- Create: `kid-workbench/pinyin-app/nginx.conf`
- Create: `kid-workbench/pinyin-app/playwright.config.ts`
- Create: `kid-workbench/pinyin-app/e2e/pinyin-flow.spec.ts`
- Create: `kid-workbench/pinyin-app/README.md`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Add the production proxy**

```nginx
location /api/ {
    proxy_pass http://pinyin-server:19111;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}

location / {
    try_files $uri $uri/ /index.html;
}
```

- [ ] **Step 2: Add Compose service**

Expose `19112:80`, depend only on healthy `pinyin-server`, and leave content-admin and parent backend out of `depends_on`.

- [ ] **Step 3: Write iPad acceptance flow**

At viewport `1024×768`, open home, enter pinyin map, play one available item, return home, create or resume a plan, answer through completion, and assert the result page. Add assertions that primary buttons have at least 56px rendered height.

- [ ] **Step 4: Run all checks**

Run: `cd kid-workbench/pinyin-app && npm test -- --run`

Run: `cd kid-workbench/pinyin-app && npm run build`

Run: `cd kid-workbench && docker compose config --quiet && docker compose build pinyin-app`

Run: `cd kid-workbench/pinyin-app && npm run e2e`

Expected: unit tests, TypeScript build, Docker build and iPad flow all PASS.

- [ ] **Step 5: Verify runtime dependency boundary**

Stop content-admin and parent backend while leaving PostgreSQL, MinIO, pinyin-server and pinyin-app running. Reload an already-generated pinyin item and complete a practice plan.

Expected: the app still works and browser network requests target only port `19112` with `/api` proxied to pinyin-server.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/pinyin-app kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "build: add pinyin app to workspace"
```
