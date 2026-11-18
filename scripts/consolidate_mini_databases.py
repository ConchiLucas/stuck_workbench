#!/usr/bin/env python3
"""One-time, guarded Mini consolidation. Private archives and logs never enter Git.

prepare restores the approved Mini snapshots into a dedicated staging database and
compares every row and sequence with the still-unmodified sources. apply atomically
archives the old target schema and restores that verified stage into study_workbench.
Never re-run apply after completion. Old databases are retained, not dropped.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import secrets
import subprocess

ROOT = Path(__file__).resolve().parents[1]
PRIVATE = ROOT / ".private/study-consolidation-20261114"
STAGE = "study_consolidation_20261114"
TARGET = "study_workbench"
ARCHIVE_SCHEMA = "legacy_study_20261114"
SOURCES = {"kid_workbench": "kid", "rob_english_word": "public", "select_english_word": "public"}


def run(args: list[str], data: bytes | None = None) -> bytes:
    p = subprocess.run(args, input=data, capture_output=True)
    if p.returncode:
        path = PRIVATE / "last-error.log"
        path.write_bytes(p.stderr)
        path.chmod(0o600)
        raise RuntimeError(f"Command failed (exit {p.returncode}); details in private last-error.log")
    return p.stdout


def docker(*args: str, data: bytes | None = None) -> bytes:
    # All arguments are fixed internal identifiers, never user text or credentials.
    import shlex
    return run(["ssh", "-o", "BatchMode=yes", "conchi@100.120.144.76",
                "/usr/local/bin/docker " + shlex.join(list(args))], data)


def sql(db: str, query: str, atomic: bool = False) -> str:
    args = ["exec", "-i", "postgres16", "psql", "-X", "-U", "conchi", "-d", db,
            "-At", "-v", "ON_ERROR_STOP=1"]
    if atomic:
        args.append("-1")
    return docker(*args, data=query.encode()).decode()


def save(name: str, value: object) -> None:
    path = PRIVATE / name
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")
    path.chmod(0o600)


def ident(value: str) -> str:
    return '"' + value.replace('"', '""') + '"'


def fingerprint(db: str, schema: str = "public") -> dict:
    tables = sql(db, f"SELECT tablename FROM pg_tables WHERE schemaname='{schema}' ORDER BY 1;").splitlines()
    queries = []
    for table in tables:
        q = (f"SELECT json_build_object('table','{table}','count',count(*),'sha',"
             "md5(coalesce(string_agg(h,'' ORDER BY h COLLATE \"C\"),''))) FROM "
             f"(SELECT md5(row_to_json(t)::text) h FROM {ident(schema)}.{ident(table)} t) x;")
        queries.append(q)
    rows = [json.loads(line) for line in sql(db, '\n'.join(queries)).splitlines()]
    result = {r.pop('table'): r for r in rows}
    seqs = sql(db, f"SELECT sequencename FROM pg_sequences WHERE schemaname='{schema}' ORDER BY 1;").splitlines()
    seq_queries = [f"SELECT json_build_object('sequence','{s}','value',last_value::text || '|' || CASE WHEN is_called THEN 't' ELSE 'f' END) FROM {ident(schema)}.{ident(s)};" for s in seqs]
    seq_rows = [json.loads(line) for line in sql(db, '\n'.join(seq_queries)).splitlines()]
    sequences = {r['sequence']: r['value'] for r in seq_rows}
    return {"tables": result, "sequences": sequences}


def assert_idle() -> None:
    active = sql("postgres", "SELECT count(*) FROM pg_stat_activity WHERE datname IN "
                 "('kid_workbench','rob_english_word','select_english_word','study_workbench') "
                 "AND pid<>pg_backend_pid();").strip()
    if active != "0":
        raise RuntimeError("Source/target has other connections; inspect and quiesce the owning service first")


def prepare() -> None:
    if (PRIVATE / "applied.json").exists():
        raise RuntimeError("Already applied; refusing to prepare another migration")
    if sql("postgres", f"SELECT count(*) FROM pg_database WHERE datname='{STAGE}';").strip() != "0":
        raise RuntimeError("Stage already exists; refusing to overwrite")
    assert_idle()
    manifest = json.loads((PRIVATE / "backups.json").read_text())
    for item in manifest:
        data = (PRIVATE / item["file"]).read_bytes()
        if hashlib.sha256(data).hexdigest() != item["sha256"]:
            raise RuntimeError("Backup archive checksum mismatch")
    sql("postgres", f"CREATE DATABASE {STAGE} TEMPLATE template0;")
    for source in SOURCES:
        data = (PRIVATE / f"mini-{source}.dump").read_bytes()
        docker("exec", "-i", "postgres16", "pg_restore", "-U", "conchi", "-d", STAGE,
               "--no-owner", "--no-acl", "--exit-on-error", "--single-transaction", data=data)
        if source == "kid_workbench":
            sql(STAGE, "ALTER SCHEMA public RENAME TO kid; CREATE SCHEMA public;")
    source_prints = {db: fingerprint(db) for db in SOURCES}
    for db, schema in SOURCES.items():
        staged = fingerprint(STAGE, schema)
        for kind in ["tables", "sequences"]:
            for name, value in source_prints[db][kind].items():
                if staged[kind].get(name) != value:
                    raise RuntimeError(f"Stage differs from source: {db}/{kind}/{name}")
    expected_names = set(source_prints['rob_english_word']['tables']) | set(source_prints['select_english_word']['tables'])
    if set(fingerprint(STAGE)['tables']) != expected_names:
        raise RuntimeError("Unexpected public tables in stage")
    before = fingerprint(TARGET)
    save("prepared.json", {"sources": source_prints, "target_before": before,
                           "stage_public": fingerprint(STAGE), "stage_kid": fingerprint(STAGE, "kid")})
    archive = docker("exec", "postgres16", "pg_dump", "-U", "conchi", "-d", STAGE, "-Fc")
    path = PRIVATE / "verified-combined.dump"
    path.write_bytes(archive)
    path.chmod(0o600)
    save("verified-archive.json", {"sha256": hashlib.sha256(archive).hexdigest(), "bytes": len(archive)})
    print("Prepared: 78 child tables + 28 English tables; every row and sequence verified")


def apply() -> None:
    if (PRIVATE / "applied.json").exists():
        raise RuntimeError("Already applied; refusing replay")
    expected = json.loads((PRIVATE / "prepared.json").read_text())
    assert_idle()
    for db in SOURCES:
        if fingerprint(db) != expected['sources'][db]:
            raise RuntimeError(f"Source changed since preparation: {db}")
    if fingerprint(TARGET) != expected['target_before']:
        raise RuntimeError("Target changed since preparation")
    if sql(TARGET, f"SELECT count(*) FROM pg_namespace WHERE nspname IN ('kid','{ARCHIVE_SCHEMA}');").strip() != "0":
        raise RuntimeError("Target schemas already exist; refusing overwrite")
    archive = (PRIVATE / "verified-combined.dump").read_bytes()
    archive_meta = json.loads((PRIVATE / "verified-archive.json").read_text())
    if hashlib.sha256(archive).hexdigest() != archive_meta['sha256']:
        raise RuntimeError("Verified archive checksum mismatch")
    credentials = {"kid": {"role": "sw_kid_runtime", "password": secrets.token_hex(32)},
                   "english": {"role": "sw_english_runtime", "password": secrets.token_hex(32)}}
    for c in credentials.values():
        if sql("postgres", f"SELECT count(*) FROM pg_roles WHERE rolname='{c['role']}';").strip() != "0":
            raise RuntimeError("Runtime role already exists; refusing replacement")
    save("runtime-credentials.json", credentials)
    for c in credentials.values():
        sql("postgres", f"CREATE ROLE {c['role']} LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION PASSWORD '{c['password']}';")
    restored_sql = run(["docker", "exec", "-i", "postgres16", "pg_restore", "--no-owner", "--no-acl", "-f", "-"], archive).decode()
    prefix = f"ALTER SCHEMA public RENAME TO {ARCHIVE_SCHEMA};\nCREATE SCHEMA public;\n"
    suffix = """
