# Kid Question Type Playground Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 孩子在拼音 / 科普 / 英语 / 算数四个首页点任意题型卡，都能直接做完 4 题并看到对错结果。

**Architecture:** 不抽公共包。四个孩子端各自实现同一套交互契约。拼音继续调用 `pinyin-server` 运行时出题；科普、英语、算数用本地题库。算数首页改为直达练习，不再先进入详情页。

**Tech Stack:** React 19, TypeScript, Vite, React Router, Zustand, Vitest, Testing Library, Go/Gin（仅拼音出题），Docker Compose 孩子端镜像

**Spec:** `kid-workbench/docs/superpowers/specs/2026-09-06-kid-question-type-playground-design.md`

---

## File map

| 职责 | 文件 |
|------|------|
| 拼音出题客户端与错误文案 | `kid-workbench/pinyin-app/src/api/client.ts`, `src/api/pinyin.ts`, `src/store/demoQuizStore.ts`, `src/App.test.tsx` |
| 科普结果页 | `kid-workbench/science-app/src/store/demoAnswerStore.ts`（新建）, `src/pages/QuestionTypePage.tsx`, `src/pages/QuestionTypeResultPage.tsx`（新建）, `src/App.tsx`, `src/App.test.tsx` |
| 英语走查 | `kid-workbench/english-app/src/App.test.tsx`, `src/QuestionTypePreview.tsx`（仅缺口才改） |
| 算数直达练习 | `kid-workbench/math-app/src/content/typePracticeBanks.ts`（新建）, `src/pages/TypePracticePage.tsx`（新建）, `src/pages/TypeResultPage.tsx`（新建）, `src/store/typePracticeStore.ts`（新建）, `src/pages/HomePage.tsx`, `src/App.tsx`, `src/App.test.tsx`, `src/content/questionTypePrototype.ts` |
| 算数卡片图 | `kid-workbench/math-app/public/cards/*.png`（已存在，重建镜像后才会出现在 `:19142`） |

---

### Task 1: Pinyin client treats non-JSON as a retryable kid error

**Files:**
- Modify: `kid-workbench/pinyin-app/src/api/client.ts`
- Modify: `kid-workbench/pinyin-app/src/App.test.tsx`
- Modify: `kid-workbench/pinyin-app/src/store/demoQuizStore.ts` only if error text needs a kid-facing fallback

- [ ] **Step 1: Write the failing test**

Add to `kid-workbench/pinyin-app/src/App.test.tsx`:

```tsx
it('shows a retryable message when quiz generate returns HTML', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => new Response('<html>404 page not found</html>', {
    status: 404,
    headers: { 'Content-Type': 'text/plain; charset=utf-8' },
  })))
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByText('出题没有成功，点下面再试一次')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '再试一次' })).toBeInTheDocument()
})
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/App.test.tsx`

Expected: FAIL because the current HTML parse path still shows `服务返回了无法识别的内容`.

- [ ] **Step 3: Map invalid JSON to a kid-facing error**

In `kid-workbench/pinyin-app/src/api/client.ts`, keep envelope parsing, but change the catch:

```ts
  } catch {
    throw new ApiError(response.status, 'invalid_response', '出题没有成功，点下面再试一次')
  }
```

In `kid-workbench/pinyin-app/src/store/demoQuizStore.ts`, when `generatePinyinQuizSet` throws, store `error instanceof ApiError ? error.message : '出题没有成功，点下面再试一次'`.

- [ ] **Step 4: Run tests**

Run: `cd kid-workbench/pinyin-app && npm test -- --run src/App.test.tsx`

Expected: PASS, including existing listen/inword/shape/blend navigation tests.

- [ ] **Step 5: Confirm live generate still returns JSON**

Run:

```bash
curl -sS -X POST http://127.0.0.1:19112/api/v1/pinyin/quiz/generate \
  -H 'Content-Type: application/json' \
  -d '{"type":"listen","excludeTargetIds":[]}'
```

