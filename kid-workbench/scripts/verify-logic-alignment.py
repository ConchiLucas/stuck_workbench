#!/usr/bin/env python3
"""Logic five-end acceptance data. Uses normal plan/answer APIs.

Does not insert complete history snapshots as the qualified freeze path.
Does not modify real-child ledgers.
"""
import hashlib
import json
import uuid
import urllib.error
import urllib.request
from pathlib import Path

TAG = 'logic-alignment-v1'
CHILD = '逻辑验收演示（非真实记录）'
EMPTY_CHILD = '逻辑验收空账本（非真实记录）'
TASK_TITLE = '逻辑验收演示题包（非真实记录）'
REAL_CHILD = '卢沁一'
OUT = Path('docs/verification/logic-alignment')
LOCK = 9152026
LOGIC = 'http://localhost:19191'
CONTENT = 'http://localhost:19091'
TASK = 'http://localhost:19201'
PROGRESS = 'http://localhost:19081'
KNOWLEDGE = 'http://localhost:19211'
RUN = uuid.uuid4().hex[:10]
KINDS = ['pattern', 'classify', 'order', 'shape_reason', 'diff', 'compare']


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
    try:
        a = json.loads(left) if isinstance(left, str) and str(left).startswith('{') else {}
        b = json.loads(right) if isinstance(right, str) and str(right).startswith('{') else {}
    except json.JSONDecodeError:
        a, b = {}, {}
    if kind == 'order':
        left_seq = a.get('sequence') or ([] if not isinstance(left, str) else [p for p in left.split(',') if p])
        right_seq = b.get('sequence') or ([] if not isinstance(right, str) else [p for p in right.split(',') if p])
        return list(left_seq) == list(right_seq)
    left_id = a.get('selectedId') or left
    right_id = b.get('selectedId') or right
    return (left_id or '') == (right_id or '')


def frozen_urls(example):
    urls = []
    for url in (example.get('imageUrls') or {}).values():
        if (url or '').startswith('/api/v1/logic/task-media/'):
            urls.append(url)
    return urls


def task_ok(task):
    items = task.get('items') or []
    by_kind = {}
    frozen = False
    for q in items:
        by_kind.setdefault(q.get('kind'), []).append(q)
        example = q.get('example') or {}
        for url in (example.get('imageUrls') or {}).values():
            if not url:
                continue
            if '/logic/items/' in url or (url.startswith('/api/v1/logic/') and not url.startswith('/api/v1/logic/task-media/')):
                return False
            if url.startswith('/api/v1/logic/task-media/'):
                frozen = True
    return frozen and all(len(by_kind.get(kind, [])) >= 3 for kind in KINDS)


def ensure_task():
    existing = sql(f"SELECT COALESCE(json_agg(id ORDER BY id),'[]') FROM logic_question_tasks WHERE title={literal(TASK_TITLE)};")
    ids = json.loads(existing) if existing else []
    for task_id in ids:
        task = get_json(f'{TASK}/api/v1/logic/question-tasks/{task_id}')
        if task_ok(task):
            return task
    return post_json(f'{TASK}/api/v1/logic/question-tasks', {'title': TASK_TITLE, 'count': 18, 'types': KINDS}, timeout=180)


def create_plan(child_id):
    return unwrap(post_json(f'{LOGIC}/api/v1/children/{child_id}/logic/plans', {'types': KINDS, 'count': 18}))


def answer(child_id, plan_id, item_id, payload):
    body = {'clientId': payload['clientId'], 'costMs': 900}
    body.update({k: v for k, v in payload.items() if k != 'clientId'})
    return unwrap(post_json(
        f'{LOGIC}/api/v1/children/{child_id}/logic/plans/{plan_id}/items/{item_id}/answer',
        body))


def snapshot_of(item_id):
    return json.loads(sql(f"SELECT question_snapshot::text FROM plan_items WHERE id={int(item_id)};"))


def wrong_payload(example):
    kind = example['kind']
    if kind == 'order':
        order = list(example['correctSequence'])
        if len(order) > 1:
            order[0], order[1] = order[1], order[0]
        rejected = [{'id': order[0], 'atIndex': 0}] if order[0] != example['correctSequence'][0] else [{'id': example['correctSequence'][1], 'atIndex': 0}]
        return {'sequence': order, 'rejected': rejected}
    answer_id = example['answerId']
    other = next(oid for oid in example['options'] if oid != answer_id)
    return {'selectedId': other}