DO $$ DECLARE r record; own text; BEGIN
  FOR r IN SELECT c.relname,c.relkind,n.nspname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
           WHERE n.nspname IN ('kid','public') AND c.relkind IN ('r','p','S','v','m','f')
           ORDER BY CASE WHEN c.relkind='S' THEN 1 ELSE 0 END LOOP
    own := CASE WHEN r.nspname='kid' THEN 'sw_kid_runtime' ELSE 'sw_english_runtime' END;
    EXECUTE format('ALTER %s %I.%I OWNER TO %I', CASE r.relkind WHEN 'S' THEN 'SEQUENCE' WHEN 'v' THEN 'VIEW' WHEN 'm' THEN 'MATERIALIZED VIEW' ELSE 'TABLE' END,r.nspname,r.relname,own);
  END LOOP;
  FOR r IN SELECT p.oid::regprocedure AS signature FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='public' LOOP
    EXECUTE format('ALTER FUNCTION %s OWNER TO sw_english_runtime',r.signature);
  END LOOP;
END $$;
ALTER SCHEMA kid OWNER TO sw_kid_runtime;
ALTER SCHEMA public OWNER TO sw_english_runtime;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA kid FROM PUBLIC;
REVOKE ALL ON SCHEMA legacy_study_20261114 FROM PUBLIC;
GRANT CONNECT ON DATABASE study_workbench TO sw_kid_runtime,sw_english_runtime;
ALTER ROLE sw_kid_runtime IN DATABASE study_workbench SET search_path=kid,pg_catalog;
ALTER ROLE sw_english_runtime IN DATABASE study_workbench SET search_path=public,pg_catalog;
"""
    # psql -1 wraps archive, schema move and ownership in one atomic transaction.
    sql(TARGET, prefix + restored_sql + suffix, atomic=True)
    actual_public = fingerprint(TARGET)
    actual_kid = fingerprint(TARGET, "kid")
    legacy = fingerprint(TARGET, ARCHIVE_SCHEMA)
    if actual_public != expected['stage_public'] or actual_kid != expected['stage_kid'] or legacy != expected['target_before']:
        raise RuntimeError("Post-commit verification failed; preserve state and investigate, never replay")
    save("applied.json", {"target": "100.120.144.76:5432/study_workbench", "public": actual_public,
                          "kid": actual_kid, "legacy": legacy})
    print("Applied atomically: 106 active tables, 17 archived tables; all row/sequence fingerprints match")


def main() -> None:
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=["prepare", "apply"])
    args = parser.parse_args()
    if args.mode == "prepare":
        prepare()
    else:
        apply()


if __name__ == "__main__":
    main()
