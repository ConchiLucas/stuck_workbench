# Mini study_workbench consolidation

> Historical first cutover. The user subsequently required all tables in `public`.
> The current schema layout is documented in [public schema migration](PUBLIC_SCHEMA_20261114.md);
> the kid/legacy schemas below describe the earlier completed phase.

User authorization: consolidate this workspace into Mini `study_workbench`, with
Mini data authoritative (2026-11-14). Status: applied and verified on 2026-11-14.

| Runtime group | Database | Schema | Runtime role |
| --- | --- | --- | --- |
| Child apps and four backends | 100.120.144.76:5432/study_workbench | kid | sw_kid_runtime |
| English game, cloze, Go admin, Python Agent | same database | public | sw_english_runtime |
| Previous study_workbench data | same database | legacy_study_20261114 | maintenance only |

The active Mini child dataset is actually `kid_workbench` (78 tables), not the
older `study_workbench` dataset (17 tables). Mini English sources have 20 game
tables and 8 administration tables. Schema separation prevents collision between
the child and English `users` tables. Both English connections use the same
physical database and public schema, including existing explicit-public SQL.

All six source archives (Mini four, MacBook two) are stored in ignored, private
`.private/study-consolidation-20261114/`, with byte counts and SHA-256 manifests.
Archives contain private data and must never be committed or published. Mini and
MacBook have divergent English data; the user chose Mini. MacBook data is retained
in its original databases and backups, not silently merged.

The prepare step restores all three approved sources to an isolated, migration-owned
stage, then compares every table's row count/content fingerprint and every sequence.
Apply repeats source/target checks and atomically archives the old public schema,
restores the verified combined archive, and sets ownership/search paths. It refuses
replay, changed sources, unexpected target schemas, or active business connections.

Passwords are generated independently for two non-superuser roles, saved privately,
and never passed as command-line arguments. Schema ownership permits each existing
application's own maintenance behavior; neither runtime role is a superuser or can
create databases/roles. Cross-schema child/English access is not granted.

Current-checkout configuration changes include remote-only child Compose, explicit
maintenance-only seeding, native backend defaults, English templates, and the Go
admin's former hardcoded cross-database connection. Python scoring now refuses a
missing DSN rather than falling back to an old local database with embedded credentials.

Original databases and existing Docker volumes are retained. Redis/MinIO settings,
media objects, other workspace databases, and other Mini source checkouts are outside
this migration. No real model/TTS jobs or learning answers are created for verification.

## Runtime commands

English: root `.env.local` supplies both logical connections to the same Mini database;
`./deploy-compose-full.sh --check` validates all six Compose definitions. Existing
services are updated selectively with `--project`; a full start also starts the Agent
and CLI runner, so it is not a harmless migration check.

Child: root `.env` supplies all Compose backends. Standalone backend Compose files
must use `docker compose --env-file ../.env ...`. Native Go processes must load/export
the private environment before starting. Normal `make up` no longer creates a local
database or runs seed data. Seed/demo commands write production data and are not tests.

## Completion evidence

- Six original private dumps retained (Mini four and MacBook two), with SHA-256
  manifests. Verified combined archive also retained. The migration-owned temporary
  database `study_consolidation_20261114` was verified against that archive and removed
  after successful cutover; no original databases or volumes were deleted.
- Mini `study_workbench`: `kid` 78 tables, `public` 28 tables, and
  `legacy_study_20261114` 17 tables. Every table's count/content fingerprint and
  every sequence matched the source/staging data after restore. The same complete
  comparison passed again after application restart and HTTP reads.
- Direct TCP authentication from MacBook passed for both newly generated runtime
  roles. Child: `current_database=study_workbench`, `current_schema=kid`, 1 user,
  2,947 questions. English: same database, `current_schema=public`, 11 users,
  22,098 cleaned words. Both roles are non-superusers and lack USAGE on the other
  application's schema and the archived schema. Neither role can create databases
  or roles. No credentials are included in this report.
- Child Compose: the original service set is preserved except for the intentionally
  removed local PostgreSQL service. All 13 business backends plus the manual seed
  definition resolve to Mini/study_workbench/kid. All three standalone backend
  Compose files pass the same check. Normal service lists exclude maintenance seed;
  no seed/demo command was executed.
- English `./deploy-compose-full.sh --check` passed for all six Compose definitions
  and existing PostgreSQL, Redis and MinIO dependencies. The private root environment
  preserves the existing Java host ports 6012/6013, Agent 6017, Go admin 6015, and
  management frontend 6016. Redis/MinIO credentials and content were not changed;
  container DNS `minio:9000` was translated to the same existing host endpoint
  `127.0.0.1:19100` for host-side preflight.
- Existing Go dashboard was rebuilt from this checkout and recreated on its original
  external `vibedeploy-shared` network. It is running with restart count 0. HTTP
  `/health` returns 200; `/users` and `/cloze-results/users` return code 0, total 11;
  `/word-libraries` returns code 0, total 81. The existing management frontend still
  returns HTTP 200. No frontend rebuild was needed.
- Originally stopped Java and Agent containers were recreated with their existing
  images and the new database environment, without starting them. Both are `created`
  and stopped, and inspection confirms both use Mini `study_workbench`. Their
  previous container metadata/logs were saved privately. No Agent loop, real model,
  TTS, answer submission, CLI runner, or child backend was started for verification.
- Go `go test ./api/v1/system ./core ./initialize` passed. Python's four relevant
  suites passed **21 tests**, including a regression proving missing scoring DSN
  cannot fall back to the former local database. Python tests were run from the
  workspace root without loading the production Agent `.env`. `git diff --check`,
  syntax checks and private-file ignore checks were also performed.

## Build and environment limitations

The ordinary Docker build failed while fetching `golang:1.26-alpine` metadata from
the configured `docker.1panel.live` mirror (403); it did not replace the running
container. The tested source was instead compiled with local Go 1.26.3 for
Linux/arm64, CGO disabled, and layered onto the existing dashboard runtime image.
The image contains the repository's non-private Docker config; database credentials
are injected through the private Compose environment. The old runtime image was
retained with tag `word-select-dashboard:before-study-20261114` for recovery evidence.

An initial recreation also found the host's Docker address pools exhausted. No
networks were pruned and no daemon/global settings changed. Go, Java and Agent
Compose now explicitly reuse their previous `vibedeploy-shared` network. The private
build context, compiled binary and commands remain under the private migration folder.
The standard builder still depends on the configured mirror being available.

No full end-to-end session or visual UI audit was performed for stopped services;
there was no UI change. Other Mini code checkouts were not copied or deployed.
Original databases remain available as historical sources; current runtime
connections use only `study_workbench`. Do not independently restart old checkouts
against those historical databases. No Git commit/push or MCP gateway onboarding
was performed. These are developer checks, not an independent review.

## Private evidence inventory

In `.private/study-consolidation-20261114/`: `backups.json`,
`verified-archive.json`, `prepared.json`, `applied.json`,
`runtime-access-check.json`, `post-start-fingerprints.json`, `compose-audit.json`,
`container-and-http-audit.json`, `stage-cleanup.json`, source archives and runtime
credentials. This directory is ignored and mode 0700; private configurations,
credential files, archives and JSON evidence are mode 0600.
Do not publish it or automatically replay the one-time migration.
