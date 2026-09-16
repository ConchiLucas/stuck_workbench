#!/usr/bin/env python3
"""Idempotent, explicitly synthetic English acceptance data. Run from kid-workbench.

Uses existing material/task APIs and a dedicated named child; never resets a child.
SQL values are literal-escaped and passed to psql stdin, not interpolated into a shell.
"""
import hashlib
import json
import subprocess
import urllib.request
from pathlib import Path

TAG = 'english-review-demo-v1'
CHILD = '英语验收演示（非真实记录）'
OUT = Path('docs/verification/english-review-demo')
GROUPS = [
    ('听音选词', ['audio-choice'], 3),
    ('看图选词', ['image-text'], 3),
    ('组句子', ['card-builder'], 3),
    ('写单词', ['input-gap'], 3),
    ('读一读', ['reading-qa'], 3),
]


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


def api(path, data=None):
    request = urllib.request.Request(
        'http://localhost:19201/api/v1/english/' + path,
        data=None if data is None else json.dumps(data).encode(),
        headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(request, timeout=90) as response:
        return json.load(response)


def real_fingerprint():
    return sql("""SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.id)::text,''))
      FROM attempts t JOIN children c ON c.id=t.child_id WHERE c.name<>%s;
      SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.child_id,t.kp_id,t.skill_code)::text,''))
      FROM mastery_skills t JOIN children c ON c.id=t.child_id WHERE c.name<>%s;
      SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.child_id,t.kp_id)::text,''))
      FROM mastery_states t JOIN children c ON c.id=t.child_id WHERE c.name<>%s;""" %
      (literal(CHILD), literal(CHILD), literal(CHILD)))


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


def snapshot_for(item, selected):
    return {
        'schema': 1,
        'kind': item['kind'],
        'skillCode': item['skillCode'],
        'responseKind': response_kind(item['kind']),
        'selected': selected,
        'example': item['example'],
    }


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


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    baseline = real_fingerprint()
    existing = api('question-tasks')['items']
    tasks = []
    for label, types, count in GROUPS:
        title = f'【验收演示】英语·{label}·{TAG}'
        matches = [task for task in existing if task['title'] == title]
        if len(matches) > 1:
            raise RuntimeError('Duplicate demo task title; inspect manually before rerunning')
        task = api('question-tasks/' + str(matches[0]['id'])) if matches else api(
            'question-tasks', {'title': title, 'types': types, 'count': count})
        if 'items' not in task:
            task = api('question-tasks/' + str(task['id']))
        assert len(task['items']) == count, task
        assert {item['kind'] for item in task['items']} == set(types)
        for item in task['items']:
            assert item['example']['kind'] == item['kind']
            if item['kind'] == 'card-builder':
                assert item['example'].get('bank') and item['example'].get('answer')
            elif item['kind'] == 'input-gap':
                assert item['example'].get('answer')
            else:
                assert item['example'].get('options') and item['example'].get('answerId')
        tasks.append(task)
        (OUT / f'task-{task["id"]}.json').write_text(json.dumps(task, ensure_ascii=False, indent=2))

    events = []
    states = []
    used = set()
    number = 0
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
                events.append((item, question_id, correct, selected, snapshot_for(item, selected), number, repeat))
            states.append((item['targetId'], item['skillCode'], status, repetitions, repetitions if correct else 0, due))

    extra = sql(f"""SELECT kp.id FROM knowledge_points kp
      JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id
      JOIN english_assets ea ON ea.kp_id=kp.id
      WHERE s.code='english' AND COALESCE(ea.sense_image_url,'')<>'' AND COALESCE(ea.speech_audio_url,'')<>''
        AND kp.id NOT IN ({','.join(str(kp) for kp in sorted(used)) or '0'})
      ORDER BY kp.id LIMIT 1;""")
    assert extra, 'need an unused English word for the fully mastered demo point'
    extra_kp = int(extra)

    statements = [
        "BEGIN; SELECT pg_advisory_xact_lock(9132027);",
        f"INSERT INTO children(name,grade) SELECT {literal(CHILD)},'验收演示' WHERE NOT EXISTS (SELECT 1 FROM children WHERE name={literal(CHILD)});",
        f"CREATE TEMP TABLE demo_child AS SELECT id FROM children WHERE name={literal(CHILD)};",
        "DO $$ BEGIN IF (SELECT count(*) FROM demo_child)<>1 THEN RAISE EXCEPTION 'Ambiguous demo child'; END IF; END $$;",
        f"CREATE TEMP TABLE seed_needed AS SELECT id FROM demo_child WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE child_id=demo_child.id AND client_id LIKE '{TAG}:%');",
    ]
    for item, question_id, correct, selected, snapshot, number, repeat in events:
        at = f"now()-interval '{3 - repeat} days'-interval '{number} minutes'"
        key = f'{TAG}:{number}'
        picks = '' if item['kind'] in ('card-builder', 'input-gap') else selected
        statements.append(f"""WITH p AS (
          INSERT INTO study_plans(child_id,plan_date,seq_no,status,target_count,done_count,correct_count,
            duration_sec,created_at,started_at,completed_at,subject_code,plan_kind,module_code,task_claim_key)
          SELECT id,({at})::date,{number},'done',1,1,{int(correct)},8,{at},{at},{at},'english','demo',
            {literal(item['moduleCode'])},{literal(key)} FROM seed_needed RETURNING id)
          INSERT INTO plan_items(plan_id,seq,kp_id,question_id,bucket,status,tries,cost_ms,answered_at,picks,question_snapshot,content_snapshot_version)
          SELECT id,1,{item['targetId']},{question_id},'demo',{literal('correct' if correct else 'wrong')},1,8000,{at},
            {literal(picks)},{literal(json.dumps(snapshot, ensure_ascii=False))}::jsonb,1 FROM p;
          INSERT INTO attempts(child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at)
          SELECT id,{item['targetId']},{question_id},{str(correct).lower()},8000,'quiz',{literal(key)},{at} FROM seed_needed;""")
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
    summary = json.loads(sql(f"""SELECT json_build_object('attempts',count(*),'correct',count(*) FILTER(WHERE is_correct),
      'wrong',count(*) FILTER(WHERE NOT is_correct)) FROM attempts WHERE child_id={child_id};"""))
    assert summary == {'attempts': 25, 'correct': 20, 'wrong': 5}, summary
    checked_media = 0
    for task in tasks:
        for item in task['items']:
            for url in media_urls(item['example']):
                with urllib.request.urlopen('http://localhost:19201' + url, timeout=15) as response:
                    body = response.read()
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
        assert 'invalid_snapshot' not in (wrong.get('evidenceReasonCodes') or [])
        assert wrong['response']['value']
        kinds.add(wrong['englishExample']['kind'])
        if wrong['skillCode'] == 'listen':
            url = f'http://localhost:19211/api/v1/children/{child_id}/knowledge/attempts/{wrong["attemptId"]}/media/stem-audio'
            with urllib.request.urlopen(url, timeout=15) as response:
                assert response.headers.get_content_type() == 'audio/mpeg' and response.read()
    assert kinds == {'audio-choice', 'image-text', 'card-builder', 'input-gap', 'reading-qa'}, kinds
    (OUT / 'wrongs.json').write_text(json.dumps(wrongs, ensure_ascii=False, indent=2))
    manifest = {
        'tag': TAG, 'childId': child_id, 'childName': CHILD, 'synthetic': True,
        'taskIds': [task['id'] for task in tasks], 'taskQuestions': 15, 'knowledge': summary,
        'fullyMasteredDemoKpId': extra_kp, 'verifiedTaskMediaReferences': checked_media,
        'nonDemoLearningFingerprint': baseline.splitlines(),
        'cleanup': 'Only delete this named child after confirming name and client_id prefix; never child 1 or math demo child.',
    }
    (OUT / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
    print(json.dumps(manifest, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
