# Pinyin Runtime Quiz Generation

## Goal
Make the four quiz cards on `http://localhost:19091/pinyin` request a newly generated question from database-backed material on every entry.

## Scope
- Content-admin database schema and initial blend material.
- Content-admin runtime generator and HTTP preview endpoint.
- Content-admin frontend integration with recent-target exclusion.
- Verification on the running 19091 service.
- Formal pinyin-server plan integration is a follow-up phase because its persisted plan snapshots require a separate migration of mastery skills and plan selection.

## Phases
- [complete] Phase 1: Record design and inspect backend/router/test patterns.
- [complete] Phase 2: Add failing generator tests and database-backed implementation.
- [complete] Phase 3: Add failing HTTP tests and expose generation endpoint.
- [complete] Phase 4: Add failing frontend tests and replace local/fake generation with API calls.
- [complete] Phase 5: Run full backend/frontend verification, rebuild 19091, and visually test all four types.

## Decisions
- Questions are generated at request time; source material remains in database tables.
- The API accepts recent target IDs so repeated entries avoid immediate repetition.
- Shape/blend answer choices expose audio metadata, while the UI hides answer text.
- Blend uses curated valid syllable rows rather than combining every initial and final.

## Errors Encountered
| Error | Attempt | Resolution |
|---|---:|---|
| Go test could not write the default macOS build cache inside the sandbox | 1 | Re-run with a task-specific `GOCACHE` under `/private/tmp`. |
| CSS patch context contained an incorrect mobile-rule fragment | 1 | Inspected the exact selectors and reapplied only against verified lines. |
| Full Go suite could not open localhost ports used by existing `httptest` cases | 1 | Re-run the unchanged suite with sandbox escalation; focused generator and route tests already pass without listeners. |