Expected: `{"data":{"type":"listen",...},"error":null}`. If this is HTML/404, rebuild before browser work:

```bash
cd kid-workbench && docker compose up -d --build pinyin-server pinyin-app
```

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/pinyin-app/src/api/client.ts kid-workbench/pinyin-app/src/store/demoQuizStore.ts kid-workbench/pinyin-app/src/App.test.tsx
git commit -m "$(cat <<'EOF'
fix: show a retryable pinyin quiz error when generate returns non-JSON

EOF
)"
```

---

### Task 2: Science last question opens a result page

**Files:**
- Create: `kid-workbench/science-app/src/store/demoAnswerStore.ts`
- Create: `kid-workbench/science-app/src/pages/QuestionTypeResultPage.tsx`
- Modify: `kid-workbench/science-app/src/pages/QuestionTypePage.tsx`
- Modify: `kid-workbench/science-app/src/App.tsx`
- Modify: `kid-workbench/science-app/src/App.test.tsx`

- [ ] **Step 1: Write the failing tests**

Add to `kid-workbench/science-app/src/App.test.tsx`:

```tsx
it('opens the choice practice from the home card', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('link', { name: '查看题型：选择题' })).toHaveAttribute('href', '/question-types/choice')
})

it('shows a result summary after the last science question', async () => {
  render(<MemoryRouter initialEntries={['/question-types/choice']}><AppRoutes /></MemoryRouter>)
  expect(screen.queryByRole('link', { name: '完成' })).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', '/question-types/choice/plant-sun')
})
```

Also add a result-route test once the last question id is known from `questionsFor('choice')`. After implementation the last question must expose `下一题` or a `查看结果` link to `/question-types/choice/result`, and that page must contain `答题结果` and `答对`.

- [ ] **Step 2: Run tests to verify failure**

Run: `cd kid-workbench/science-app && npm test -- --run src/App.test.tsx`

Expected: FAIL on the missing result route / leftover「完成」home link.

- [ ] **Step 3: Add a tiny pick store**

Create `kid-workbench/science-app/src/store/demoAnswerStore.ts`:

```ts
import { create } from 'zustand'

type DemoAnswerStore = {
  solved: Record<string, boolean>
  mark: (questionId: string, correct: boolean) => void
  clearType: (slug: string) => void
}

