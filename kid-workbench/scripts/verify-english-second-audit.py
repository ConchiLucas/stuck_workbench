#!/usr/bin/env python3
"""Verify EN-A04/A06 through live english-server plan/answer APIs.

Does not insert complete history snapshots by hand. Does not modify
child 1, math demo, previous English demos, or the broken child-4 pair.
"""
import hashlib
import json
import uuid
import urllib.error
import urllib.request
from datetime import date
from pathlib import Path

TAG = 'english-second-audit-v1'
CHILD = '英语二次审核演示（非真实记录）'
EMPTY_CHILD = '英语二次审核空账本（非真实记录）'
OLD_FIX_CHILD = '英语审核修复演示（非真实记录）'
OLD_CHILD = '英语验收演示（非真实记录）'
MATH_CHILD = '算术验收演示（非真实记录）'
REAL_CHILD = '卢沁一'
OUT = Path('docs/verification/english-second-audit-v2')
LOCK = 9132029
ENGLISH = 'http://localhost:19131'
PROGRESS = 'http://localhost:19081'
KNOWLEDGE = 'http://localhost:19211'
RUN = uuid.uuid4().hex[:10]


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


def http_status(url):
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            return response.status, len(response.read())
    except urllib.error.HTTPError as exc:
        return exc.code, 0
    except Exception as exc:
        return 0, str(exc)


def post_json(url, payload):
    request = urllib.request.Request(
        url, data=json.dumps(payload).encode(), method='POST',
        headers={'Content-Type': 'application/json'})
    try:
        with urllib.request.urlopen(request, timeout=90) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        raise RuntimeError((url, exc.code, exc.read().decode(errors='replace'))) from exc


def unwrap(body):
    if isinstance(body, dict) and 'data' in body and 'error' in body:
        if body.get('error'):
            raise RuntimeError(body['error'])
        return body['data']
    return body


def real_fingerprint():
    names = ','.join(literal(name) for name in (CHILD, EMPTY_CHILD, OLD_FIX_CHILD, OLD_CHILD, MATH_CHILD))
    return sql(f"""SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.id)::text,''))
      FROM attempts t JOIN children c ON c.id=t.child_id WHERE c.name NOT IN ({names});""")


def child_attempt_counts():
    return sql(f"""SELECT json_object_agg(name, n) FROM (
      SELECT c.name, count(a.id)::int AS n FROM children c
      LEFT JOIN attempts a ON a.child_id=c.id
      WHERE c.name IN ({literal(REAL_CHILD)},{literal(MATH_CHILD)},{literal(OLD_CHILD)},{literal(OLD_FIX_CHILD)})
      GROUP BY c.name) s;""")


def ensure_child(name):
    sql(f"INSERT INTO children(name,grade) SELECT {literal(name)},'验收演示' WHERE NOT EXISTS (SELECT 1 FROM children WHERE name={literal(name)});")
    child_id = int(sql(f"SELECT id FROM children WHERE name={literal(name)};").splitlines()[-1])
    return child_id


def create_plan(child_id, count):
    return unwrap(post_json(f'{ENGLISH}/api/v1/children/{child_id}/english/plans', {'mode': 'today', 'count': count}))


def answer(child_id, plan_id, item_id, option_index, client_id):
    return unwrap(post_json(
        f'{ENGLISH}/api/v1/children/{child_id}/english/plans/{plan_id}/items/{item_id}/answer',
        {'clientId': client_id, 'optionIndex': option_index, 'costMs': 900}))


def original_answer_index(item_id):
    raw = sql(f"SELECT question_answer FROM plan_items WHERE id={int(item_id)};")
    return json.loads(raw)['index']


def option_count(item):
    opts = item['question']['options']
    if isinstance(opts, str):
        opts = json.loads(opts)
    return len(opts)


def display_correct(item):
    order = [int(part) for part in item['optionOrder'].split(',') if part != '']
    original = original_answer_index(item['id'])
    return order.index(original)


