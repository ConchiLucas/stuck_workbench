#!/usr/bin/env python3
"""Idempotent English audit-fix acceptance data. Run from kid-workbench.

New batch english-audit-fix-v1 uses a dedicated child. It does not modify
child 1, math demo child 2, or the previous English demo child.

Do not UPDATE question_snapshot to simulate a later material change, and do
not write non-existent media hashes. Repeat answers must keep increasing
created_at. Child 4 attempts 149/174 remain pre-repair evidence if present.
"""
import copy
import hashlib
import json
import shutil
import subprocess
import tempfile
import urllib.error
import urllib.request
from pathlib import Path

TAG = 'english-audit-fix-v1'
CHILD = '英语审核修复演示（非真实记录）'
OLD_CHILD = '英语验收演示（非真实记录）'
MATH_CHILD = '算术验收演示（非真实记录）'
REAL_CHILD = '卢沁一'
OUT = Path('docs/verification/english-audit-fix-v1')
LOCK = 9132028
GROUPS = [
    ('听音选词', ['audio-choice'], 3),
    ('看图选词', ['image-text'], 3),
    ('组句子', ['card-builder'], 3),
    ('写单词', ['input-gap'], 3),
    ('读一读', ['reading-qa'], 3),
]
ILLEGAL = (
    'This is a nine', 'This is a white', 'This is a mouth',
    'red nine', 'red white', 'red mouth', 'red nose',
)


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
    with urllib.request.urlopen(url, timeout=90) as response:
        return json.load(response)


def get_bytes(url):
    with urllib.request.urlopen(url, timeout=30) as response:
        return response.read(), response.headers.get_content_type()


def put_bytes(url, data, content_type):
    request = urllib.request.Request(url, data=data, method='PUT', headers={'Content-Type': content_type})
    with urllib.request.urlopen(request, timeout=30) as response:
        return response.read()


