#!/usr/bin/env python3
"""Poem five-end acceptance data. Uses normal plan/answer APIs.

Does not insert complete history snapshots as the qualified freeze path.
Does not modify real-child ledgers.
"""
import hashlib
import json
import uuid
import urllib.error
import urllib.request
from pathlib import Path

TAG = 'poem-alignment-v1'
CHILD = '古诗验收演示（非真实记录）'
EMPTY_CHILD = '古诗验收空账本（非真实记录）'
TASK_TITLE = '古诗验收演示题包（非真实记录）'
REAL_CHILD = '卢沁一'
OUT = Path('docs/verification/poem-alignment')
LOCK = 9142028
POEM = 'http://localhost:19161'
CONTENT = 'http://localhost:19091'
TASK = 'http://localhost:19201'
PROGRESS = 'http://localhost:19081'
KNOWLEDGE = 'http://localhost:19211'
RUN = uuid.uuid4().hex[:10]
KINDS = ['title', 'fill', 'couplet', 'recite']


def sql(query):
    import subprocess
    result = subprocess.run(
        ['docker', 'exec', '-i', 'kid-workbench-postgres-1', 'psql',
         '-X', '-qAt', '-v', 'ON_ERROR_STOP=1', '-U', 'conchi', '-d', 'study_workbench'],
        input=query, text=True, capture_output=True)
    if result.returncode:
        raise RuntimeError(result.stderr)
    return result.stdout.strip()


def literal(value):
    return "'" + str(value).replace("'", "''") + "'"


def get_json(url):
    try:
        with urllib.request.urlopen(url, timeout=90) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        raise RuntimeError((url, exc.code, exc.read().decode(errors='replace'))) from exc


def get_bytes(url):
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            return response.read(), response.headers.get_content_type()
    except urllib.error.HTTPError as exc:
        raise RuntimeError((url, exc.code, exc.read()[:200])) from exc


def post_json(url, payload, timeout=120):
    request = urllib.request.Request(
        url, data=json.dumps(payload).encode(), method='POST',
        headers={'Content-Type': 'application/json'})
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        raise RuntimeError((url, exc.code, exc.read().decode(errors='replace'))) from exc


def unwrap(body):
    if isinstance(body, dict) and 'data' in body and 'error' in body:
        if body.get('error'):
            raise RuntimeError(body['error'])
        return body['data']
    return body


def ensure_child(name):
    sql(f"INSERT INTO children(name,grade) SELECT {literal(name)},'验收演示' WHERE NOT EXISTS (SELECT 1 FROM children WHERE name={literal(name)});")
    return int(sql(f"SELECT id FROM children WHERE name={literal(name)};").splitlines()[-1])


def fingerprint():
    names = ','.join(literal(name) for name in (CHILD, EMPTY_CHILD))
    return sql(f"""SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.id)::text,''))
      FROM attempts t JOIN children c ON c.id=t.child_id WHERE c.name NOT IN ({names});""")


def client(suffix):
    return f'{TAG}:{RUN}:{suffix}'


def wait_health(url):
    last = None
    for _ in range(30):
        try:
            get_json(url)
            return
        except Exception as exc:
            last = exc
            import time
            time.sleep(2)
    raise RuntimeError((url, last))


def same_selected(kind, left, right):
    if left == right:
        return True
    if kind != 'recite':
        return (left or '') == (right or '')
    try:
        a = json.loads(left) if isinstance(left, str) and str(left).startswith('{') else {}
        b = json.loads(right) if isinstance(right, str) and str(right).startswith('{') else {}
    except json.JSONDecodeError:
        return False
    return list(a.get('sequence') or []) == list(b.get('sequence') or [])


def frozen_urls(example):
    url = (example or {}).get('speechUrl') or ''
    if url.startswith('/api/v1/poem/task-media/'):
        return [url]
    return []


def task_ok(task):
    items = task.get('items') or []
    by_kind = {}
    frozen_speech = False
    for q in items:
        by_kind.setdefault(q.get('kind'), []).append(q)
        example = q.get('example') or {}
        url = example.get('speechUrl') or ''
        if '/poem/items/' in url or (url.startswith('/api/v1/poem/') and not url.startswith('/api/v1/poem/task-media/')):
            return False
        if url.startswith('/api/v1/poem/task-media/'):
            frozen_speech = True
    return frozen_speech and all(len(by_kind.get(kind, [])) >= 3 for kind in KINDS)