def snapshot_of(item_id):
    raw = sql(f"SELECT question_snapshot::text FROM plan_items WHERE id={int(item_id)};")
    return json.loads(raw)


def client(suffix):
    return f'{TAG}:{RUN}:{suffix}'


def media_url_status(url):
    if not url:
        return {'url': url, 'status': 0, 'bytes': 0}
    if url.startswith('/api/v1/english/task-media/'):
        status, n = http_status('http://localhost:19201' + url)
    elif url.startswith('/api/v1/english/words/') or url.startswith('/api/v1/english/sentences/'):
        status, n = http_status(ENGLISH + url)
    elif url.startswith('/api/english/'):
        status, n = http_status(PROGRESS + url)
    else:
        status, n = http_status(url if url.startswith('http') else ENGLISH + url)
    return {'url': url, 'status': status, 'bytes': n}


def dump_broken_child4(out):
    row = sql(f"SELECT id FROM children WHERE name={literal(OLD_FIX_CHILD)};")
    if not row:
        evidence = {'present': False, 'qualified': False}
        (out / 'child4-before-repair.json').write_text(json.dumps(evidence, ensure_ascii=False, indent=2))
        return evidence
    child_id = int(row)
    pair = sql("""SELECT json_agg(json_build_object(
        'attemptId', a.id, 'createdAt', a.created_at, 'isCorrect', a.is_correct,
        'planItemId', a.plan_item_id, 'speechUrl', pi.question_snapshot->'example'->>'speechUrl',
        'selected', COALESCE(NULLIF(a.selected,''), pi.question_snapshot->>'selected'),
        'labelSample', pi.question_snapshot->'example'->'options'->0->>'label') ORDER BY a.id)
      FROM attempts a JOIN plan_items pi ON pi.id=a.plan_item_id
      WHERE a.id IN (149,174);""")
    rows = json.loads(pair) if pair else []
    for item in rows or []:
        item['speechHttp'] = media_url_status(item.get('speechUrl'))
        item['qualified'] = False
        item['reason'] = 'pre-repair evidence: inverted timestamps and/or snapshot mutation with unplayable media'
    evidence = {
        'present': True, 'childId': child_id, 'qualified': False,
        'attempts149and174': rows,
        'note': 'Kept as pre-repair evidence. Not a qualified sample. Do not mutate these snapshots.',
    }
    (out / 'child4-before-repair.json').write_text(json.dumps(evidence, ensure_ascii=False, indent=2))
    return evidence