def api(path, data=None):
    request = urllib.request.Request(
        'http://localhost:19201/api/v1/english/' + path,
        data=None if data is None else json.dumps(data).encode(),
        headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(request, timeout=90) as response:
        return json.load(response)


def real_fingerprint():
    names = ','.join(literal(name) for name in (CHILD, OLD_CHILD, MATH_CHILD))
    return sql(f"""SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.id)::text,''))
      FROM attempts t JOIN children c ON c.id=t.child_id WHERE c.name NOT IN ({names});
      SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.child_id,t.kp_id,t.skill_code)::text,''))
      FROM mastery_skills t JOIN children c ON c.id=t.child_id WHERE c.name NOT IN ({names});
      SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.child_id,t.kp_id)::text,''))
      FROM mastery_states t JOIN children c ON c.id=t.child_id WHERE c.name NOT IN ({names});""")


def child_attempt_counts():
    return sql("""SELECT json_agg(row_obj ORDER BY id) FROM (
        SELECT c.id, json_build_object('id',c.id,'name',c.name,'attempts',count(a.id)) AS row_obj
        FROM children c LEFT JOIN attempts a ON a.child_id=c.id
        WHERE c.name IN (%s,%s,%s)
        GROUP BY c.id,c.name) s;""" % (literal(REAL_CHILD), literal(MATH_CHILD), literal(OLD_CHILD)))


def response_kind(kind):
    return {'card-builder': 'order', 'input-gap': 'input'}.get(kind, 'choice')


def wrong_value(example):
    kind = example['kind']
    if kind == 'input-gap':
        return 'zzzz'
    if kind == 'card-builder':
        tokens = example['answer'].split(' ')
        wrong = ' '.join(reversed(tokens))
        return wrong if wrong != example['answer'] else tokens[0] + ' ' + tokens[0]
    answer = example['answerId']
    for option in example['options']:
        if option['id'] != answer:
            return option['id']
    raise RuntimeError('choice item missing distractor')


def snapshot_for(item, selected, example=None):
    return {
        'schema': 1,
        'kind': item['kind'],
        'skillCode': item['skillCode'],
        'responseKind': response_kind(item['kind']),
        'selected': selected,
        'example': example or item['example'],
    }


def rotate_options(example):
    example = copy.deepcopy(example)
    opts = example['options']
    example['options'] = opts[1:] + opts[:1]
    return example


def media_urls(example):
    urls = [example[key] for key in ('speechUrl', 'cue') if example.get(key)]
    urls += [option['picture'] for option in example.get('options') or [] if option.get('picture')]
    return urls


def upsert_question(item):
    stem = item['example'].get('prompt') or item['example'].get('passage') or item['example'].get('speech') or item['kind']
    return int(sql(f"""INSERT INTO questions (kp_id, code, type, stem, options, answer)
      VALUES ({item['targetId']},{literal(item['skillCode'])},{literal(item['kind'])},{literal(stem)},'[]','{{}}')
      ON CONFLICT (kp_id, code) DO UPDATE SET stem = questions.stem
      RETURNING id;"""))


def pick_voice():
    voices = subprocess.run(['say', '-v', '?'], capture_output=True, text=True, check=True).stdout
    for voice in ('Samantha', 'Allison', 'Ava', 'Alex'):
        if voice in voices:
            return voice
    return ''


def make_sentence_mp3(text, dest):
    aiff = dest.with_suffix('.aiff')
    voice = pick_voice()
    cmd = ['say', '-o', str(aiff), text] if not voice else ['say', '-v', voice, '-o', str(aiff), text]
    subprocess.run(cmd, check=True)
    ffmpeg = shutil.which('ffmpeg')
    if not ffmpeg:
        raise RuntimeError('ffmpeg is required to encode sentence speech')
    subprocess.run(
        [ffmpeg, '-y', '-i', str(aiff), '-codec:a', 'libmp3lame', '-qscale:a', '4', str(dest)],
        check=True, capture_output=True)
    return voice


def prepare_sentence_speech(sentences):
    logs = []
    ffmpeg = shutil.which('ffmpeg')
    afplay = shutil.which('afplay')
    with tempfile.TemporaryDirectory() as tmp:
        tmpdir = Path(tmp)
        for sentence in sentences:
            code = sentence['code']
            text = sentence['text']
            assert ' '.join(sentence['tokens']) == text, sentence
            local = tmpdir / f'{code}.mp3'
            existing = b''
            try:
                existing, _ = get_bytes(f'http://localhost:19091/api/v1/english/sentences/{code}/speech.mp3')
            except urllib.error.HTTPError:
                existing = b''
            voice = ''
            if not existing:
                voice = make_sentence_mp3(text, local)
                put_bytes(f'http://localhost:19091/api/v1/english/sentences/{code}/speech', local.read_bytes(), 'audio/mpeg')
                existing, _ = get_bytes(f'http://localhost:19091/api/v1/english/sentences/{code}/speech.mp3')
            else:
                local.write_bytes(existing)
            digest = hashlib.sha256(existing).hexdigest()
            duration = ''
            if ffmpeg:
                probe = subprocess.run(
                    [ffmpeg, '-i', str(local)], capture_output=True, text=True)
                for line in (probe.stderr or '').splitlines():
                    if 'Duration:' in line:
                        duration = line.strip()
                        break
            played = False
            play_error = ''
            if afplay:
                played = subprocess.run(['afplay', str(local)], capture_output=True).returncode == 0
                if not played:
                    play_error = 'afplay failed'
            else:
                play_error = 'afplay missing'
            logs.append({
                'code': code, 'text': text, 'voice': voice or 'existing-file',
                'sha256': digest, 'bytes': len(existing), 'duration': duration,
                'played': played, 'playError': play_error, 'source': 'say+ffmpeg' if voice else 'existing-content-admin',
            })
    return logs


def assert_legal_task(task):
    for item in task['items']:
        example = item['example']
        blob = json.dumps(example, ensure_ascii=False)
        for bad in ILLEGAL:
            assert bad not in blob, (task['title'], bad, blob)
        assert item.get('sourceContentHash')
        assert not item.get('sourceRevision')
        if item['kind'] == 'card-builder':
            assert item['sourceTable'] == 'english_sentences'
            assert example['speech'] == example['answer']
            assert ' ' in example['answer']
            assert any('/sentences/' in source for source in item['mediaSHA256'])
            assert '/words/' not in example.get('speechUrl', '')
        if item['kind'] == 'reading-qa':
            assert item['sourceTable'] == 'english_passages'
            assert example.get('passage') and example.get('prompt')


def snapshot_selected(row):
    snap = row['snapshot'] if isinstance(row['snapshot'], dict) else json.loads(row['snapshot'])
    return snap.get('selected')


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    sql('ALTER TABLE attempts ADD COLUMN IF NOT EXISTS plan_item_id BIGINT; CREATE INDEX IF NOT EXISTS idx_attempts_plan_item ON attempts(plan_item_id); ALTER TABLE attempts ADD COLUMN IF NOT EXISTS selected TEXT NOT NULL DEFAULT \'\';')
    baseline = real_fingerprint()
    before_counts = child_attempt_counts()
    materials = get_json('http://localhost:19091/api/v1/english/words?view=groups')
    sentences = materials.get('sentences') or []
    passages = materials.get('passages') or []
    assert len(sentences) >= 3, 'need curated sentence materials'
    assert len(passages) >= 3, 'need curated passage materials'
    listen_logs = prepare_sentence_speech(sentences)
    (OUT / 'sentence-listen.json').write_text(json.dumps(listen_logs, ensure_ascii=False, indent=2))
    apple = next(item for item in sentences if item['code'] == 'this-is-an-apple')
    word_audio, _ = get_bytes(f"http://localhost:19091/api/v1/english/words/{apple['targetKpId']}/speech.mp3")
    sentence_audio, _ = get_bytes(f"http://localhost:19091/api/v1/english/sentences/{apple['code']}/speech.mp3")
    assert hashlib.sha256(word_audio).hexdigest() != hashlib.sha256(sentence_audio).hexdigest()

    existing = api('question-tasks')['items']
    tasks = []
    samples = []
    for label, types, count in GROUPS:
        title = f'【审核修复】英语·{label}·{TAG}'
        matches = [task for task in existing if task['title'] == title]
        if len(matches) > 1:
            raise RuntimeError('Duplicate audit-fix task title; inspect manually before rerunning')
        task = api('question-tasks/' + str(matches[0]['id'])) if matches else api(
            'question-tasks', {'title': title, 'types': types, 'count': count})
        if 'items' not in task:
            task = api('question-tasks/' + str(task['id']))
        assert len(task['items']) == count, task
        assert {item['kind'] for item in task['items']} == set(types)
        assert_legal_task(task)
        tasks.append(task)
        (OUT / f'task-{task["id"]}.json').write_text(json.dumps(task, ensure_ascii=False, indent=2))
        samples.append({'label': label, 'taskId': task['id'], 'item': task['items'][0]})
    (OUT / 'sample-one-of-each.json').write_text(json.dumps(samples, ensure_ascii=False, indent=2))

    events = []
    states = []
    used = set()
    number = 0
    listen_first = None
    for task in tasks:
        items = sorted(task['items'], key=lambda item: item['id'])
        for index, item in enumerate(items):
            used.add(item['targetId'])
            question_id = upsert_question(item)
            correct = index >= 1
            selected = item['example'].get('answer') if item['kind'] in ('card-builder', 'input-gap') else item['example']['answerId']
            if not correct:
                selected = wrong_value(item['example'])
            repetitions = 3 if index == 2 else 1
            status = 'mastered' if index == 2 else 'learning' if correct else 'shaky'
            due = "now()-interval '1 day'" if index == 2 and item['kind'] == 'audio-choice' else "now()+interval '7 days'"
            for repeat in range(repetitions):
                number += 1
                events.append((item, question_id, correct, selected, snapshot_for(item, selected), f'{TAG}:{number}', item['example']))
            states.append((item['targetId'], item['skillCode'], status, repetitions, repetitions if correct else 0, due))
            if item['kind'] == 'audio-choice' and index == 0:
                listen_first = (item, question_id, selected)

    if not listen_first:
        raise RuntimeError('missing first listening item for same-question history')
    item, question_id, _ = listen_first
    rotated = rotate_options(item['example'])
    number += 1
    events.append((item, question_id, True, rotated['answerId'], snapshot_for(item, rotated['answerId'], rotated), f'{TAG}:same-q-correct', rotated))

    extra = sql(f"""SELECT kp.id FROM knowledge_points kp
      JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id
      JOIN english_assets ea ON ea.kp_id=kp.id
      WHERE s.code='english' AND COALESCE(ea.sense_image_url,'')<>'' AND COALESCE(ea.speech_audio_url,'')<>''
        AND kp.id NOT IN ({','.join(str(kp) for kp in sorted(used)) or '0'})
      ORDER BY kp.id LIMIT 1;""")
    assert extra, 'need an unused English word for the fully mastered demo point'
    extra_kp = int(extra)

    statements = [
        f"BEGIN; SELECT pg_advisory_xact_lock({LOCK});",
        f"INSERT INTO children(name,grade) SELECT {literal(CHILD)},'验收演示' WHERE NOT EXISTS (SELECT 1 FROM children WHERE name={literal(CHILD)});",
        f"CREATE TEMP TABLE demo_child AS SELECT id FROM children WHERE name={literal(CHILD)};",
        "DO $$ BEGIN IF (SELECT count(*) FROM demo_child)<>1 THEN RAISE EXCEPTION 'Ambiguous demo child'; END IF; END $$;",
        f"CREATE TEMP TABLE seed_needed AS SELECT id FROM demo_child WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE child_id=demo_child.id AND client_id LIKE '{TAG}:%');",
    ]
    for offset, (item, question_id, correct, selected, snapshot, key, _example) in enumerate(events):
        at = f"now()-interval '{max(1, len(events) - offset)} minutes'"
        picks = '' if item['kind'] in ('card-builder', 'input-gap') else selected
        statements.append(f"""WITH p AS (
          INSERT INTO study_plans(child_id,plan_date,seq_no,status,target_count,done_count,correct_count,
            duration_sec,created_at,started_at,completed_at,subject_code,plan_kind,module_code,task_claim_key)
          SELECT id,({at})::date,{offset + 1},'done',1,1,{int(correct)},8,{at},{at},{at},'english','demo',
            {literal(item['moduleCode'])},{literal(key)} FROM seed_needed RETURNING id),
        i AS (
          INSERT INTO plan_items(plan_id,seq,kp_id,question_id,bucket,status,tries,cost_ms,answered_at,picks,question_snapshot,content_snapshot_version)
          SELECT id,1,{item['targetId']},{question_id},'demo',{literal('correct' if correct else 'wrong')},1,8000,{at},
            {literal(picks)},{literal(json.dumps(snapshot, ensure_ascii=False))}::jsonb,0 FROM p RETURNING id)
          INSERT INTO attempts(child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected)
          SELECT seed_needed.id,{item['targetId']},{question_id},{str(correct).lower()},8000,'quiz',{literal(key)},{at},i.id,{literal(selected)}
          FROM seed_needed CROSS JOIN i;""")
    for kp, code, status, attempts, correct, due in states:
        mastered_at = "now()-interval '1 day'" if status == 'mastered' else 'NULL'
        statements.append(f"""INSERT INTO mastery_skills(child_id,kp_id,skill_code,status,attempts,correct,
          streak,best_streak,first_seen_at,mastered_at,due_at,updated_at)
          SELECT id,{kp},{literal(code)},{literal(status)},{attempts},{correct},{correct},{correct},now()-interval '4 days',
          {mastered_at},{due},now() FROM seed_needed;""")
    for code in ('listen', 'picture', 'build', 'type', 'read'):
        statements.append(f"""INSERT INTO mastery_skills(child_id,kp_id,skill_code,status,attempts,correct,
          streak,best_streak,first_seen_at,mastered_at,due_at,updated_at)
          SELECT id,{extra_kp},{literal(code)},'mastered',3,3,3,3,now()-interval '4 days',
          now()-interval '1 day',now()+interval '7 days',now() FROM seed_needed;""")
    statements += ["""INSERT INTO mastery_states(child_id,kp_id,status,attempts,correct,streak,best_streak,first_seen_at,mastered_at,due_at,updated_at)
       SELECT child_id,kp_id,CASE WHEN bool_and(status IN ('mastered','review_due')) THEN
         CASE WHEN bool_or(due_at < now() AND status='mastered') THEN 'review_due' ELSE 'mastered' END
         WHEN bool_or(status='shaky') THEN 'shaky' ELSE 'learning' END,
         sum(attempts),sum(correct),min(streak),min(best_streak),min(first_seen_at),max(mastered_at),max(due_at),now()
       FROM mastery_skills WHERE child_id IN (SELECT id FROM seed_needed) GROUP BY child_id,kp_id;
       INSERT INTO daily_stats(child_id,stat_date,practice_sec,attempts,correct,newly_mastered,checked_in)
       SELECT child_id,created_at::date,count(*)*8,count(*),count(*) FILTER (WHERE is_correct),
         COUNT(*) FILTER (WHERE is_correct AND created_at::date=(now()-interval '1 day')::date),TRUE
       FROM attempts WHERE child_id IN (SELECT id FROM seed_needed) GROUP BY child_id,created_at::date;
       """,
       "COMMIT; SELECT id FROM children WHERE name=" + literal(CHILD) + ";"]
    child_id = int(sql('\n'.join(statements)).splitlines()[-1])
    assert real_fingerprint() == baseline, 'Non-demo learning data changed; investigate concurrent writes'
    assert child_attempt_counts() == before_counts, 'Protected children changed'

    pair = json.loads(sql(f"""SELECT json_agg(json_build_object('attemptId',a.id,'planItemId',a.plan_item_id,'isCorrect',a.is_correct,'createdAt',a.created_at,'snapshot',pi.question_snapshot) ORDER BY a.created_at, a.id)
      FROM attempts a JOIN plan_items pi ON pi.id=a.plan_item_id
      WHERE a.child_id={child_id} AND a.question_id={listen_first[1]};"""))
    assert len(pair) == 2, pair
    assert pair[0]['createdAt'] <= pair[1]['createdAt'], pair
    first_after = get_json(f"http://localhost:19211/api/v1/children/{child_id}/knowledge/attempts/{pair[0]['attemptId']}")
    later_after = get_json(f"http://localhost:19211/api/v1/children/{child_id}/knowledge/attempts/{pair[1]['attemptId']}")
    assert first_after['response']['selectedOptionId'] == snapshot_selected(pair[0])
    assert later_after['response']['selectedOptionId'] == snapshot_selected(pair[1])
    assert first_after['response']['selectedOptionId'] != later_after['response']['selectedOptionId']
    (OUT / 'same-question-history.json').write_text(json.dumps({
        'questionId': listen_first[1],
        'wrongAttemptId': pair[0]['attemptId'],
        'laterAttemptId': pair[1]['attemptId'],
        'first': first_after,
        'later': later_after,
    }, ensure_ascii=False, indent=2))

    summary = json.loads(sql(f"""SELECT json_build_object('attempts',count(*),'correct',count(*) FILTER(WHERE is_correct),
      'wrong',count(*) FILTER(WHERE NOT is_correct),'linked',count(*) FILTER(WHERE plan_item_id IS NOT NULL))
      FROM attempts WHERE child_id={child_id};"""))
    expected_attempts = len(events)
    assert summary['attempts'] == expected_attempts, summary
    assert summary['linked'] == expected_attempts, summary
    checked_media = 0
    for task in tasks:
        for item in task['items']:
            for url in media_urls(item['example']):
                body, content_type = get_bytes('http://localhost:19201' + url)
                digest = url.rsplit('/', 1)[-1].split('.')[0]
                assert hashlib.sha256(body).hexdigest() == digest
                assert len(body) > 32
                checked_media += 1
    with urllib.request.urlopen(f'http://localhost:19211/api/v1/children/{child_id}/knowledge/wrongs?subject=english') as response:
        wrongs = json.load(response)
    assert len(wrongs['items']) == 5, wrongs
    kinds = set()
    for wrong in wrongs['items']:
        assert wrong.get('englishExample') and wrong['selectionFidelity'] == 'stable_option'
        assert 'unlinked_plan_item' not in (wrong.get('evidenceReasonCodes') or [])
        assert wrong['response']['value']
        kinds.add(wrong['englishExample']['kind'])
    assert kinds == {'audio-choice', 'image-text', 'card-builder', 'input-gap', 'reading-qa'}, kinds
    (OUT / 'wrongs.json').write_text(json.dumps(wrongs, ensure_ascii=False, indent=2))
    old_child = sql(f"SELECT c.id||' '||count(*) FROM children c JOIN attempts a ON a.child_id=c.id WHERE c.name={literal(OLD_CHILD)} GROUP BY c.id;")
    manifest = {
        'tag': TAG, 'childId': child_id, 'childName': CHILD, 'synthetic': True,
        'lock': LOCK, 'taskIds': [task['id'] for task in tasks], 'taskQuestions': 15,
        'knowledge': summary, 'fullyMasteredDemoKpId': extra_kp,
        'verifiedTaskMediaReferences': checked_media,
        'sentenceListen': listen_logs,
        'sameQuestion': {'wrongAttemptId': pair[0]['attemptId'], 'laterAttemptId': pair[1]['attemptId'], 'questionId': listen_first[1]},
        'protectedChildAttempts': json.loads(before_counts),
        'oldEnglishDemo': old_child,
        'oldEnglishDemoHandling': 'retained for traceability; not included in this batch checklist; unlinked historical attempts stay unrestorable',
        'nonDemoLearningFingerprint': baseline.splitlines(),
        'cleanup': 'Only delete this named child after confirming name and client_id prefix; never child 1, math demo, or the previous English demo.',
    }
    (OUT / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
    print(json.dumps(manifest, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