def ensure_task():
    existing = sql(f"SELECT COALESCE(json_agg(id ORDER BY id),'[]') FROM poem_question_tasks WHERE title={literal(TASK_TITLE)};")
    ids = json.loads(existing) if existing else []
    for task_id in ids:
        task = get_json(f'{TASK}/api/v1/poem/question-tasks/{task_id}')
        if task_ok(task):
            return task
    return post_json(f'{TASK}/api/v1/poem/question-tasks', {'title': TASK_TITLE, 'count': 12, 'types': KINDS}, timeout=180)


def create_plan(child_id):
    return unwrap(post_json(f'{POEM}/api/v1/children/{child_id}/poem/plans', {'types': KINDS, 'count': 12}))


def answer(child_id, plan_id, item_id, payload):
    body = {'clientId': payload['clientId'], 'costMs': 900}
    body.update({k: v for k, v in payload.items() if k != 'clientId'})
    return unwrap(post_json(
        f'{POEM}/api/v1/children/{child_id}/poem/plans/{plan_id}/items/{item_id}/answer',
        body))


def snapshot_of(item_id):
    return json.loads(sql(f"SELECT question_snapshot::text FROM plan_items WHERE id={int(item_id)};"))


def wrong_payload(example):
    kind = example['kind']
    if kind == 'recite':
        order = list(example['correctSequence'])
        if len(order) > 1:
            order[0], order[1] = order[1], order[0]
        return {'sequence': order}
    answer_id = example['answerId']
    other = next(o['id'] for o in example['options'] if o['id'] != answer_id)
    return {'selectedId': other}


def right_payload(example):
    kind = example['kind']
    if kind == 'recite':
        return {'sequence': list(example['correctSequence'])}
    return {'selectedId': example['answerId']}


def existing_plan(child_id):
    raw = sql(f"""
      SELECT p.id
      FROM study_plans p
      WHERE p.child_id={int(child_id)} AND p.subject_code='poem' AND p.target_count>=12
        AND (SELECT COUNT(*) FROM attempts a JOIN plan_items i ON a.plan_item_id=i.id WHERE i.plan_id=p.id)>=8
      ORDER BY p.id DESC LIMIT 1""")
    return int(raw) if raw else 0


