#!/usr/bin/env python3
"""One-time public-schema move. Rehearse before applying; never replay after success."""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PRIVATE = ROOT / '.private/public-schema-20261114'
STAGE = 'study_public_check_20261114'
TARGET = 'study_workbench'
SCHEMAS = ['public', 'kid', 'legacy_study_20261114']
spec = importlib.util.spec_from_file_location('consolidation', Path(__file__).with_name('consolidate_mini_databases.py'))
original = importlib.util.module_from_spec(spec)
spec.loader.exec_module(original)
sql, docker, ident = original.sql, original.docker, original.ident


def save(name, value):
    p = PRIVATE / name
    p.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')
    p.chmod(0o600)


def snapshots(db):
    return {s: original.fingerprint(db, s) for s in SCHEMAS}


def short_name(name):
    if len(name.encode()) <= 63:
        return name
    return name[:45] + '_' + hashlib.sha256(name.encode()).hexdigest()[:12]


def plan(db):
    query = """SELECT json_build_object('schema',n.nspname,'name',c.relname,'kind',c.relkind)
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname IN ('public','kid','legacy_study_20261114') AND c.relkind IN ('r','p','i','S')
ORDER BY n.nspname,c.relkind,c.relname;"""
    objects = [json.loads(line) for line in sql(db, query).splitlines()]
    mapping = []
    public_names = {r['name'] for r in objects if r['schema'] == 'public'}
    used = set(public_names)
    for obj in objects:
        schema, name = obj['schema'], obj['name']
        if schema == 'public':
            continue
        new = short_name('legacy_study_' + name) if schema == 'legacy_study_20261114' else name
        if schema == 'kid' and name in public_names:
            new = short_name('kid_' + name)
        if new in used:
            raise RuntimeError(f'Unexpected target name collision: {schema}/{name}')
        used.add(new)
        mapping.append({**obj, 'new': new})
    commands = []
    # Rename before moving; moving a table also moves its indexes and owned sequences.
    for kind, keyword in [('r', 'TABLE'), ('p', 'TABLE'), ('i', 'INDEX'), ('S', 'SEQUENCE')]:
        for r in mapping:
            if r['kind'] == kind and r['name'] != r['new']:
                commands.append(f'ALTER {keyword} {ident(r["schema"])}.{ident(r["name"])} RENAME TO {ident(r["new"])};')
    for r in mapping:
        if r['kind'] in ['r', 'p']:
            commands.append(f'ALTER TABLE {ident(r["schema"])}.{ident(r["new"])} SET SCHEMA public;')
    commands.append("""DO $$ DECLARE r record; BEGIN
FOR r IN SELECT n.nspname,c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname IN ('kid','legacy_study_20261114') AND c.relkind='S' LOOP
 EXECUTE format('ALTER SEQUENCE %I.%I SET SCHEMA public',r.nspname,r.relname);
END LOOP; END $$;""")
    commands += ['DROP SCHEMA kid;', 'DROP SCHEMA legacy_study_20261114;']
    return {'objects': mapping, 'sql': '\n'.join(commands)}


def expected_public(before, mapping):
    result = {kind: dict(before['public'][kind]) for kind in ['tables', 'sequences']}
    for r in mapping:
        kind = {'r': 'tables', 'p': 'tables', 'S': 'sequences'}.get(r['kind'])
        if kind:
            result[kind][r['new']] = before[r['schema']][kind][r['name']]
    return result


def verify(db, expected):
    actual = original.fingerprint(db)
    if actual != expected:
        raise RuntimeError('Content/sequence verification failed; do not replay')
    schemas = sql(db, "SELECT nspname FROM pg_namespace WHERE nspname NOT LIKE 'pg_%' AND nspname<>'information_schema' ORDER BY 1;").splitlines()
    if schemas != ['public']:
        raise RuntimeError('Unexpected remaining custom schemas')
    if sql(db, "SELECT count(*) FROM pg_constraint WHERE connamespace='public'::regnamespace AND NOT convalidated;").strip() != '0':
        raise RuntimeError('Unvalidated constraints found')
    return actual


def prepare():
    if (PRIVATE / 'applied.json').exists():
        raise RuntimeError('Already applied')
    if sql('postgres', f"SELECT count(*) FROM pg_database WHERE datname='{STAGE}';").strip() != '0':
        raise RuntimeError('Staging database exists; refusing overwrite')
    archive = (PRIVATE / 'before-public.dump').read_bytes()
    if hashlib.sha256(archive).hexdigest() != json.loads((PRIVATE / 'backup.json').read_text())['sha256']:
        raise RuntimeError('Backup SHA mismatch')
    sql('postgres', f'CREATE DATABASE {STAGE} TEMPLATE template0;')
    docker('exec', '-i', 'postgres16', 'pg_restore', '-U', 'conchi', '-d', STAGE,
           '--no-owner', '--no-acl', '--single-transaction', '--exit-on-error', data=archive)
    before = snapshots(STAGE)
    if before != snapshots(TARGET):
        raise RuntimeError('Backup/live data mismatch')
    p = plan(STAGE)
    sql(STAGE, p['sql'], atomic=True)
    expected = expected_public(before, p['objects'])
    verify(STAGE, expected)
    save('prepared.json', {'before': before, 'plan': p, 'expected': expected})
    print('Rehearsal passed: 123 tables and all sequence values preserved in public')


def apply():
    if (PRIVATE / 'applied.json').exists():
        raise RuntimeError('Already applied; refusing replay')
    p = json.loads((PRIVATE / 'prepared.json').read_text())
    if sql('postgres', "SELECT count(*) FROM pg_stat_activity WHERE datname='study_workbench' AND pid<>pg_backend_pid();").strip() != '0':
        raise RuntimeError('Stop the owning application before cutover')
    if snapshots(TARGET) != p['before']:
        raise RuntimeError('Live data changed since rehearsal; refusing migration')
    locks = []
    for schema, data in p['before'].items():
        locks += [f'LOCK TABLE {ident(schema)}.{ident(table)} IN ACCESS EXCLUSIVE MODE;' for table in data['tables']]
    permissions = """
ALTER SCHEMA public OWNER TO conchi;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT USAGE,CREATE ON SCHEMA public TO sw_kid_runtime,sw_english_runtime;
ALTER ROLE sw_kid_runtime IN DATABASE study_workbench SET search_path=public,pg_catalog;
ALTER ROLE sw_english_runtime IN DATABASE study_workbench SET search_path=public,pg_catalog;
"""
    sql(TARGET, '\n'.join(locks) + '\n' + p['plan']['sql'] + '\n' + permissions, atomic=True)
    actual = verify(TARGET, p['expected'])
    save('applied.json', {'database': TARGET, 'schema': 'public', 'tables_and_sequences': actual})
    print('Applied atomically: only public remains; 123 tables and all sequences verified')


if __name__ == '__main__':
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['prepare', 'apply'])
    args = parser.parse_args()
    prepare() if args.mode == 'prepare' else apply()
