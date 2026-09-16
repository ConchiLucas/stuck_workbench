# Math Question Type Gallery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single oversized math prototype card with a three-section gallery containing twelve compact question-type cards.

**Architecture:** Extend the existing static prototype content module with typed module and card metadata. Render cards through a focused component so linked and display-only cards share one visual structure while preserving correct semantics. Existing detail, practice, result, and server-backed routes remain unchanged.

**Tech Stack:** React 19, React Router, TypeScript, Vitest, Testing Library, CSS.

---

### Task 1: Define and validate the twelve-card catalog

**Files:**
- Modify: `math-app/src/content/questionTypePrototype.ts`
- Create: `math-app/src/content/questionTypePrototype.test.ts`

- [x] **Step 1: Write the failing catalog test**

Assert that `questionTypeGroups` contains exactly three groups, each has exactly four cards, all twelve IDs are unique, status counts are six `supported` and six `expandable`, and only `addition-equation` has `href: '/types/addition-equation'`.

```ts
const cards = questionTypeGroups.flatMap((group) => group.types)
expect(questionTypeGroups).toHaveLength(3)
expect(questionTypeGroups.map((group) => group.types.length)).toEqual([4, 4, 4])
expect(cards.filter((card) => card.status === 'supported')).toHaveLength(6)
expect(cards.filter((card) => card.href)).toEqual([
  expect.objectContaining({ id: 'addition-equation', href: '/types/addition-equation' }),
])
```

- [x] **Step 2: Run the focused test and verify RED**

Run: `npm test -- --run src/content/questionTypePrototype.test.ts`

Expected: FAIL because `questionTypeGroups` is not exported.

- [x] **Step 3: Add typed card metadata**

Add `QuestionTypeCard` and `QuestionTypeGroup` interfaces and the twelve entries defined in `docs/superpowers/specs/2026-09-02-math-question-type-gallery-design.md`. Preserve `additionEquationType` and its five questions without changing their values.

- [x] **Step 4: Run the focused test and verify GREEN**

Run: `npm test -- --run src/content/questionTypePrototype.test.ts`

Expected: all catalog tests pass.

### Task 2: Render the gallery with correct interaction semantics

**Files:**
- Create: `math-app/src/components/QuestionTypeCard.tsx`
- Modify: `math-app/src/pages/HomePage.test.tsx`
- Modify: `math-app/src/pages/HomePage.tsx`

- [x] **Step 1: Write failing home-page tests**

Assert all three group headings and twelve card titles are present, the status labels each occur six times, and exactly one card is a link to `/types/addition-equation`.

```ts
expect(screen.getByRole('heading', { name: '20 以内加法' })).toBeInTheDocument()
expect(screen.getByRole('heading', { name: '20 以内减法' })).toBeInTheDocument()
expect(screen.getByRole('heading', { name: '认识图形' })).toBeInTheDocument()
expect(screen.getAllByText('当前支持')).toHaveLength(6)
expect(screen.getAllByText('可以扩展')).toHaveLength(6)
expect(screen.getAllByRole('link')).toHaveLength(1)
```

- [x] **Step 2: Run the home test and verify RED**

Run: `npm test -- --run src/pages/HomePage.test.tsx`

Expected: FAIL because the home page still renders one oversized card.

- [x] **Step 3: Implement the reusable card**

`QuestionTypeCard` accepts one typed card. It renders a `Link` only when `href` exists; otherwise it renders an `article`. Both variants contain order, status, mark, title, description, example, and an arrow only for the linked card.

- [x] **Step 4: Replace the home hero card with grouped cards**

Map `questionTypeGroups` into three sections. Each section uses its module key as a CSS modifier and renders four `QuestionTypeCard` components. Keep the top bar and child chip; remove all single-card markup.

- [x] **Step 5: Run the home test and verify GREEN**

Run: `npm test -- --run src/pages/HomePage.test.tsx`

Expected: the gallery assertions pass.

### Task 3: Style and verify the responsive gallery

**Files:**
- Modify: `math-app/src/styles/app.css`
- Modify: `math-app/src/App.test.tsx`

- [x] **Step 1: Replace obsolete single-card styles**

Remove `.single-type-card`, `.single-type-copy`, `.single-type-equation`, and `.single-type-action`. Add `.question-type-section`, `.question-type-grid`, `.question-type-card`, status badge, mark, example, and linked-card arrow styles. Use green, coral, and purple module modifiers.

- [x] **Step 2: Add responsive rules**

Use four columns above 960px, two columns from 520px through 960px, and one column below 520px. Preserve visible focus, reduced motion, and minimum touch sizes for the one linked card.

- [x] **Step 3: Run all tests and build**

Run: `npm test`

Expected: every test passes with zero failures.

Run: `npm run build`

Expected: TypeScript and Vite complete with exit code 0.

- [x] **Step 4: Inspect desktop and mobile in the running app**

Verify `http://localhost:19142/` shows three groups, four cards per desktop row, one linked card, and no overflow at 390px. Confirm the existing detail and practice routes still work and the console has no errors.
