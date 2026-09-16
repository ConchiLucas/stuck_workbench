#!/usr/bin/env python3
"""Idempotent, explicitly synthetic math acceptance data. Run from repository root.

Uses existing material/task APIs and a dedicated named child; never resets a child.
SQL values are literal-escaped and passed to psql stdin, not interpolated into a shell.
"""
import json
import hashlib
import subprocess
import urllib.request
from pathlib import Path

TAG = 'math-review-demo-v1'
CHILD = '算术验收演示（非真实记录）'
OUT = Path('docs/verification/math-review-demo')


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
    request = urllib.request.Request('http://localhost:19201/api/v1/math/' + path,
        data=None if data is None else json.dumps(data).encode(),
        headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(request, timeout=60) as response:
        return json.load(response)


def real_fingerprint():
    return sql("""SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.id)::text,''))
      FROM attempts t JOIN children c ON c.id=t.child_id WHERE c.name<>%s;
      SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.child_id,t.kp_id,t.skill_code)::text,''))
      FROM mastery_skills t JOIN children c ON c.id=t.child_id WHERE c.name<>%s;
      SELECT md5(COALESCE(jsonb_agg(t ORDER BY t.child_id,t.kp_id)::text,''))
      FROM mastery_states t JOIN children c ON c.id=t.child_id WHERE c.name<>%s;""" %
      (literal(CHILD), literal(CHILD), literal(CHILD)))


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    baseline = real_fingerprint()
    groups = [
        ('算式选择', ['addition-equation', 'subtraction-equation'], 6),
        ('数量图', ['addition-story', 'subtraction-story'], 6),
        ('补全算式', ['addition-missing', 'subtraction-missing'], 6),
        ('判断与找错', ['addition-judge', 'subtraction-error'], 6),
        ('认识图形', ['shape-find', 'shape-name', 'shape-feature', 'shape-sort'], 12),
    ]
    existing = api('question-tasks')['items']
    tasks = []
    for label, ids, count in groups:
        title = f'【验收演示】算术·{label}·{TAG}'
        matches = [task for task in existing if task['title'] == title]
        if len(matches) > 1:
            raise RuntimeError('Duplicate demo task title; inspect manually before rerunning')
        task = api('question-tasks/' + str(matches[0]['id'])) if matches else api(
            'question-tasks', {'title': title, 'detailIds': ids, 'rangeMax': 20, 'count': count})
        # Some handlers return only an id on creation.
        if 'items' not in task:
            task = api('question-tasks/' + str(task['id']))
        assert len(task['items']) == count
        assert {item['detail']['id'] for item in task['items']} == set(ids)
        tasks.append(task)
        (OUT / f'task-{task["id"]}.json').write_text(json.dumps(task, ensure_ascii=False, indent=2))

    questions = json.loads(sql("""SELECT json_agg(x) FROM (
      SELECT q.*,m.code AS module FROM questions q
      JOIN knowledge_points kp ON kp.id=q.kp_id JOIN modules m ON m.id=kp.module_id
      JOIN subjects s ON s.id=m.subject_id WHERE s.code='math' ORDER BY q.id) x;"""))
    scopes = [('add10', 'calc'), ('add10', 'story'), ('sub10', 'calc'),
              ('sub10', 'story'), ('shape', 'find'), ('shape', 'name')]
    events = []
    states = []
    for module, code in scopes:
        selected = [q for q in questions if q['module'] == module and q['code'] == code][:4]
        assert len(selected) == 4
        for index, q in enumerate(selected):
            correct = index >= 2
            repetitions = 3 if index == 3 else 1
            answer = json.loads(q['answer'])['index']
            options = json.loads(q['options'])
            assert len(options) == 4 and 0 <= answer < 4
            raw = json.loads(q['visual'])
            if code == 'calc':
                visual = {'kind': 'equation', 'a': raw['a'], 'b': raw['b'],
                          'operator': '-' if module == 'sub10' else '+'}
            elif code == 'story':
                visual = {'kind': raw['kind'], 'leftCount': raw['a'], 'rightCount': raw['b'],
                          'object': {'🍎': 'apple', '🍓': 'strawberry'}[raw['emoji']]}
            elif code == 'name':
                visual = {'kind': 'shape', 'shape': raw['text']}
            else:
                visual = {'kind': 'none'}
            snapshot = {'questionId': q['id'], 'code': code, 'stem': q['stem'],
                        'options': options, 'answerIndex': answer, 'visual': visual,
                        'audioObjectKey': q['media_url']}
            for repeat in range(repetitions):
                events.append((q, correct, answer if correct else (answer + 1) % 4, snapshot, repeat))
            states.append((q['kp_id'], code, 'mastered' if index == 3 else 'learning' if correct else 'shaky',
                           repetitions, repetitions if correct else 0))

    # All learning rows are in one transaction and tied to an explicitly synthetic child.
    # A repeated run skips the already committed cohort, preserving later manual inspection.
    statements = ["BEGIN; SELECT pg_advisory_xact_lock(9132026);",
        f"INSERT INTO children(name,grade) SELECT {literal(CHILD)},'验收演示' WHERE NOT EXISTS (SELECT 1 FROM children WHERE name={literal(CHILD)});",
        f"CREATE TEMP TABLE demo_child AS SELECT id FROM children WHERE name={literal(CHILD)};",
        "DO $$ BEGIN IF (SELECT count(*) FROM demo_child)<>1 THEN RAISE EXCEPTION 'Ambiguous demo child'; END IF; END $$;",
        f"CREATE TEMP TABLE seed_needed AS SELECT id FROM demo_child WHERE NOT EXISTS (SELECT 1 FROM attempts WHERE child_id=demo_child.id AND client_id LIKE '{TAG}:%');"]
    for number, (q, correct, pick, snapshot, repeat) in enumerate(events, 1):
        # Repeat successes on separate days; one question never mixes correct/wrong outcomes,
        # because the legacy evidence adapter uses the latest plan item for that question.
        at = f"now()-interval '{3-repeat} days'-interval '{number} minutes'"
        key = f'{TAG}:{number}'
        statements.append(f"""WITH p AS (
          INSERT INTO study_plans(child_id,plan_date,seq_no,status,target_count,done_count,correct_count,
            duration_sec,created_at,started_at,completed_at,subject_code,plan_kind,module_code,task_claim_key)
          SELECT id,({at})::date,{number},'completed',1,1,{int(correct)},8,{at},{at},{at},'math','demo',
            {literal(q['module'])},{literal(key)} FROM seed_needed RETURNING id)
          INSERT INTO plan_items(plan_id,seq,kp_id,question_id,bucket,status,tries,cost_ms,answered_at,picks,question_snapshot,content_snapshot_version)
          SELECT id,1,{q['kp_id']},{q['id']},'demo',{literal('correct' if correct else 'wrong')},1,8000,{at},
            {literal(pick)},{literal(json.dumps(snapshot, ensure_ascii=False))}::jsonb,1 FROM p;
          INSERT INTO attempts(child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at)
          SELECT id,{q['kp_id']},{q['id']},{str(correct).lower()},8000,'quiz',{literal(key)},{at} FROM seed_needed;""")
    for kp, code, status, attempts, correct in states:
        mastered_at = "now()-interval '1 day'" if status == 'mastered' else 'NULL'
        statements.append(f"""INSERT INTO mastery_skills(child_id,kp_id,skill_code,status,attempts,correct,
          streak,best_streak,first_seen_at,mastered_at,due_at,updated_at)
          SELECT id,{kp},{literal(code)},{literal(status)},{attempts},{correct},{correct},{correct},now()-interval '4 days',
          {mastered_at},now()+interval '7 days',now() FROM seed_needed;""")
    statements += ["""INSERT INTO mastery_states(child_id,kp_id,status,attempts,correct,streak,best_streak,first_seen_at,mastered_at,due_at,updated_at)
       SELECT child_id,kp_id,CASE WHEN bool_and(status='mastered') THEN 'mastered'
         WHEN bool_or(status='shaky') THEN 'shaky' ELSE 'learning' END,
         sum(attempts),sum(correct),min(streak),min(best_streak),min(first_seen_at),max(mastered_at),max(due_at),now()
       FROM mastery_skills WHERE child_id IN (SELECT id FROM seed_needed) GROUP BY child_id,kp_id;""",
       "COMMIT; SELECT id FROM children WHERE name=" + literal(CHILD) + ";"]
    child_id = int(sql('\n'.join(statements)).splitlines()[-1])
    assert real_fingerprint() == baseline, 'Non-demo learning data changed; investigate concurrent writes'
    summary = json.loads(sql(f"""SELECT json_build_object('attempts',count(*),'correct',count(*) FILTER(WHERE is_correct),
      'wrong',count(*) FILTER(WHERE NOT is_correct)) FROM attempts WHERE child_id={child_id};"""))
    assert summary == {'attempts': 36, 'correct': 24, 'wrong': 12}, summary
    checked_media = 0
    for task in tasks:
        for item in task['items']:
            example = item['detail']['example']
            urls = [example[k] for k in ['audioUrl', 'objectImageUrl'] if example.get(k)]
            urls += list(example.get('shapeImageUrls', {}).values())
            for url in urls:
                with urllib.request.urlopen('http://localhost:19201' + url, timeout=15) as response:
                    assert hashlib.sha256(response.read()).hexdigest() == url.split('/')[-1].split('.')[0]
                checked_media += 1
    with urllib.request.urlopen(f'http://localhost:19211/api/v1/children/{child_id}/knowledge/wrongs?subject=math') as response:
        wrongs = json.load(response)
    assert len(wrongs['items']) == 12
    for wrong in wrongs['items']:
        assert wrong.get('mathExample') and wrong['selectionFidelity'] == 'stable_option'
        assert not wrong['evidenceReasonCodes']
        if wrong['skillCode'] == 'find':
            url = f'http://localhost:19211/api/v1/children/{child_id}/knowledge/attempts/{wrong["attemptId"]}/media/stem-audio'
            with urllib.request.urlopen(url, timeout=15) as response:
                assert response.headers.get_content_type() == 'audio/mpeg' and response.read()
    (OUT / 'wrongs.json').write_text(json.dumps(wrongs, ensure_ascii=False, indent=2))
    manifest = {'tag': TAG, 'childId': child_id, 'childName': CHILD, 'synthetic': True,
                'taskIds': [task['id'] for task in tasks], 'taskQuestions': 36, 'knowledge': summary,
                'verifiedTaskMediaReferences': checked_media,
                'nonDemoLearningFingerprint': baseline.splitlines()}
    (OUT / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
    print(json.dumps(manifest, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
