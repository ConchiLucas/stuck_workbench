#!/usr/bin/env python3
"""Phrase five-end acceptance data. Uses normal plan/answer APIs.

Does not insert complete history snapshots as the qualified freeze path.
Does not modify real-child ledgers.
"""
import hashlib
import json
import os
import shutil
import subprocess
import tempfile
import uuid
import urllib.error
import urllib.request
from pathlib import Path

TAG = 'phrase-alignment-v1'
CHILD = '短句验收演示（非真实记录）'
EMPTY_CHILD = '短句验收空账本（非真实记录）'
TASK_TITLE = '短句验收演示题包（非真实记录）'
REAL_CHILD = '卢沁一'
OUT = Path('docs/verification/phrase-alignment')
LOCK = 9142026
PHRASE = 'http://localhost:19171'
CONTENT = 'http://localhost:19091'
TASK = 'http://localhost:19201'
PROGRESS = 'http://localhost:19081'
KNOWLEDGE = 'http://localhost:19211'
RUN = uuid.uuid4().hex[:10]
KINDS = ['listen_zh', 'listen_en', 'scene', 'reply']


def sql(query):
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


def put_bytes(url, data, content_type):
    request = urllib.request.Request(url, data=data, method='PUT', headers={'Content-Type': content_type})
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
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


