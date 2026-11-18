# study_workbench: public schema

Completed 2026-11-14 following the user's explicit instruction that all schemas use
public. This supersedes the kid/legacy schema layout in the earlier consolidation
report. Current database: **Mac mini `100.120.144.76:5432/study_workbench`**.

| Table group | Current location | Count | Runtime owner |
| --- | --- | --- | --- |
| Child learning and four administration backends | public | 78 | sw_kid_runtime |
| English game, cloze, administration and Agent | public | 28 | sw_english_runtime |
| Preserved earlier study_workbench data | public.legacy_study_* | 17 | maintenance |

There are **123 tables in public**, including the 17 preserved archival tables.
The former `kid` and `legacy_study_20261114` schemas were dropped without CASCADE
only after their objects were moved. PostgreSQL's system schemas are unchanged.

## Collision handling and application compatibility

- English accounts remain `public.users` (11 rows). Child accounts are now
  `public.kid_users` (1 row). These are distinct identities and structures.
- Child `users_id_seq` and `users_pkey` became `kid_users_id_seq` and
  `kid_users_pkey`. The parent_child foreign key still references the same child
  account table by object identity and now resolves to public.kid_users.
- All archival tables, indexes and sequences have `legacy_study_` prefixes to
  avoid collisions with active child and English objects. Their data and sequence
  values were preserved rather than merged into current learning records.
- The child User model, fresh PostgreSQL/SQLite bootstrap SQL, tests, Compose
  definitions and private environment were updated together. Existing applied
  bootstrap migration records were retained; the actual rename/move was performed
  by the explicit one-time maintenance script, not by an ordinary app startup.
- Both runtime roles now default to public. Public is maintenance-owned; each
  runtime role retains ownership of its own tables and USAGE/CREATE in public.
  Neither can SELECT the other account table or archived accounts. Sharing public
  does not grant cross-project business data access.

## Execution and verification

1. Saved a full custom-format study_workbench backup privately and recorded its
   SHA-256. All earlier source backups and databases were retained.
2. Restored that archive into migration-owned `study_public_check_20261114` and
   rehearsed the exact collision renames and schema moves in one transaction.
   Compared every table's count/content fingerprint and every sequence value.
3. Stopped only the running Go management backend, verified no other connections
   to study_workbench and no data changes since rehearsal, then applied the same
   plan with table locks in one transaction. Original object ownership and foreign
   keys were preserved. No rows were inserted/deleted/merged for the migration.
4. Verified only public remains as a custom schema and all 123 tables and sequence
   values match. All public constraints are validated. Restarted the original Go
   container without rebuilding or touching the frontend/Redis/MinIO.
5. Direct TCP authentication with both runtime roles returns current_schema=public.
   Child reads kid_users=1; English reads users=11. Both fail privilege checks for
   the other account table and legacy_study_users. The parent_child account foreign
   key resolves to kid_users.
6. Go admin HTTP /health returns 200; /users and /cloze-results/users return code 0
   with total 11; /word-libraries returns code 0 with total 81. Complete fingerprints
   were checked again after restart and HTTP reads, with no content changes.
7. All 13 child business backends plus the explicit maintenance-only seed definition
   resolve to Mini/study_workbench/public. Standalone backend configs and the private
   child environment use the same search path. No child service or AI processor was
   started, no seed/demo data was generated, and previously stopped Java/Agent
   containers remain stopped.
8. Parent backend db/model/seed tests passed. Added a regression that bootstraps
   child tables alongside an existing English users table, verifies both accounts
   survive, checks the Gorm User model selects the child account, and verifies the
   parent_child foreign key references kid_users. The db suite passed again after
   the final model assertion. Diff whitespace and Python syntax checks passed.

The temporary rehearsal database is removed only after successful final checks and
verification against the private archive; no original database or backup is deleted.
This is a developer verification, not an independent review or full UI acceptance.
No Git commit/push or changes to other Mini source checkouts were performed.

## Private evidence and replay boundary

Ignored mode-0700 `.private/public-schema-20261114/` holds the full backup,
SHA-256 manifest, exact object rename/move plan, before/after table and sequence
fingerprints, authentication/permission checks and HTTP/configuration results.
Configuration, archives and JSON evidence are mode 0600. Do not publish this folder.

`scripts/flatten_public_schema.py` refuses replay after applied.json exists. Do not
run the previous consolidation tool, reverse schema moves, re-import source data,
or drop archived tables as normal startup/cleanup steps. The previous consolidation
report remains historical evidence rather than current schema configuration.
