# stuck_workbench database boundary

- User approved consolidating this workspace into Mac mini PostgreSQL
  `100.120.144.76:5432/study_workbench` on 2026-11-14 and explicitly chose Mini
  data as authoritative. This supersedes child documents describing separate
  `rob_english_word`, `select_english_word`, or `kid_workbench` runtime databases.
- Latest user instruction: all business tables use `public` (2026-11-14).
  Both child (`sw_kid_runtime`) and English (`sw_english_runtime`) use the same
  `study_workbench.public`. Child accounts are `kid_users`; English accounts remain
  `users`. Never merge these identities or change their distinct structures.
- The old target's 17 tables are retained as `public.legacy_study_*`; they are
  maintenance-only archives, not runtime fallbacks. The former `kid` and
  `legacy_study_20261114` schemas are removed only after all objects have moved.
  Public is maintenance-owned; runtime access is restricted by table ownership and
  privileges. Sharing a schema does not authorize reading another application's tables.
- Private passwords belong only in ignored `.env`, `.env.local`, `config.yaml`,
  and `.private/` files with mode 0600. Do not print configuration or connection
  strings. No implicit fallback to MacBook PostgreSQL or SQLite in normal operation.
- Child Compose no longer starts PostgreSQL or seeds automatically. The `seed`
  service has the `maintenance` profile; do not run it against production as a check.
- `scripts/consolidate_mini_databases.py` is a one-time guarded maintenance tool.
  Do not replay after application or drop source databases/archives without explicit
  authorization. Temporary stage cleanup is limited to this migration's own stage.
  `scripts/flatten_public_schema.py` is the separately rehearsed one-time follow-up;
  never replay either tool after success.
- Do not copy configuration to other Mini checkouts or deploy their services as
  part of ordinary work in this checkout. Redis, MinIO, models, and other projects'
  databases are separate dependencies and are not part of this PostgreSQL move.
- See `docs/PUBLIC_SCHEMA_20261114.md` for the latest schema move and
  `docs/MINI_DATABASE_CONSOLIDATION_20261114.md` for the earlier database consolidation.
  These record actual verification and
  deployment status. Developer self-checks are not an independent review.