def upsert_questions():
    compose = Path(__file__).resolve().parents[1] / 'docker-compose.yml'
    result = subprocess.run(
        ['docker', 'compose', '-f', str(compose), 'run', '--no-deps', '--rm', '--entrypoint', './seed', 'seed', '-mode=questions'],
        cwd=str(compose.parent), capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError(result.stderr or result.stdout)
    return result.stdout.strip().splitlines()[-8:]


def available_say_voice():
    forced = os.environ.get('PHRASE_SAY_VOICE', '').strip()
    listed = subprocess.check_output(['say', '-v', '?'], text=True, errors='replace')
    names = [line.split()[0] for line in listed.splitlines() if line.strip()]
    for voice in ([forced] if forced else []) + ['Samantha', 'Alex', 'Fred', 'Daniel']:
        if voice and voice in names:
            return voice
    raise RuntimeError('no English macOS say voice (tried Samantha/Alex/Fred/Daniel)')


def synth_mp3(text, dest: Path):
    if shutil.which('say') is None or shutil.which('ffmpeg') is None:
        raise RuntimeError('need macOS say and ffmpeg to synthesize whole-sentence MP3')
    aiff = dest.with_suffix('.aiff')
    voice = available_say_voice()
    last = 0.0
    try:
        for rate in (140, 110, 80, 50):
            subprocess.run(['say', '-v', voice, '-r', str(rate), '-o', str(aiff), text], check=True)
            subprocess.run(['ffmpeg', '-y', '-hide_banner', '-loglevel', 'error', '-i', str(aiff), '-codec:a', 'libmp3lame', '-qscale:a', '4', str(dest)], check=True)
            last = duration_seconds(dest)
            if last >= 0.6:
                return last
    finally:
        aiff.unlink(missing_ok=True)
    raise RuntimeError(f'{text!r} speech too short after slower rates: {last}s')


def duration_seconds(path: Path):
    out = subprocess.check_output(['ffprobe', '-v', 'error', '-show_entries', 'format=duration', '-of', 'csv=p=0', str(path)], text=True).strip()
    return float(out)


def ensure_speech(items):
    evidence = []
    with tempfile.TemporaryDirectory() as tmp:
        tmpdir = Path(tmp)
        for item in items:
            if item.get('hasSpeech') and item.get('speechSha256'):
                data, ctype = get_bytes(CONTENT + f"/api/v1/phrase/items/{item['kpId']}/speech.mp3")
                assert ctype.startswith('audio/'), (item['title'], ctype)
                digest = hashlib.sha256(data).hexdigest()
                assert digest == item['speechSha256'], item['title']
                evidence.append({'kpId': item['kpId'], 'title': item['title'], 'sha256': digest, 'bytes': len(data), 'uploaded': False})
                continue
            dest = tmpdir / f"{item['kpId']}.mp3"
            seconds = synth_mp3(item['title'], dest)
            stored = put_bytes(CONTENT + f"/api/v1/phrase/items/{item['kpId']}/speech", dest.read_bytes(), 'audio/mpeg')
            evidence.append({'kpId': item['kpId'], 'title': item['title'], 'sha256': stored['speechSha256'], 'bytes': dest.stat().st_size, 'durationSec': seconds, 'uploaded': True})
    return evidence


def task_ok(task):
    items = task.get('items') or []
    by_kind = {}
    for q in items:
        by_kind.setdefault(q.get('kind'), []).append(q)
        if q.get('kind') != 'scene':
            url = (q.get('example') or {}).get('speechUrl') or ''
            if not url.startswith('/api/v1/phrase/task-media/'):
                return False
    return all(len(by_kind.get(kind, [])) >= 3 for kind in KINDS)


def ensure_task():
    existing = sql(f"SELECT COALESCE(json_agg(id ORDER BY id),'[]') FROM phrase_question_tasks WHERE title={literal(TASK_TITLE)};")
    ids = json.loads(existing) if existing else []
    for task_id in ids:
        task = get_json(f'{TASK}/api/v1/phrase/question-tasks/{task_id}')
        if task_ok(task):
            return task
    return post_json(f'{TASK}/api/v1/phrase/question-tasks', {'title': TASK_TITLE, 'count': 12, 'types': KINDS}, timeout=180)


def create_plan(child_id, question_code, count=4):
    return unwrap(post_json(f'{PHRASE}/api/v1/children/{child_id}/phrase/plans', {'mode': 'type', 'questionCode': question_code, 'count': count}))


def answer(child_id, plan_id, item_id, option_index, client_id):
    return unwrap(post_json(
        f'{PHRASE}/api/v1/children/{child_id}/phrase/plans/{plan_id}/items/{item_id}/answer',
        {'clientId': client_id, 'optionIndex': option_index, 'costMs': 900}))


def original_answer_index(item_id):
    return json.loads(sql(f"SELECT question_answer FROM plan_items WHERE id={int(item_id)};"))['index']


def display_correct(item):
    order = [int(part) for part in item['optionOrder'].split(',') if part != '']
    return order.index(original_answer_index(item['id']))


def snapshot_of(item_id):
    return json.loads(sql(f"SELECT question_snapshot::text FROM plan_items WHERE id={int(item_id)};"))


def client(suffix):
    return f'{TAG}:{RUN}:{suffix}'


def option_count(item):
    opts = item['question']['options']
    if isinstance(opts, str):
        opts = json.loads(opts)
    return len(opts)


def existing_retry(child_id, kind):
    raw = sql(f"""
      SELECT json_build_object('planId', p.id, 'itemId', i.id)
      FROM study_plans p
      JOIN plan_items i ON i.plan_id=p.id AND i.seq=1
      JOIN questions q ON q.id=i.question_id
      WHERE p.child_id={int(child_id)} AND p.subject_code='phrase' AND q.code={literal(kind)}
        AND (SELECT COUNT(*) FROM attempts a WHERE a.plan_item_id=i.id)=2
      ORDER BY p.id DESC LIMIT 1""")
    return json.loads(raw) if raw else None


def plan_item(plan, item_id):
    for item in plan['items']:
        if item['id'] == item_id:
            return item
    return plan['items'][0]


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    baseline = fingerprint()
    question_log = upsert_questions()
    materials = get_json(f'{CONTENT}/api/v1/phrase/items?view=table')
    items = materials.get('items') or []
    if len(items) < 32:
        raise RuntimeError(f'need 32 phrase materials, have {len(items)}')
    speech = ensure_speech(items)
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

    plans = []
    attempts = []
    media_checks = []
    for kind in KINDS:
        reused = existing_retry(child_id, kind)
        if reused:
            plan = unwrap(get_json(f"{PHRASE}/api/v1/children/{child_id}/phrase/plans/{reused['planId']}"))
            item = plan_item(plan, reused['itemId'])
        else:
            plan = create_plan(child_id, kind, 4)
            item = plan['items'][0]
        plans.append({'kind': kind, 'planId': plan['plan']['id'], 'itemIds': [it['id'] for it in plan['items']], 'reused': bool(reused)})
        snap = snapshot_of(item['id'])
        assert snap.get('schema') == 1, snap
        assert snap.get('kind') == kind, snap
        assert not snap.get('selected')
        example = snap['example']
        if kind != 'scene':
            assert snap.get('mediaSHA256'), snap
            url = example['speechUrl']
            assert url.startswith('/api/v1/phrase/task-media/'), url
            digest = url.split('/')[-1].split('.')[0]
            assert len(digest) == 64, url
            frozen, _ = get_bytes(PHRASE + url)
            task_bytes, _ = get_bytes(TASK + url)
            progress_bytes, _ = get_bytes(PROGRESS + url.replace('/api/v1/phrase/', '/api/phrase/'))
            assert hashlib.sha256(frozen).hexdigest() == digest
            assert frozen == task_bytes == progress_bytes
            media_checks.append({'kind': kind, 'sha256': digest, 'bytes': len(frozen)})
        existing = json.loads(sql(f"SELECT COALESCE(json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id),'[]') FROM attempts WHERE child_id={child_id} AND plan_item_id={item['id']};"))
        if not existing:
            correct = display_correct(item)
            answer(child_id, plan['plan']['id'], item['id'], (correct + 1) % option_count(item), client(f'{kind}-wrong'))
            answer(child_id, plan['plan']['id'], item['id'], correct, client(f'{kind}-right'))
            existing = json.loads(sql(f"SELECT json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id) FROM attempts WHERE child_id={child_id} AND plan_item_id={item['id']};"))
        assert len(existing) == 2 and not existing[0]['correct'] and existing[1]['correct'], existing
        progress = unwrap(get_json(f"{PROGRESS}/api/v1/children/{child_id}/knowledge-points/{item['kpId']}"))
        by_id = {h.get('attempt_id'): h for h in progress.get('history') or []}
        for saved in existing:
            review = by_id[saved['id']]['phrase_review']
            assert review['selected'] == saved['selected'], (kind, review, saved)
            if kind != 'scene':
                evidence = get_json(f"{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{saved['id']}")
                assert evidence['response']['selectedOptionId'] == saved['selected']
                if evidence.get('mediaFidelity') == 'immutable':
                    data, _ = get_bytes(f"{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{saved['id']}/media/stem-audio")
                    assert hashlib.sha256(data).hexdigest() == example['speechUrl'].split('/')[-1].split('.')[0]
                (OUT / f"attempt-{saved['id']}.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2))
        attempts.append({'kind': kind, 'item': item['id'], 'attempts': existing, 'optionOrder': item['optionOrder']})

    orders = {row['optionOrder'] for row in attempts}
    assert len(orders) >= 1

    fail_kp = int(sql("SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='phrase' AND kp.title='Good morning.' ORDER BY kp.id LIMIT 1;"))
    sql(f"""INSERT INTO attempts(child_id,kp_id,is_correct,cost_ms,source,client_id,created_at)
        SELECT {child_id},{fail_kp},false,800,'quiz',{literal(TAG + ':fail-unlinked')},NOW()
        WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE client_id={literal(TAG + ':fail-unlinked')});""")
    fail_id = int(sql(f"SELECT id FROM attempts WHERE client_id={literal(TAG + ':fail-unlinked')};"))
    fail_evidence = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{fail_id}')
    assert 'unlinked_plan_item' in (fail_evidence.get('evidenceReasonCodes') or [])
    (OUT / f'fail-unlinked-{fail_id}.json').write_text(json.dumps({'qualified': False, **fail_evidence}, ensure_ascii=False, indent=2))

    live_client = TAG + ':fail-live-media'
    live_snap = json.dumps({
        'schema': 1, 'kind': 'listen_zh', 'skillCode': 'listen_zh', 'responseKind': 'choice',
        'example': {
            'kind': 'listen_zh', 'speech': 'Good morning.',
            'speechUrl': f'/api/v1/phrase/items/{fail_kp}/speech.mp3',
            'options': [{'id': '1', 'label': '早上好。'}, {'id': '2', 'label': '下午好。'}],
            'answerId': '1',
        },
    }, ensure_ascii=False)
    sql(f"""
      WITH q AS (
        SELECT id FROM questions WHERE kp_id={fail_kp} AND code='listen_zh' ORDER BY id LIMIT 1
      ),
      p AS (
        INSERT INTO study_plans(child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count)
        SELECT {child_id},CURRENT_DATE,
               COALESCE((SELECT MAX(seq_no) FROM study_plans WHERE child_id={child_id} AND plan_date=CURRENT_DATE),0)+1,
               'phrase','done',1,1,0
        WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE client_id={literal(live_client)})
        RETURNING id
      ),
      i AS (
        INSERT INTO plan_items(plan_id,seq,kp_id,question_id,bucket,status,tries,picks,question_snapshot)
        SELECT p.id,1,{fail_kp},q.id,'new','wrong',1,'2',{literal(live_snap)} FROM p, q
        RETURNING id, question_id
      )
      INSERT INTO attempts(child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected)
      SELECT {child_id},{fail_kp},i.question_id,false,700,'quiz',{literal(live_client)},NOW(),i.id,'2' FROM i;
    """)
    live_id = int(sql(f"SELECT id FROM attempts WHERE client_id={literal(live_client)};"))
    live_evidence = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{live_id}')
    assert live_evidence.get('mediaFidelity') != 'immutable'
    assert 'media_not_frozen' in (live_evidence.get('evidenceReasonCodes') or [])
    (OUT / f'fail-live-media-{live_id}.json').write_text(json.dumps({'qualified': False, **live_evidence}, ensure_ascii=False, indent=2))

    freeze_snap = snapshot_of(attempts[0]['item'])
    source_url = next(iter(freeze_snap.get('mediaSHA256') or {freeze_snap['example']['speechUrl']: ''}))
    source_kp = int(source_url.split('/')[5])
    source_hex = sql(f"SELECT encode(data,'hex') FROM phrase_item_speech WHERE kp_id={source_kp};")
    other_kp = next(item['kpId'] for item in items if item['kpId'] != source_kp)
    other_hex = sql(f"SELECT encode(data,'hex') FROM phrase_item_speech WHERE kp_id={int(other_kp)};")
    source_sha = hashlib.sha256(bytes.fromhex(source_hex)).hexdigest()
    other_sha = hashlib.sha256(bytes.fromhex(other_hex)).hexdigest()
    sql(f"UPDATE phrase_item_speech SET data=decode({literal(other_hex)},'hex'), sha256={literal(other_sha)} WHERE kp_id={source_kp};")
    changed = sql(f"SELECT sha256 FROM phrase_item_speech WHERE kp_id={source_kp};")
    assert changed != source_sha
    frozen_url = freeze_snap['example']['speechUrl']
    frozen_digest = frozen_url.split('/')[-1].split('.')[0]
    after, _ = get_bytes(TASK + frozen_url)
    progress_after, _ = get_bytes(PROGRESS + frozen_url.replace('/api/v1/phrase/', '/api/phrase/'))
    assert hashlib.sha256(after).hexdigest() == frozen_digest
    assert after == progress_after
    sql(f"UPDATE phrase_item_speech SET data=decode({literal(source_hex)},'hex'), sha256={literal(source_sha)} WHERE kp_id={source_kp};")

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
        'speech': speech,
        'mediaChecks': media_checks,
        'sourceReplacement': {'kpId': source_kp, 'sourceChanged': True, 'historyUnchanged': True, 'sha256': frozen_digest},
        'failSamples': [
            {'id': fail_id, 'qualified': False, 'reason': 'unlinked plan item'},
            {'id': live_id, 'qualified': False, 'reason': 'live speech url not frozen'},
        ],
        'protectedAttemptsUnchanged': True,
        'questionSeedLog': question_log,
        'emptyOverviewAttempts': empty_detail.get('today', {}).get('attempts', 0),
    }
    (OUT / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
    print(json.dumps({'child': child_id, 'task': task['id'], 'plans': [p['planId'] for p in plans], 'mediaChecks': len(media_checks)}, ensure_ascii=False))


if __name__ == '__main__':
    main()
