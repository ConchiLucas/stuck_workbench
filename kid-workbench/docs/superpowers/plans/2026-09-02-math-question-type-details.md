# Math Question Type Details Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make all twelve math question-type cards open a complete, data-driven explanatory detail page while preserving the existing five-question addition practice.

**Architecture:** Extend the existing static catalog with one detail record per card and a discriminated example model. A dynamic route resolves the type ID and renders one shared detail page plus a focused example-preview component. Only `addition-equation` exposes the existing practice route; all other detail pages remain informational.

**Tech Stack:** React 19, React Router, TypeScript, Vitest, Testing Library, CSS.

---

### Task 1: Define the twelve complete detail records

**Files:**
- Modify: `math-app/src/content/questionTypePrototype.ts`
- Modify: `math-app/src/content/questionTypePrototype.test.ts`

- [x] **Step 1: Write the failing detail-data tests**

Add assertions that every flattened card has `href: /types/<id>`, every ID resolves through `getQuestionTypeDetail`, every detail has exactly three rules, and only `addition-equation` has `practiceHref`.

```ts
const cards = questionTypeGroups.flatMap((group) => group.types)
expect(cards.every((card) => card.href === `/types/${card.id}`)).toBe(true)
expect(cards.map((card) => getQuestionTypeDetail(card.id)?.id)).toEqual(cards.map((card) => card.id))
expect(questionTypeDetails.every((detail) => detail.rules.length === 3)).toBe(true)
expect(questionTypeDetails.filter((detail) => detail.practiceHref)).toEqual([
  expect.objectContaining({ id: 'addition-equation', practiceHref: '/types/addition-equation/practice' }),
])
```

- [x] **Step 2: Run the focused test and verify RED**

Run: `npm test -- --run src/content/questionTypePrototype.test.ts`

Expected: FAIL because the detail lookup and eleven card links do not exist.

- [x] **Step 3: Add typed detail and example data**

Define `QuestionTypeDetail` with `id`, `groupId`, `moduleTitle`, `title`, `status`, `learningGoal`, three `rules`, `example`, and optional `practiceHref`. Define a discriminated `QuestionTypeExample` union covering `choice`, `objects`, `missing`, `judgement`, `audio-shape`, `shape-name`, `shape-feature`, and `shape-sort`. Export twelve complete records and:

```ts
export function getQuestionTypeDetail(id: string) {
  return questionTypeDetails.find((detail) => detail.id === id)
}
```

Set every card `href` to `/types/<id>` without changing status labels.

- [x] **Step 4: Run the focused test and verify GREEN**

Run: `npm test -- --run src/content/questionTypePrototype.test.ts`

Expected: all catalog and detail-data assertions pass.

### Task 2: Render type-specific example previews

**Files:**
- Create: `math-app/src/components/QuestionTypeExample.tsx`
- Create: `math-app/src/components/QuestionTypeExample.test.tsx`

- [x] **Step 1: Write failing preview tests**

Render representative equation, object, judgement, and shape examples. Assert their prompt, candidates, correct annotation, and shape/category labels are present without buttons or links.

```ts
render(<QuestionTypeExample example={detail.example} />)
expect(screen.getByText(detail.example.prompt)).toBeInTheDocument()
expect(screen.queryByRole('button')).not.toBeInTheDocument()
```

- [x] **Step 2: Run the focused test and verify RED**

Run: `npm test -- --run src/components/QuestionTypeExample.test.tsx`

Expected: FAIL because the component does not exist.

- [x] **Step 3: Implement a presentational preview component**

Switch exhaustively on `example.kind`. Render semantic `div`, `span`, and list markup only. Reuse `ShapeGlyph` for geometric candidates where practical, and mark the demonstrated answer with visible text “正确答案” rather than interaction state.

- [x] **Step 4: Run the focused test and verify GREEN**

Run: `npm test -- --run src/components/QuestionTypeExample.test.tsx`

Expected: all preview variants pass.

### Task 3: Convert the detail page and route to dynamic lookup

**Files:**
- Modify: `math-app/src/pages/QuestionTypeDetailPage.tsx`
- Modify: `math-app/src/pages/QuestionTypeDetailPage.test.tsx`
- Modify: `math-app/src/pages/HomePage.test.tsx`
- Modify: `math-app/src/App.tsx`
- Modify: `math-app/src/App.test.tsx`

- [x] **Step 1: Write failing routing and page tests**

Render with `MemoryRouter initialEntries={['/types/subtraction-missing']}` and route `types/:typeId`. Assert the subtraction detail, three rules, example, informational note, and “返回题型首页” link. Add shape coverage and an invalid-ID case. Update the home test to expect twelve card links.

- [x] **Step 2: Run focused tests and verify RED**

Run: `npm test -- --run src/pages/HomePage.test.tsx src/pages/QuestionTypeDetailPage.test.tsx src/App.test.tsx`

Expected: FAIL because the route and page still use the fixed addition record.

- [x] **Step 3: Implement the shared page and dynamic route**

Use `useParams<{ typeId: string }>()`, resolve with `getQuestionTypeDetail`, and render `QuestionTypeExample`. If no detail exists, render “没有找到这个题型” and a home link. For `practiceHref`, preserve the store `start` callback and “开始答题”; otherwise show the informational note and home action.

Change the route to:

```tsx
<Route path="types/:typeId" element={<QuestionTypeDetailPage />} />
```

Keep the specific practice and result routes after it.

- [x] **Step 4: Run focused tests and verify GREEN**

Run: `npm test -- --run src/pages/HomePage.test.tsx src/pages/QuestionTypeDetailPage.test.tsx src/App.test.tsx`

Expected: all gallery, dynamic detail, invalid route, and existing practice-link assertions pass.

### Task 4: Style, verify, and inspect all detail categories

**Files:**
- Modify: `math-app/src/styles/app.css`

- [x] **Step 1: Add module-aware detail and preview styles**

Add detail modifiers `.type-detail-page.addition`, `.subtraction`, and `.shape` using local `--type-accent` and `--type-pale` variables. Style preview prompts, choices, object groups, judgement rows, shape candidates, and classification buckets. Keep previews visibly static with no hover or pointer affordance.

- [x] **Step 2: Add responsive rules**

At 760px keep the existing single-column detail layout and move the preview first. At 519px collapse four-choice and category grids to two columns or one column as appropriate, with no horizontal overflow.

- [x] **Step 3: Run complete verification**

Run: `npm test -- --run`

Expected: every test passes with zero failures.

Run: `npm run build`

Expected: TypeScript and Vite finish with exit code 0.

Run: `git diff --check`

Expected: no whitespace errors.

- [x] **Step 4: Inspect the running application**

At `http://localhost:19142/`, verify twelve linked cards. Open representative addition, subtraction, and shape details, confirm their distinct previews and correct bottom actions, confirm the existing addition practice begins, test an unknown ID, and inspect a detail page at 390 × 844 without horizontal overflow.