export const useDemoAnswerStore = create<DemoAnswerStore>((set) => ({
  solved: {},
  mark: (questionId, correct) => set((state) => ({ solved: { ...state.solved, [questionId]: correct } })),
  clearType: (slug) => set((state) => {
    const solved = { ...state.solved }
    for (const key of Object.keys(solved)) {
      if (key.startsWith(`${slug}:`)) delete solved[key]
    }
    return { solved }
  }),
}))
```

Call `mark(\`${type.slug}:${question.id}\`, correct)` from `ChoiceLab` / `MatchLab` / `SequenceLab` / `LabelLab` when the child finishes that interaction correctly or incorrectly. If a lab currently only reports success, also `mark(..., false)` on the first wrong attempt that stays wrong (choice already has `picked !== question.correct`).

- [ ] **Step 4: Add the result page and route**

Create `kid-workbench/science-app/src/pages/QuestionTypeResultPage.tsx` that:

- Reads `slug` from the route
- Uses `questionsFor(slug)` and `useDemoAnswerStore`
- Renders `答题结果`、`答对 {n} / {total} 题`、每题一行、`再练一次` → `/question-types/${slug}`（并 `clearType`）、`回到首页` → `/`

In `kid-workbench/science-app/src/App.tsx` register:

```tsx
<Route path="question-types/:slug/result" element={<QuestionTypeResultPage />} />
<Route path="question-types/:slug/:questionId" element={<QuestionTypePage />} />
<Route path="question-types/:slug" element={<QuestionTypePage />} />
```

Put the result route **above** `:questionId`.

In `QuestionTypePage.tsx`, replace the last-question home `完成` link:

```tsx
{solved && !next && <Link className="next-question" to={`/question-types/${type.slug}/result`}>查看结果</Link>}
```

Also make the stepper's last `下一题` go to the result route when `!next`.

- [ ] **Step 5: Run tests**

Run: `cd kid-workbench/science-app && npm test -- --run && npm run build`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add kid-workbench/science-app/src
git commit -m "$(cat <<'EOF'
feat: add science question-type result page

EOF
)"
```

---

### Task 3: Math home cards go straight into 4-question practice

**Files:**
- Create: `kid-workbench/math-app/src/content/typePracticeBanks.ts`
- Create: `kid-workbench/math-app/src/store/typePracticeStore.ts`
- Create: `kid-workbench/math-app/src/pages/TypePracticePage.tsx`
- Create: `kid-workbench/math-app/src/pages/TypeResultPage.tsx`
- Modify: `kid-workbench/math-app/src/content/questionTypePrototype.ts`
- Modify: `kid-workbench/math-app/src/pages/HomePage.tsx` (only if cards still read `href` from packs)
- Modify: `kid-workbench/math-app/src/App.tsx`
- Modify: `kid-workbench/math-app/src/App.test.tsx`

- [ ] **Step 1: Write failing home and practice tests**

Replace the home assertion in `kid-workbench/math-app/src/App.test.tsx`:

```tsx
it('renders the math home route', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('link', { name: '看算式选答案' })).toHaveAttribute('href', '/practice/type/equation')
  expect(screen.getByRole('link', { name: '看数量图选答案' })).toHaveAttribute('href', '/practice/type/story')
  expect(screen.getByRole('link', { name: '补全算式' })).toHaveAttribute('href', '/practice/type/missing')
  expect(screen.getByRole('link', { name: '判断算式对错' })).toHaveAttribute('href', '/practice/type/judge')
  expect(screen.getByRole('link', { name: '认识图形' })).toHaveAttribute('href', '/practice/type/shape')
  for (const name of ['看算式选答案', '看数量图选答案', '补全算式', '判断算式对错', '认识图形']) {
    expect(screen.getByRole('link', { name }).querySelector('img')).toHaveAttribute('src', expect.stringMatching(/^\/cards\/.+\.png$/))
  }
})

it('starts equation practice with four choices', () => {
  render(<MemoryRouter initialEntries={['/practice/type/equation']}><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('region', { name: '当前题目' })).toBeInTheDocument()
  expect(screen.getAllByRole('button').length).toBeGreaterThanOrEqual(4)
})
```

Keep the existing `/types/shape-feature` detail test so old routes stay intact.

- [ ] **Step 2: Run tests to verify failure**

Run: `cd kid-workbench/math-app && npm test -- --run src/App.test.tsx`

Expected: FAIL because home still links to `/types/addition-equation`.

- [ ] **Step 3: Point packs at practice routes**

In `kid-workbench/math-app/src/content/questionTypePrototype.ts`, change the five `href` values:

```ts
href: '/practice/type/equation'
href: '/practice/type/story'
href: '/practice/type/missing'
href: '/practice/type/judge'
href: '/practice/type/shape'
```

- [ ] **Step 4: Add the local 4-question banks**

Create `kid-workbench/math-app/src/content/typePracticeBanks.ts`:

```ts
export const practiceTypes = ['equation', 'story', 'missing', 'judge', 'shape'] as const
export type PracticeType = (typeof practiceTypes)[number]

export type PracticeQuestion = {
  id: string
  prompt: string
  visual: string
  options: string[]
  answerIndex: number
}

export const typeTitles: Record<PracticeType, string> = {
  equation: '看算式选答案',
  story: '看数量图选答案',
  missing: '补全算式',
  judge: '判断算式对错',
  shape: '认识图形',
}

