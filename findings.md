# Findings

- `pinyin_assets` contains 45 rows: 45 glyphs, 45 word texts/audio, 44 solo texts/audio. `eng` intentionally has no solo reading.
- `questions` currently contains 45 `inword` and 44 `listen` rows only.
- Current content-admin preview picks the first generated real example and uses hard-coded fake shape/blend samples.
- Current pinyin-server plan query only accepts `inword` and `listen`, orders candidates, and snapshots them into `plan_items`.
- Existing confusion groups are code-defined in parent-dashboard quiz generation and the content-admin frontend.
- Runtime generation should preserve plan stability later by snapshotting once when a formal plan is created.
- `db.Migrate` is the single content-admin schema bootstrap and supports both PostgreSQL and SQLite; the syllable table and idempotent seed belong there.
- Pinyin HTTP routes are grouped directly in `NewRouter`; handlers consistently return `503` when the service is absent and JSON errors otherwise.
- Router tests construct a real SQLite-backed service, so the new endpoint can be exercised without mocks.
- The initial blend seed contains 16 curated valid syllables in four families; it is inserted with `ON CONFLICT DO NOTHING` so restarts are safe.
- The generator uses cryptographic randomness, prioritizes code-defined confusion groups, falls back within the same module, and honors recent target exclusions.
- Frontend uses a request counter instead of cache invalidation: every card entry and “换一题” action creates a distinct query key and POST request.
- Recent targets are tracked per type in a three-item window and sent only in the request body, avoiding query-key feedback loops.