def right_payload(example):
    if example['kind'] == 'order':
        return {'sequence': list(example['correctSequence'])}
    return {'selectedId': example['answerId']}


def existing_plan(child_id):
    raw = sql(f"""
      SELECT p.id
      FROM study_plans p
      WHERE p.child_id={int(child_id)} AND p.subject_code='logic' AND p.target_count>=12
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
    wait_health(f'{LOGIC}/healthz')
    wait_health(f'{TASK}/healthz')
    wait_health(f'{PROGRESS}/healthz')
    wait_health(f'{KNOWLEDGE}/healthz')
    baseline = fingerprint()
    materials = get_json(f'{CONTENT}/api/v1/logic/items?view=table')
    items = materials.get('items') or []
    structured = [item for item in items if item.get('example')]
    if len(structured) < 18:
        raise RuntimeError(f'need structured logic materials, have {len(structured)}')
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
        plan = unwrap(get_json(f"{LOGIC}/api/v1/children/{child_id}/logic/plans/{reused_id}"))
    else:
        plan = create_plan(child_id)
    if not any(frozen_urls(snapshot_of(it['id'])) for it in plan['items']):
        plan = create_plan(child_id)
        reused_id = 0
    plans = [{'planId': plan['plan']['id'], 'itemIds': [it['id'] for it in plan['items']], 'reused': bool(reused_id)}]
    attempts = []
    media_checks = []
    first_frozen = None
    for item in plan['items']:
        snap = snapshot_of(item['id'])
        for url in frozen_urls(snap):
            digest = url.split('/')[-1].split('.')[0]
            frozen, _ = get_bytes(LOGIC + url)
            task_bytes, _ = get_bytes(TASK + url)
            progress_bytes, _ = get_bytes(PROGRESS + url.replace('/api/v1/logic/', '/api/logic/'))
            assert hashlib.sha256(frozen).hexdigest() == digest
            assert frozen == task_bytes == progress_bytes
            media_checks.append({'item': item['id'], 'sha256': digest, 'bytes': len(frozen)})
            if first_frozen is None:
                first_frozen = {'itemId': item['id'], 'url': url, 'snap': snap}
    assert first_frozen, 'plan missing frozen glyph media'
    for kind in KINDS:
        item = item_by_kind(plan, kind)
        snap = snapshot_of(item['id'])
        assert snap.get('schema') == 1, snap
        assert snap.get('kind') == kind, snap
        assert not snap.get('selected')
        example = {
            'kind': snap['kind'], 'prompt': snap['prompt'], 'rule': snap['rule'], 'objects': snap['objects'],
            'options': snap.get('options'), 'answerId': snap.get('answerId'),
            'correctSequence': snap.get('correctSequence'), 'displayOrder': snap.get('displayOrder'),
            'imageUrls': snap.get('imageUrls') or {},
        }
        existing = json.loads(sql(f"SELECT COALESCE(json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id),'[]') FROM attempts WHERE child_id={child_id} AND plan_item_id={item['id']};"))
        if not existing:
            answer(child_id, plan['plan']['id'], item['id'], {'clientId': client(f'{kind}-wrong'), **wrong_payload(example)})
            answer(child_id, plan['plan']['id'], item['id'], {'clientId': client(f'{kind}-right'), **right_payload(example)})
            existing = json.loads(sql(f"SELECT json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id) FROM attempts WHERE child_id={child_id} AND plan_item_id={item['id']};"))
        assert len(existing) == 2 and not existing[0]['correct'] and existing[1]['correct'], existing
        progress = unwrap(get_json(f"{PROGRESS}/api/v1/children/{child_id}/knowledge-points/{item['kpId']}"))
        by_id = {h.get('attempt_id'): h for h in progress.get('history') or []}
        for saved in existing:
            review = by_id[saved['id']]['logic_review']
            assert same_selected(kind, review.get('selected'), saved['selected']), (kind, review, saved)
            evidence = get_json(f"{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{saved['id']}")
            assert same_selected(kind, (evidence.get('response') or {}).get('selectedOptionId') or (evidence.get('response') or {}).get('value'), saved['selected'])
            if kind == 'order' and not saved['correct']:
                assert review.get('facts'), review
            urls = frozen_urls(example)
            if evidence.get('mediaFidelity') == 'immutable' and urls:
                object_id = next(iter((example.get('imageUrls') or {})))
                data, _ = get_bytes(f"{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{saved['id']}/media/glyph-{object_id}")
                assert hashlib.sha256(data).hexdigest() == urls[0].split('/')[-1].split('.')[0]
            (OUT / f"attempt-{saved['id']}.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2))
        attempts.append({'kind': kind, 'item': item['id'], 'kpId': item['kpId'], 'attempts': existing, 'optionOrder': item.get('optionOrder')})

    fail_kp = int(sql("SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='logic' AND kp.code='logic-classify-animal' ORDER BY kp.id LIMIT 1;"))
    sql(f"""INSERT INTO attempts(child_id,kp_id,is_correct,cost_ms,source,client_id,created_at)
        SELECT {child_id},{fail_kp},false,800,'quiz',{literal(TAG + ':fail-unlinked')},NOW()
        WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE client_id={literal(TAG + ':fail-unlinked')});""")
    fail_id = int(sql(f"SELECT id FROM attempts WHERE client_id={literal(TAG + ':fail-unlinked')};"))
    fail_evidence = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{fail_id}')
    assert 'unlinked_plan_item' in (fail_evidence.get('evidenceReasonCodes') or [])
    (OUT / f'fail-unlinked-{fail_id}.json').write_text(json.dumps({'qualified': False, **fail_evidence}, ensure_ascii=False, indent=2))

    live_client = TAG + ':fail-live-media'
    live_snap = json.dumps({
        'schema': 1, 'kind': 'classify', 'prompt': '哪个不属于这一类？',
        'rule': {'type': 'odd-one-out', 'dimension': 'kingdom', 'inGroup': 'animal', 'explain': '动物'},
        'objects': [
            {'id': 'cat', 'caption': '猫', 'glyph': 'cat', 'attrs': {'category': 'animal'}},
            {'id': 'car', 'caption': '汽车', 'glyph': 'car', 'attrs': {'category': 'vehicle'}},
        ],
        'options': ['cat', 'car'], 'answerId': 'car',
        'imageUrls': {'car': f'/api/v1/logic/items/{fail_kp}/glyph/car.svg'},
    }, ensure_ascii=False)
    sql(f"""
      WITH q AS (
        SELECT id FROM questions WHERE kp_id={fail_kp} AND code='classify' ORDER BY id LIMIT 1
      ),
      p AS (
        INSERT INTO study_plans(child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count)
        SELECT {child_id},CURRENT_DATE,
               COALESCE((SELECT MAX(seq_no) FROM study_plans WHERE child_id={child_id} AND plan_date=CURRENT_DATE),0)+1,
               'logic','done',1,1,0
        WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE client_id={literal(live_client)})
        RETURNING id
      ),
      i AS (
        INSERT INTO plan_items(plan_id,seq,kp_id,question_id,bucket,status,tries,picks,question_snapshot)
        SELECT p.id,1,{fail_kp},q.id,'new','wrong',1,{literal('{"selectedId":"cat"}')},{literal(live_snap)} FROM p, q
        RETURNING id, question_id
      )
      INSERT INTO attempts(child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected)
      SELECT {child_id},{fail_kp},i.question_id,false,700,'quiz',{literal(live_client)},NOW(),i.id,{literal('{"selectedId":"cat"}')} FROM i;
    """)
    live_id = int(sql(f"SELECT id FROM attempts WHERE client_id={literal(live_client)};"))
    live_evidence = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{live_id}')
    assert live_evidence.get('mediaFidelity') != 'immutable'
    assert 'media_not_frozen' in (live_evidence.get('evidenceReasonCodes') or [])
    (OUT / f'fail-live-media-{live_id}.json').write_text(json.dumps({'qualified': False, **live_evidence}, ensure_ascii=False, indent=2))

    rule_isolation = None
    if first_frozen:
        freeze_snap = first_frozen['snap']
        item_id = int(first_frozen['itemId'])
        kp_id = int(sql(f'SELECT kp_id FROM plan_items WHERE id={item_id};'))
        orig_payload = sql(f'SELECT payload FROM knowledge_points WHERE id={kp_id};')
        orig_visual = sql(f"SELECT visual FROM questions WHERE kp_id={kp_id} AND code={literal(freeze_snap.get('kind') or '')} ORDER BY id LIMIT 1;")
        tampered = json.loads(orig_payload) if orig_payload.startswith('{') else {'kind': freeze_snap.get('kind')}
        tampered['rule'] = {**(tampered.get('rule') or {}), 'explain': '隔离测试改写的规则', 'period': 99}
        if isinstance(tampered.get('example'), dict):
            tampered['example'] = {**tampered['example'], 'prompt': '隔离测试改写的题干', 'rule': {**(tampered['example'].get('rule') or {}), 'explain': '隔离测试改写的规则'}}
        sql(f'UPDATE knowledge_points SET payload={literal(json.dumps(tampered, ensure_ascii=False))} WHERE id={kp_id};')
        if orig_visual:
            visual_obj = json.loads(orig_visual) if orig_visual.startswith('{') else {}
            if isinstance(visual_obj.get('example'), dict):
                visual_obj['example'] = {**visual_obj['example'], 'prompt': '隔离测试改写的题干'}
            sql(f"UPDATE questions SET visual={literal(json.dumps(visual_obj, ensure_ascii=False))} WHERE kp_id={kp_id} AND code={literal(freeze_snap.get('kind') or '')};")
        after_snap = snapshot_of(item_id)
        assert after_snap.get('rule') == freeze_snap.get('rule'), (after_snap.get('rule'), freeze_snap.get('rule'))
        assert after_snap.get('prompt') == freeze_snap.get('prompt')
        sql(f'UPDATE knowledge_points SET payload={literal(orig_payload)} WHERE id={kp_id};')
        if orig_visual:
            sql(f"UPDATE questions SET visual={literal(orig_visual)} WHERE kp_id={kp_id} AND code={literal(freeze_snap.get('kind') or '')};")
        rule_isolation = {'kpId': kp_id, 'itemId': item_id, 'sourceChanged': True, 'historyUnchanged': True}

    source_replacement = None
    if first_frozen:
        freeze_snap = first_frozen['snap']
        source_url = next((url for url in (freeze_snap.get('mediaSha256') or {}) if '/logic/items/' in url), None)
        if not source_url:
            raise RuntimeError('frozen snapshot missing live source URL')
        source_kp = int(source_url.split('/')[5])
        object_id = source_url.split('/')[-1].split('.')[0]
        source_hex = sql(f"SELECT encode(data,'hex') FROM logic_item_media WHERE kp_id={source_kp} AND object_id={literal(object_id)};")
        other = sql(f"SELECT kp_id||' '||object_id||' '||encode(data,'hex') FROM logic_item_media WHERE NOT (kp_id={source_kp} AND object_id={literal(object_id)}) ORDER BY kp_id LIMIT 1;").split(' ', 2)
        other_kp, other_id, other_hex = int(other[0]), other[1], other[2]
        source_sha = hashlib.sha256(bytes.fromhex(source_hex)).hexdigest()
        other_sha = hashlib.sha256(bytes.fromhex(other_hex)).hexdigest()
        sql(f"UPDATE logic_item_media SET data=decode({literal(other_hex)},'hex'), sha256={literal(other_sha)} WHERE kp_id={source_kp} AND object_id={literal(object_id)};")
        changed = sql(f"SELECT sha256 FROM logic_item_media WHERE kp_id={source_kp} AND object_id={literal(object_id)};")
        assert changed != source_sha
        frozen_url = first_frozen['url']
        frozen_digest = frozen_url.split('/')[-1].split('.')[0]
        after, _ = get_bytes(LOGIC + frozen_url)
        progress_after, _ = get_bytes(PROGRESS + frozen_url.replace('/api/v1/logic/', '/api/logic/'))
        task_after, _ = get_bytes(TASK + frozen_url)
        assert hashlib.sha256(after).hexdigest() == frozen_digest
        assert after == progress_after == task_after
        sql(f"UPDATE logic_item_media SET data=decode({literal(source_hex)},'hex'), sha256={literal(source_sha)} WHERE kp_id={source_kp} AND object_id={literal(object_id)};")
        source_replacement = {'kpId': source_kp, 'objectId': object_id, 'otherKpId': other_kp, 'otherObjectId': other_id, 'sourceChanged': True, 'historyUnchanged': True, 'sha256': frozen_digest}

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
        'ruleIsolation': rule_isolation,
        'sourceReplacement': source_replacement,
        'failSamples': [
            {'id': fail_id, 'qualified': False, 'reason': 'unlinked plan item'},
            {'id': live_id, 'qualified': False, 'reason': 'live glyph url not frozen'},
        ],
        'protectedAttemptsUnchanged': True,
        'emptyOverviewAttempts': empty_detail.get('today', {}).get('attempts', 0),
        'realChildUntouched': REAL_CHILD,
    }
    (OUT / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
    print(json.dumps({'child': child_id, 'task': task['id'], 'plan': plan['plan']['id'], 'mediaChecks': len(media_checks)}, ensure_ascii=False))


if __name__ == '__main__':
    main()