export const typePracticeBanks: Record<PracticeType, PracticeQuestion[]> = {
  equation: [
    { id: 'eq-1', prompt: '3 + 5 = ?', visual: '3 + 5', options: ['7', '8', '9', '6'], answerIndex: 1 },
    { id: 'eq-2', prompt: '9 − 4 = ?', visual: '9 − 4', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'eq-3', prompt: '6 + 7 = ?', visual: '6 + 7', options: ['12', '13', '14', '11'], answerIndex: 1 },
    { id: 'eq-4', prompt: '12 − 5 = ?', visual: '12 − 5', options: ['6', '7', '8', '9'], answerIndex: 1 },
  ],
  story: [
    { id: 'st-1', prompt: '一共有几颗星星？', visual: '★★★  ★★', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'st-2', prompt: '8 个拿走 3 个，还剩几个？', visual: '●●●●●●●●  −3', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'st-3', prompt: '2 个苹果再放上 2 个，一共几个？', visual: '🍎🍎  🍎🍎', options: ['3', '4', '5', '6'], answerIndex: 1 },
    { id: 'st-4', prompt: '5 只小鸟飞走 1 只，还剩几只？', visual: '🐦🐦🐦🐦🐦  −1', options: ['3', '4', '5', '6'], answerIndex: 1 },
  ],
  missing: [
    { id: 'mi-1', prompt: '3 + □ = 8', visual: '3 + □ = 8', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'mi-2', prompt: '12 − □ = 7', visual: '12 − □ = 7', options: ['3', '4', '5', '6'], answerIndex: 2 },
    { id: 'mi-3', prompt: '□ + 6 = 10', visual: '□ + 6 = 10', options: ['3', '4', '5', '6'], answerIndex: 1 },
    { id: 'mi-4', prompt: '9 − □ = 2', visual: '9 − □ = 2', options: ['5', '6', '7', '8'], answerIndex: 2 },
  ],
  judge: [
    { id: 'ju-1', prompt: '7 + 6 = 12，对吗？', visual: '7 + 6 = 12', options: ['对', '错'], answerIndex: 1 },
    { id: 'ju-2', prompt: '8 + 8 = 16，对吗？', visual: '8 + 8 = 16', options: ['对', '错'], answerIndex: 0 },
    { id: 'ju-3', prompt: '15 − 6 = 9，对吗？', visual: '15 − 6 = 9', options: ['对', '错'], answerIndex: 0 },
    { id: 'ju-4', prompt: '11 − 3 = 7，对吗？', visual: '11 − 3 = 7', options: ['对', '错'], answerIndex: 1 },
  ],
  shape: [
    { id: 'sh-1', prompt: '这是什么图形？', visual: '△', options: ['圆形', '三角形', '正方形', '长方形'], answerIndex: 1 },
    { id: 'sh-2', prompt: '听到：圆形。哪一个是？', visual: '👂', options: ['○', '△', '□', '◇'], answerIndex: 0 },
    { id: 'sh-3', prompt: '找出没有角的图形', visual: '？', options: ['○', '△', '□', '◇'], answerIndex: 0 },
    { id: 'sh-4', prompt: '这是什么图形？', visual: '□', options: ['圆形', '三角形', '正方形', '长方形'], answerIndex: 2 },
  ],
}

export function isPracticeType(value: string | undefined): value is PracticeType {
  return practiceTypes.includes(value as PracticeType)
}

export function practiceHref(type: PracticeType, n = 1) {
  return n <= 1 ? `/practice/type/${type}` : `/practice/type/${type}/${n}`
}

