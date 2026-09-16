# Math Question Type Map Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a comprehensive, clearly scoped math question-type catalog to the standalone math frontend.

**Architecture:** Keep the catalog as typed frontend content because it describes the product possibility space rather than published questions. Add one route and page that filters categories client-side, while existing practice routes and server contracts remain unchanged.

**Tech Stack:** React 19, React Router, TypeScript, Vitest, Testing Library, CSS.

---

### Task 1: Lock the catalog behavior with tests

**Files:**
- Create: `math-app/src/pages/QuestionTypesPage.test.tsx`
- Modify: `math-app/src/pages/HomePage.test.tsx`

- [ ] Assert the page explains the distinction between content types and interaction modes.
- [ ] Assert the catalog exposes more than 100 named content types across at least 10 domains.
- [ ] Assert search and domain filters narrow visible cards.
- [ ] Assert current playable types are visibly labeled.
- [ ] Assert the home page links to `/types`.
- [ ] Run focused tests and confirm they fail because the page and entry do not exist.

### Task 2: Implement the typed catalog and page

**Files:**
- Create: `math-app/src/content/questionTypes.ts`
- Create: `math-app/src/pages/QuestionTypesPage.tsx`
- Modify: `math-app/src/App.tsx`

- [ ] Define domain, content-type, availability, age-band, and interaction-mode types.
- [ ] Populate a broad catalog covering number sense, operations, word problems, fractions, algebra, geometry, measurement, time and money, data and probability, logic, and practical mathematics.
- [ ] Implement domain chips, keyword search, result count, current-availability markers, and interaction-mode reference section.
- [ ] Register `/types` and run focused tests until green.

### Task 3: Promote the catalog from the home page

**Files:**
- Modify: `math-app/src/pages/HomePage.tsx`
- Modify: `math-app/src/styles/app.css`

- [ ] Add a prominent question-type index panel before the existing learning map.
- [ ] Style the index as a labeled-card drawer while preserving the notebook identity.
- [ ] Ensure responsive behavior, keyboard focus, and reduced-motion support.
- [ ] Run all frontend tests and production build.

### Task 4: Verify visually and document scope

**Files:**
- Modify: `docs/superpowers/specs/2026-09-01-math-app-server-design.md`

- [ ] Document that the catalog is a planning taxonomy and only four types are currently executable.
- [ ] Open the running app, inspect desktop and narrow layouts, and correct any visual defects.
- [ ] Report the local URL and the distinction between displayed possibilities and implemented exercises.