def item_by_kind(plan, kind):
    for item in plan['items']:
        example = item.get('example') or {}
        if example.get('kind') == kind or item.get('question', {}).get('code') == kind:
            return item
    raise RuntimeError(f'plan missing {kind}')


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    wait_health(f'{CONTENT}/healthz')
    wait_health(f'{POEM}/healthz')
    wait_health(f'{TASK}/healthz')
    wait_health(f'{PROGRESS}/healthz')
    wait_health(f'{KNOWLEDGE}/healthz')
    baseline = fingerprint()
    materials = get_json(f'{CONTENT}/api/v1/poem/items?view=groups')
    total = materials.get('total') or 0
    if total < 5:
        raise RuntimeError(f'need poem materials, have {total}')
    task = ensure_task()
    by_kind = {}
    for q in task.get('items') or []:
        by_kind.setdefault(q['kind'], []).append(q)
    for kind in KINDS:
        if len(by_kind.get(kind, [])) < 3:
            raise RuntimeError(f'task missing 3 {kind} items: { {k: len(v) for k, v in by_kind.items()} }')

    child_id = ensure_child(CHILD)
    empty_id = ensure_child(EMPTY_CHILD)
    empty_detail = unwrap(get_json(f'{PROGRESS}/api/v1/children/{empty_id}/overview'))
    sql(f"BEGIN; SELECT pg_advisory_xact_lock({LOCK}); COMMIT;")

    reused_id = existing_plan(child_id)
    if reused_id:
        plan = unwrap(get_json(f"{POEM}/api/v1/children/{child_id}/poem/plans/{reused_id}"))
    else:
        plan = create_plan(child_id)
    if not any(frozen_urls((snapshot_of(it['id']).get('example') or {})) for it in plan['items']):
        plan = create_plan(child_id)
        reused_id = 0
    plans = [{'planId': plan['plan']['id'], 'itemIds': [it['id'] for it in plan['items']], 'reused': bool(reused_id)}]
    attempts = []
    media_checks = []
    first_frozen = None
    for item in plan['items']:
        snap = snapshot_of(item['id'])
        for url in frozen_urls(snap.get('example') or {}):
            digest = url.split('/')[-1].split('.')[0]
            frozen, _ = get_bytes(POEM + url)
            task_bytes, _ = get_bytes(TASK + url)
            progress_bytes, _ = get_bytes(PROGRESS + url.replace('/api/v1/poem/', '/api/poem/'))
            assert hashlib.sha256(frozen).hexdigest() == digest
            assert frozen == task_bytes == progress_bytes
            media_checks.append({'item': item['id'], 'sha256': digest, 'bytes': len(frozen)})
            if first_frozen is None:
                first_frozen = {'itemId': item['id'], 'url': url, 'snap': snap}
    assert first_frozen, 'plan missing frozen speech'
    for kind in KINDS:
        item = item_by_kind(plan, kind)
        snap = snapshot_of(item['id'])
        assert snap.get('schema') == 1, snap
        assert snap.get('kind') == kind, snap
        assert not snap.get('selected')
        example = snap['example']
        existing = json.loads(sql(f"SELECT COALESCE(json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id),'[]') FROM attempts WHERE child_id={child_id} AND plan_item_id={item['id']};"))
        if not existing:
            answer(child_id, plan['plan']['id'], item['id'], {'clientId': client(f'{kind}-wrong'), **wrong_payload(example)})
            answer(child_id, plan['plan']['id'], item['id'], {'clientId': client(f'{kind}-right'), **right_payload(example)})
            existing = json.loads(sql(f"SELECT json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id) FROM attempts WHERE child_id={child_id} AND plan_item_id={item['id']};"))
        assert len(existing) == 2 and not existing[0]['correct'] and existing[1]['correct'], existing
        progress = unwrap(get_json(f"{PROGRESS}/api/v1/children/{child_id}/knowledge-points/{item['kpId']}"))
        by_id = {h.get('attempt_id'): h for h in progress.get('history') or []}
        for saved in existing:
            review = by_id[saved['id']]['poem_review']
            assert same_selected(kind, review.get('selected'), saved['selected']), (kind, review, saved)
            evidence = get_json(f"{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{saved['id']}")
            assert same_selected(kind, (evidence.get('response') or {}).get('selectedOptionId'), saved['selected'])
            if kind in ('title', 'fill', 'couplet', 'recite') and not saved['correct']:
                assert review.get('facts'), review
            urls = frozen_urls(example)
            if evidence.get('mediaFidelity') == 'immutable' and urls:
                data, _ = get_bytes(f"{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{saved['id']}/media/stem-audio")
                assert hashlib.sha256(data).hexdigest() == urls[0].split('/')[-1].split('.')[0]
            (OUT / f"attempt-{saved['id']}.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2))
        attempts.append({'kind': kind, 'item': item['id'], 'kpId': item['kpId'], 'attempts': existing, 'optionOrder': item.get('optionOrder')})

    fail_kp = int(sql("SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='poem' ORDER BY kp.id LIMIT 1;"))
    sql(f"""INSERT INTO attempts(child_id,kp_id,is_correct,cost_ms,source,client_id,created_at)
        SELECT {child_id},{fail_kp},false,800,'quiz',{literal(TAG + ':fail-unlinked')},NOW()
        WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE client_id={literal(TAG + ':fail-unlinked')});""")
    fail_id = int(sql(f"SELECT id FROM attempts WHERE client_id={literal(TAG + ':fail-unlinked')};"))
    fail_evidence = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{fail_id}')
    assert 'unlinked_plan_item' in (fail_evidence.get('evidenceReasonCodes') or [])
    (OUT / f'fail-unlinked-{fail_id}.json').write_text(json.dumps({'qualified': False, **fail_evidence}, ensure_ascii=False, indent=2))

    live_client = TAG + ':fail-live-media'
    live_snap = json.dumps({
        'schema': 1, 'kind': 'title', 'skillCode': 'title', 'responseKind': 'title',
        'example': {
            'kind': 'title', 'prompt': '这首诗叫什么？', 'line': '床前明月光', 'workId': 'pm001',
            'options': [{'id': 'pm002', 'label': '春晓'}, {'id': 'pm001', 'label': '静夜思'}],
            'answerId': 'pm001',
            'speechUrl': f'/api/v1/poem/items/{fail_kp}/speech/1.wav',
        },
    }, ensure_ascii=False)
    sql(f"""
      WITH q AS (
        SELECT id FROM questions WHERE kp_id={fail_kp} AND code IN ('title','fill') ORDER BY id LIMIT 1
      ),
      p AS (
        INSERT INTO study_plans(child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count)
        SELECT {child_id},CURRENT_DATE,
               COALESCE((SELECT MAX(seq_no) FROM study_plans WHERE child_id={child_id} AND plan_date=CURRENT_DATE),0)+1,
               'poem','done',1,1,0
        WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE client_id={literal(live_client)})
        RETURNING id
      ),
      i AS (
        INSERT INTO plan_items(plan_id,seq,kp_id,question_id,bucket,status,tries,picks,question_snapshot)
        SELECT p.id,1,{fail_kp},q.id,'new','wrong',1,{literal('pm002')},{literal(live_snap)} FROM p, q
        RETURNING id, question_id
      )
      INSERT INTO attempts(child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected)
      SELECT {child_id},{fail_kp},i.question_id,false,700,'quiz',{literal(live_client)},NOW(),i.id,'pm002' FROM i;
    """)
    live_id = int(sql(f"SELECT id FROM attempts WHERE client_id={literal(live_client)};"))
    live_evidence = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{live_id}')
    assert live_evidence.get('mediaFidelity') != 'immutable'
    assert 'media_not_frozen' in (live_evidence.get('evidenceReasonCodes') or [])
    (OUT / f'fail-live-media-{live_id}.json').write_text(json.dumps({'qualified': False, **live_evidence}, ensure_ascii=False, indent=2))

    source_replacement = None
    if first_frozen:
        freeze_snap = first_frozen['snap']
        source_url = next((url for url in (freeze_snap.get('mediaSHA256') or {}) if '/poem/items/' in url), None)
        if not source_url:
            raise RuntimeError('frozen snapshot missing live source URL')
        parts = source_url.split('/')
        source_kp = int(parts[5])
        source_ord = int(parts[7].split('.')[0])
        source_hex = sql(f"SELECT encode(data,'hex') FROM poem_item_media WHERE kp_id={source_kp} AND kind='speech' AND ord={source_ord};")
        other = sql(f"SELECT kp_id||' '||ord||' '||encode(data,'hex') FROM poem_item_media WHERE NOT (kp_id={source_kp} AND ord={source_ord}) AND kind='speech' ORDER BY kp_id,ord LIMIT 1;")
        other_kp, other_ord, other_hex = other.split(' ', 2)
        source_sha = hashlib.sha256(bytes.fromhex(source_hex)).hexdigest()
        other_sha = hashlib.sha256(bytes.fromhex(other_hex)).hexdigest()
        sql(f"UPDATE poem_item_media SET data=decode({literal(other_hex)},'hex'), sha256={literal(other_sha)} WHERE kp_id={source_kp} AND kind='speech' AND ord={source_ord};")
        changed = sql(f"SELECT sha256 FROM poem_item_media WHERE kp_id={source_kp} AND kind='speech' AND ord={source_ord};")
        assert changed != source_sha
        frozen_url = first_frozen['url']
        frozen_digest = frozen_url.split('/')[-1].split('.')[0]
        after, _ = get_bytes(POEM + frozen_url)
        progress_after, _ = get_bytes(PROGRESS + frozen_url.replace('/api/v1/poem/', '/api/poem/'))
        task_after, _ = get_bytes(TASK + frozen_url)
        assert hashlib.sha256(after).hexdigest() == frozen_digest
        assert after == progress_after == task_after
        sql(f"UPDATE poem_item_media SET data=decode({literal(source_hex)},'hex'), sha256={literal(source_sha)} WHERE kp_id={source_kp} AND kind='speech' AND ord={source_ord};")
        source_replacement = {'kpId': source_kp, 'ord': source_ord, 'sourceChanged': True, 'historyUnchanged': True, 'sha256': frozen_digest}

    assert baseline == fingerprint(), 'other children changed'
    empty_hist = unwrap(get_json(f'{PROGRESS}/api/v1/children/{empty_id}/knowledge-points/{fail_kp}'))
    assert empty_hist['attempts'] == 0
    assert not empty_hist.get('history')

    manifest = {
        'qualified': True,
        'childId': child_id,
        'emptyChildId': empty_id,
        'taskId': task['id'],
        'plans': plans,
        'attempts': attempts,
        'mediaChecks': media_checks,
        'sourceReplacement': source_replacement,
        'failSamples': [
            {'id': fail_id, 'qualified': False, 'reason': 'unlinked plan item'},
            {'id': live_id, 'qualified': False, 'reason': 'live speech url not frozen'},
        ],
        'protectedAttemptsUnchanged': True,
        'emptyOverviewAttempts': empty_detail.get('today', {}).get('attempts', 0),
        'realChild': REAL_CHILD,
        'note': '脚本通过不等于用户验收通过。当前读音是可冻结的真实 WAV 字节，不是人类朗诵。',
    }
    (OUT / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
    print(json.dumps({'child': child_id, 'task': task['id'], 'plan': plan['plan']['id'], 'mediaChecks': len(media_checks)}, ensure_ascii=False))


if __name__ == '__main__':
    main()
