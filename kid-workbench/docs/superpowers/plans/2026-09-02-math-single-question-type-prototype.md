# Math Single Question Type Prototype Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the broad math catalog and module-first home page with one complete frontend-only “20 以内加法 · 看算式选答案” flow.

**Architecture:** Store the prototype definition and five immutable questions in one typed content module. Use a small Zustand session store for current question, tries, score, completion, and result gating; three route pages consume this store without calling the plan API. Existing server-backed practice routes stay available but lose their home-page entry.

**Tech Stack:** React 19, React Router, TypeScript, Zustand, Vitest, Testing Library, CSS.

---

### Task 1: Define prototype content and session behavior

**Files:**
- Create: `math-app/src/content/questionTypePrototype.ts`
- Create: `math-app/src/store/questionTypePracticeStore.ts`
- Create: `math-app/src/store/questionTypePracticeStore.test.ts`

- [x] **Step 1: Write the failing store tests**

Test these exact transitions: `start()` resets the session; a first wrong choice returns `retry`; a second wrong choice returns `revealed`; a correct choice increments `score` only once; `next()` advances and marks the session complete after the fifth question.

```ts
const store = useQuestionTypePracticeStore.getState()
store.start()
expect(store.choose(0)).toBe('retry')
expect(useQuestionTypePracticeStore.getState().choose(1)).toBe('revealed')
```

- [x] **Step 2: Run the store test and verify RED**

Run: `npm test -- --run src/store/questionTypePracticeStore.test.ts`

Expected: FAIL because `questionTypePracticeStore` does not exist.

- [x] **Step 3: Add typed static content**

Export `additionEquationType` with slug `addition-equation`, title `20 以内加法`, mode `看算式选答案`, description, learning goal, rules, one example, and five questions. Each question has `id`, `stem`, `a`, `b`, `options: [number, number, number, number]`, and `answerIndex`.

- [x] **Step 4: Implement the minimal Zustand store**

Expose state `index`, `tries`, `score`, `answered`, `completed`, plus actions `start`, `choose`, and `next`. `choose` returns `'correct' | 'retry' | 'revealed' | 'locked'`; `next` marks completion after the final question.

- [x] **Step 5: Run the store test and verify GREEN**

Run: `npm test -- --run src/store/questionTypePracticeStore.test.ts`

Expected: all store tests pass.

### Task 2: Replace the home page and add the detail page

**Files:**
- Modify: `math-app/src/pages/HomePage.test.tsx`
- Create: `math-app/src/pages/QuestionTypeDetailPage.test.tsx`
- Modify: `math-app/src/pages/HomePage.tsx`
- Create: `math-app/src/pages/QuestionTypeDetailPage.tsx`
- Modify: `math-app/src/App.tsx`

- [x] **Step 1: Write failing page tests**

Home assertions:

```ts
expect(screen.getAllByRole('link', { name: /20 以内加法/ })).toHaveLength(1)
expect(screen.queryByText('406')).not.toBeInTheDocument()
expect(screen.queryByText('我的算数地图')).not.toBeInTheDocument()
```

Detail assertions:

```ts
expect(screen.getByRole('heading', { name: '20 以内加法' })).toBeInTheDocument()
expect(screen.getByText('看算式选答案')).toBeInTheDocument()
expect(screen.getByText('3 + 5 = ?')).toBeInTheDocument()
expect(screen.getByRole('link', { name: '开始答题' })).toHaveAttribute('href', '/types/addition-equation/practice')
```

- [x] **Step 2: Run focused tests and verify RED**

Run: `npm test -- --run src/pages/HomePage.test.tsx src/pages/QuestionTypeDetailPage.test.tsx`

Expected: home still shows the old modules and the detail module/route does not exist.

- [x] **Step 3: Implement the single-card home**

Keep the top bar and child chip from `useMathHome`, then render one linked `article` with the prototype title, mode, description, and `5 道题 · 四选一 · 可重试一次`. Do not render daily plan, map modules, review note, or broad atlas entry.

- [x] **Step 4: Implement the detail route**

Render the learning goal, rules, a non-interactive `3 + 5 = ?` example with four option cells, session note, and a `开始答题` link. Add `/types/addition-equation` to `AppRoutes`.

- [x] **Step 5: Run focused tests and verify GREEN**

Run: `npm test -- --run src/pages/HomePage.test.tsx src/pages/QuestionTypeDetailPage.test.tsx`

Expected: all home and detail tests pass.

### Task 3: Add local practice and result pages

**Files:**
- Create: `math-app/src/pages/QuestionTypePracticePage.test.tsx`
- Create: `math-app/src/pages/QuestionTypeResultPage.test.tsx`
- Create: `math-app/src/pages/QuestionTypePracticePage.tsx`
- Create: `math-app/src/pages/QuestionTypeResultPage.tsx`
- Modify: `math-app/src/App.tsx`

- [x] **Step 1: Write failing interaction tests**

Practice tests cover starting at `3 + 5`, first-wrong feedback, second-wrong answer reveal, correct progression, and navigation to `/types/addition-equation/result` after question five. Result tests set `completed: true, score: 4`, assert `答对 4 / 5 题`, and verify an incomplete direct visit shows `先完成一次练习`.

- [x] **Step 2: Run focused tests and verify RED**

Run: `npm test -- --run src/pages/QuestionTypePracticePage.test.tsx src/pages/QuestionTypeResultPage.test.tsx`

Expected: FAIL because both pages are missing.

- [x] **Step 3: Implement the practice page**

Render progress marks, equation, four option buttons, feedback, and a next button after the question is settled. Call `choose`, preserve the question on `retry`, and call `next`; navigate to the result route when `completed` becomes true. Do not render TTS or make network requests.

- [x] **Step 4: Implement the result page and routes**

Render score when `completed` is true. `再练一次` calls `start()` and links to practice; `回到题型详情` links to detail. Otherwise render an explanation and a link to the detail page. Register both practice and result routes.

- [x] **Step 5: Run focused tests and verify GREEN**

Run: `npm test -- --run src/pages/QuestionTypePracticePage.test.tsx src/pages/QuestionTypeResultPage.test.tsx`

Expected: all practice and result tests pass.

### Task 4: Remove the broad atlas, finish styling, and verify

**Files:**
- Delete: `math-app/src/content/questionTypes.ts`
- Delete: `math-app/src/pages/QuestionTypesPage.tsx`
- Delete: `math-app/src/pages/QuestionTypesPage.test.tsx`
- Modify: `math-app/src/styles/app.css`
- Modify: `math-app/src/App.tsx`

- [x] **Step 1: Remove the obsolete `/types` catalog route and files**

The exact route `/types` must no longer render a catalog. Only `/types/addition-equation`, `/practice`, and `/result` remain for the prototype.

- [x] **Step 2: Add prototype-specific styles**

Add focused classes for the single home card, detail example, practice feedback/next state, and result. Preserve existing tokens, notebook grid, keyboard focus, mobile breakpoints, and reduced-motion behavior. Remove unused atlas CSS selectors.

- [x] **Step 3: Run the full test suite**

Run: `npm test`

Expected: all test files pass with zero failures.

- [x] **Step 4: Run the production build**

Run: `npm run build`

Expected: TypeScript and Vite complete with exit code 0.

- [x] **Step 5: Inspect the running frontend**

Verify `http://localhost:19142/`, the detail route, a wrong-answer retry, and the result route at desktop and 390px width. Confirm the browser console has no errors.