export function practiceResultHref(type: PracticeType) {
  return `/practice/type/${type}/result`
}
```

- [ ] **Step 5: Add pick store, practice page, result page, routes**

Create `kid-workbench/math-app/src/store/typePracticeStore.ts` mirroring pinyin's `useDemoAnswerStore`: `{ picks: Record<string, number>, setPick(type, n, index), pick(type, n), clearType(type) }`.

Create `kid-workbench/math-app/src/pages/TypePracticePage.tsx`:

- `useParams()` for `type` and `n`
- invalid type → `<Navigate to="/" />`
- current question from `typePracticeBanks[type][current-1]`
- close link to `/`
- progress `{current} / 4`
- render `prompt` + `visual` + option buttons
- `setPick` then `setTimeout(450)` navigate to next or result
- judge has 2 options, others 4

Create `kid-workbench/math-app/src/pages/TypeResultPage.tsx` with the same result list pattern as `kid-workbench/pinyin-app/src/pages/DemoResultPage.tsx` (答对 n / 4、再练一次、回到首页).

In `kid-workbench/math-app/src/App.tsx` add **before** `practice/:planId`:

```tsx
<Route path="practice/type/:type/result" element={<TypeResultPage />} />
<Route path="practice/type/:type/:n?" element={<TypePracticePage />} />
```

- [ ] **Step 6: Run tests and build**

Run: `cd kid-workbench/math-app && npm test -- --run && npm run build`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add kid-workbench/math-app
git commit -m "$(cat <<'EOF'
feat: let every math type card start a four-question practice

EOF
)"
```

---

### Task 4: English gallery walkthrough, fix only gaps

**Files:**
- Test: `kid-workbench/english-app/src/App.test.tsx`
- Modify only if a type cannot finish: `kid-workbench/english-app/src/QuestionTypePreview.tsx`, `src/App.tsx`

- [ ] **Step 1: Run the existing suite**

Run: `cd kid-workbench/english-app && npm test -- --run && npm run build`

Expected: PASS. Five home cards already go to `/types/:id` with local demos and `/types/:id/result`.

- [ ] **Step 2: Browser-check each of the five types**

Open `http://localhost:19132`. For 听音选词 / 看图选词 / 组句子 / 写单词 / 读一读: enter, answer at least one item, reach 结果. If a type cannot submit or has no result, add a focused failing test around that type then fix that preview only.

- [ ] **Step 3: Commit only if code changed**

```bash
git add kid-workbench/english-app
git commit -m "$(cat <<'EOF'
fix: complete english type-preview gaps found in playground walkthrough

EOF
)"
```

---

### Task 5: Rebuild kid images and verify all eighteen cards

**Files:** none except compose rebuild

- [ ] **Step 1: Rebuild the four child SPAs (and pinyin-server)**

```bash
cd kid-workbench && docker compose up -d --build \
  pinyin-server pinyin-app science-app english-app math-app
```

Expected: containers healthy; `http://localhost:19112` / `19122` / `19132` / `19142` return 200.

- [ ] **Step 2: Pinyin — four cards**

For listen / inword / shape / blend on `http://localhost:19112`:

1. Card art is visible.
2. Practice loads 4 questions (not HTML error).
3. Listen/inword show letter options; shape/blend require playing audio before picking.
4. Result shows 答对 n / 4.

- [ ] **Step 3: Science — four cards**

On `http://localhost:19122`, finish choice, match, sequence, label. Last step opens 答题结果, not the home gallery.

- [ ] **Step 4: English — five cards**

On `http://localhost:19132`, each card still plays and can open 答题结果.

- [ ] **Step 5: Math — five cards**

On `http://localhost:19142`:

1. All five cards show `/cards/*.png`, not empty white tiles.
2. Clicking a card opens `/practice/type/...` immediately.
3. Four questions then result. 判断算式对错 has 对/错 two options.

- [ ] **Step 6: Commit leftover verification-only doc updates if any**

No commit if only Docker was rebuilt.

---

## Self-review

| Spec requirement | Task |
|------------------|------|
| Click card → 4 questions → result | 1–4 |
| Kid-facing generate error | 1 |
| Science result instead of home「完成」 | 2 |
| Math skip detail, five playable types, card art | 3, 5 |
| English keep five scripts | 4 |
| No mastery / plans / content-admin / literacy | respected |
| Rebuild running Docker images so card PNGs and quiz routes match source | 5 |