def eligible_question_count():
    return int(sql("""SELECT count(*) FROM questions q
      JOIN knowledge_points kp ON kp.id=q.kp_id
      JOIN modules m ON m.id=kp.module_id
      JOIN subjects sub ON sub.id=m.subject_id
      JOIN english_assets ea ON ea.kp_id=kp.id
      WHERE sub.code='english' AND q.code IN ('listen','picture')
        AND COALESCE(ea.speech_audio_url,'')<>'';"""))


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    sql("ALTER TABLE attempts ADD COLUMN IF NOT EXISTS selected TEXT NOT NULL DEFAULT '';")
    sql('ALTER TABLE attempts ADD COLUMN IF NOT EXISTS plan_item_id BIGINT;')
    baseline = real_fingerprint()
    before_counts = child_attempt_counts()
    broken = dump_broken_child4(OUT)
    eligible = eligible_question_count()
    if eligible < 10:
        raise RuntimeError(f'need at least 10 live listen/picture questions, have {eligible}')

    child_id = ensure_child(CHILD)
    empty_id = ensure_child(EMPTY_CHILD)
    sql(f"BEGIN; SELECT pg_advisory_xact_lock({LOCK}); COMMIT;")

    retry_plan = create_plan(child_id, 4)
    plan_id = retry_plan['plan']['id']
    items = retry_plan['items']
    assert items, retry_plan
    item = items[0]
    snap = snapshot_of(item['id'])
    assert snap.get('schema') == 1, snap
    assert snap.get('kind') in ('audio-choice', 'image-text'), snap
    assert len(snap['example']['options']) >= 2
    assert snap['example'].get('answerId')
    assert not snap.get('selected')
    media_url = snap['example'].get('speechUrl')
    media_bytes, media_type = get_bytes(ENGLISH + snap['example']['speechUrl'])
    assert len(media_bytes) > 32, media_type
    media_digest = hashlib.sha256(media_bytes).hexdigest()

    correct = display_correct(item)
    wrong = (correct + 1) % option_count(item)
    first = answer(child_id, plan_id, item['id'], wrong, client('retry-wrong'))
    assert first['correct'] is False, first
    second = answer(child_id, plan_id, item['id'], correct, client('retry-right'))
    assert second['correct'] is True, second

    attempts = json.loads(sql(f"""SELECT json_agg(json_build_object('id',id,'selected',selected,'isCorrect',is_correct,'createdAt',created_at) ORDER BY id)
      FROM attempts WHERE child_id={child_id} AND plan_item_id={item['id']};"""))
    assert len(attempts) == 2, attempts
    assert attempts[0]['createdAt'] <= attempts[1]['createdAt']
    assert attempts[0]['selected'] != attempts[1]['selected']
    assert attempts[0]['isCorrect'] is False
    assert attempts[1]['isCorrect'] is True

    kp_id = item['kpId']
    progress = unwrap(get_json(f'{PROGRESS}/api/v1/children/{child_id}/knowledge-points/{kp_id}'))
    hist = [h for h in progress['history'] if h.get('english_review') and h['english_review'].get('example')]
    first_hist = next(h for h in hist if h.get('attempt_id') == attempts[0]['id'])
    later_hist = next(h for h in hist if h.get('attempt_id') == attempts[1]['id'])
    assert first_hist['english_review']['selected'] == attempts[0]['selected']
    assert later_hist['english_review']['selected'] == attempts[1]['selected']
    assert first_hist['english_review']['example']['options'] == later_hist['english_review']['example']['options']

    rewritten = first_hist['english_review']['example']['speechUrl']
    progress_media, progress_type = get_bytes(PROGRESS + rewritten)
    assert len(progress_media) > 32, (rewritten, progress_type)
    knowledge_media = media_url_status(
        f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{attempts[0]["id"]}/media/stem-audio')

    know_first = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{attempts[0]["id"]}')
    know_later = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{attempts[1]["id"]}')
    assert know_first['response']['selectedOptionId'] == attempts[0]['selected']
    assert know_later['response']['selectedOptionId'] == attempts[1]['selected']
    assert know_first['englishExample']['options'] == know_later['englishExample']['options']

    distractor = next(o['id'] for o in snap['example']['options'] if o['id'] != snap['example']['answerId'])
    old_payload = sql(f"SELECT payload FROM knowledge_points WHERE id={int(distractor)};")
    old_asset = sql(f"SELECT COALESCE(speech_audio_url,'') FROM english_assets WHERE kp_id={int(kp_id)};")
    snap_text = sql(f"SELECT question_snapshot::text FROM plan_items WHERE id={int(item['id'])};")
    before_first = json.dumps(know_first['englishExample'], ensure_ascii=False, sort_keys=True)
    before_later = json.dumps(know_later['englishExample'], ensure_ascii=False, sort_keys=True)
    before_selected = (know_first['response']['selectedOptionId'], know_later['response']['selectedOptionId'])
    before_media = (know_first['englishExample'].get('speechUrl'), snap.get('mediaSHA256'))
    sql(f"UPDATE knowledge_points SET payload = jsonb_set(payload::jsonb, '{{meaningZh}}', to_jsonb('验收改写义项'::text))::text WHERE id={int(distractor)};")
    if old_asset:
        sql(f"UPDATE english_assets SET speech_audio_url={literal('english-second-audit-v2-source-edit')} WHERE kp_id={int(kp_id)};")
    try:
        after_first = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{attempts[0]["id"]}')
        after_later = get_json(f'{KNOWLEDGE}/api/v1/children/{child_id}/knowledge/attempts/{attempts[1]["id"]}')
        after_progress = unwrap(get_json(f'{PROGRESS}/api/v1/children/{child_id}/knowledge-points/{kp_id}'))
        after_hist = [h for h in after_progress['history'] if h.get('english_review') and h['english_review'].get('example')]
        after_first_hist = next(h for h in after_hist if h.get('attempt_id') == attempts[0]['id'])
        after_later_hist = next(h for h in after_hist if h.get('attempt_id') == attempts[1]['id'])
        assert json.dumps(after_first['englishExample'], ensure_ascii=False, sort_keys=True) == before_first
        assert json.dumps(after_later['englishExample'], ensure_ascii=False, sort_keys=True) == before_later
        assert (after_first['response']['selectedOptionId'], after_later['response']['selectedOptionId']) == before_selected
        assert '验收改写义项' not in json.dumps(after_first, ensure_ascii=False)
        assert after_first_hist['english_review']['example']['options'] == first_hist['english_review']['example']['options']
        assert after_later_hist['english_review']['selected'] == attempts[1]['selected']
        assert sql(f"SELECT question_snapshot::text FROM plan_items WHERE id={int(item['id'])};") == snap_text
        after_snap = json.loads(snap_text)
        assert after_first['englishExample'].get('speechUrl') == before_media[0]
        assert after_snap.get('mediaSHA256') == before_media[1]
    finally:
        sql(f"UPDATE knowledge_points SET payload={literal(old_payload)} WHERE id={int(distractor)};")
        if old_asset:
            sql(f"UPDATE english_assets SET speech_audio_url={literal(old_asset)} WHERE kp_id={int(kp_id)};")

    stats_plan = create_plan(child_id, 10)
    stats_id = stats_plan['plan']['id']
    assert stats_id != plan_id
    assert len(stats_plan['items']) == 10, (eligible, len(stats_plan['items']))
    done_correct = 0
    done_wrong = 0
    for seq, it in enumerate(stats_plan['items']):
        correct_idx = display_correct(it)
        if done_wrong < 2:
            other = (correct_idx + 1) % option_count(it)
            r1 = answer(child_id, stats_id, it['id'], other, client(f'stats-wrong-{seq}-a'))
            assert r1['correct'] is False
            r2 = answer(child_id, stats_id, it['id'], other, client(f'stats-wrong-{seq}-b'))
            assert r2['correct'] is False
            done_wrong += 1
        else:
            r = answer(child_id, stats_id, it['id'], correct_idx, client(f'stats-right-{seq}'))
            assert r['correct'] is True
            done_correct += 1
    assert done_correct == 8 and done_wrong == 2, (done_correct, done_wrong)

    partial = create_plan(child_id, 6)
    partial_id = partial['plan']['id']
    assert len(partial['items']) == 6, len(partial['items'])
    for seq, it in enumerate(partial['items'][:3]):
        answer(child_id, partial_id, it['id'], display_correct(it), client(f'partial-{seq}'))

    today = date.today().isoformat()
    plan_dates = {
        (retry_plan['plan'].get('planDate') or today)[:10],
        (stats_plan['plan'].get('planDate') or today)[:10],
        (partial['plan'].get('planDate') or today)[:10],
        today,
    }
    from_day = min(plan_dates)
    to_day = max(plan_dates)
    plans = unwrap(get_json(f'{PROGRESS}/api/v1/children/{child_id}/plans?from={from_day}&to={to_day}'))
    stats_row = next(p for p in plans if p['id'] == stats_id)
    partial_row = next(p for p in plans if p['id'] == partial_id)
    english_stats = next(s for s in stats_row['subjects'] if s['code'] == 'english')
    assert english_stats['done_count'] == 10, english_stats
    assert english_stats['correct_count'] == 8, english_stats
    accuracy = round(english_stats['correct_count'] / english_stats['done_count'] * 100)
    assert accuracy == 80, english_stats
    english_partial = next(s for s in partial_row['subjects'] if s['code'] == 'english')
    assert english_partial['done_count'] == 3, english_partial
    assert english_partial['correct_count'] == 3, english_partial
    assert english_partial['count'] == 6, english_partial
    unanswered = english_partial['count'] - english_partial['done_count']
    assert unanswered == 3
    assert (english_partial['done_count'] - english_partial['correct_count']) == 0

    subjects = unwrap(get_json(f'{PROGRESS}/api/v1/children/{child_id}/subjects'))
    english_types = next(s['question_types'] for s in subjects if s['code'] == 'english')
    type_names = [t['name'] for t in english_types]
    assert type_names == ['听音选词', '看图选词', '组句子', '写单词', '读一读'], type_names

    empty_subjects = unwrap(get_json(f'{PROGRESS}/api/v1/children/{empty_id}/subjects'))
    empty_english = next(s for s in empty_subjects if s['code'] == 'english')
    assert all(t['mastered'] == 0 and t['attempted'] == 0 for t in empty_english['question_types']), empty_english
    empty_overview = unwrap(get_json(f'{PROGRESS}/api/v1/children/{empty_id}/overview'))
    assert empty_overview['counts']['mastered'] == 0
    assert empty_overview['counts']['learning'] == 0
    empty_plans = unwrap(get_json(f'{PROGRESS}/api/v1/children/{empty_id}/plans?from={from_day}&to={to_day}'))
    assert empty_plans == [] or all(p['done_count'] == 0 for p in empty_plans)
    empty_cal = unwrap(get_json(f'{PROGRESS}/api/v1/children/{empty_id}/stats/calendar?months=4'))
    assert all((d.get('mastered') or 0) == 0 and (d.get('attempts') or 0) == 0 for d in (empty_cal or []))

    assert real_fingerprint() == baseline
    assert child_attempt_counts() == before_counts

    manifest = {
        'tag': TAG,
        'run': RUN,
        'childId': child_id,
        'childName': CHILD,
        'emptyChildId': empty_id,
        'emptyChildName': EMPTY_CHILD,
        'retryPlanId': plan_id,
        'retryItemId': item['id'],
        'questionId': item['question']['id'],
        'kpId': kp_id,
        'wrongAttemptId': attempts[0]['id'],
        'rightAttemptId': attempts[1]['id'],
        'wrongSelected': attempts[0]['selected'],
        'rightSelected': attempts[1]['selected'],
        'statsPlanId': stats_id,
        'partialPlanId': partial_id,
        'planDateRange': {'from': from_day, 'to': to_day},
        'statsAccuracy': accuracy,
        'partialUnanswered': unanswered,
        'eligibleLiveQuestions': eligible,
        'media': {
            'url': media_url, 'sha256': media_digest, 'bytes': len(media_bytes),
            'contentType': media_type, 'progressProxy': rewritten,
            'progressBytes': len(progress_media), 'knowledgeMedia': knowledge_media,
        },
        'sourceEditRestored': True,
        'brokenChild4Evidence': broken,
        'protectedChildAttempts': json.loads(before_counts),
        'note': 'Qualified samples come from POST /english/plans and /answer. Child 4 attempts 149/174 remain pre-repair evidence and are not in this checklist.',
    }
    (OUT / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
    (OUT / 'retry-history.json').write_text(json.dumps({
        'snapshot': snap, 'attempts': attempts, 'progress': first_hist, 'knowledge': {'first': know_first, 'later': know_later},
    }, ensure_ascii=False, indent=2))
    print(json.dumps(manifest, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
