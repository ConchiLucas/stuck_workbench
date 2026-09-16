#!/usr/bin/env python3
"""Normal-plan smoke test; only writes to its named demonstration child's ledger.
Source-byte replacement is covered by shared-go/englishcontent/media_test.go in isolation.
"""
import importlib.util
import hashlib
import json
from pathlib import Path
spec=importlib.util.spec_from_file_location('prior',Path(__file__).with_name('verify-english-second-audit.py'))
h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)
out=Path('docs/verification/english-media-freeze');out.mkdir(parents=True,exist_ok=True)
child=h.ensure_child('英语媒体冻结演示（非真实记录）')
def fingerprint():
 return h.sql(f"SELECT md5(COALESCE(jsonb_agg(a ORDER BY a.id)::text,'')) FROM attempts a WHERE child_id <> {child}")
baseline=fingerprint()
manifest_path=out/'manifest.json'
if manifest_path.exists():
 manifest=json.loads(manifest_path.read_text());plan=manifest['plan']
else:
 plan=h.create_plan(child,8);manifest={'childId':child,'plan':plan};manifest_path.write_text(json.dumps(manifest,ensure_ascii=False,indent=2))
checks=[];attempts=[]
for item in plan['items']:
 snap=h.snapshot_of(item['id']);example=snap['example']
 for url in [example.get('speechUrl'),example.get('cue')]+[o.get('picture') for o in example.get('options',[])]:
  if not url:continue
  digest=url.split('/')[-1].split('.')[0];assert len(digest)==64,url
  assert digest in snap['mediaSHA256'].values()
  for base,path in [('http://localhost:19201',url),('http://localhost:19081',url.replace('/api/v1/english/','/api/english/'))]:
   data,_=h.get_bytes(base+path);assert hashlib.sha256(data).hexdigest()==digest
   checks.append({'url':base+path,'sha256':digest,'bytes':len(data)})
 kind=example['kind']
 if any(a['kind']==kind for a in attempts):continue
 existing=json.loads(h.sql(f"SELECT COALESCE(json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id),'[]') FROM attempts WHERE child_id={child} AND plan_item_id={item['id']};"))
 if not existing:
  correct=h.display_correct(item)
  h.answer(child,plan['plan']['id'],item['id'],(correct+1)%h.option_count(item),f"freeze-{item['id']}-wrong")
  h.answer(child,plan['plan']['id'],item['id'],correct,f"freeze-{item['id']}-right")
  existing=json.loads(h.sql(f"SELECT json_agg(json_build_object('id',id,'selected',selected,'correct',is_correct) ORDER BY id) FROM attempts WHERE child_id={child} AND plan_item_id={item['id']};"))
 assert len(existing)==2 and not existing[0]['correct'] and existing[1]['correct']
 for a in existing:
  evidence=h.get_json(f"http://localhost:19211/api/v1/children/{child}/knowledge/attempts/{a['id']}")
  assert evidence['mediaFidelity']=='immutable',evidence
  assert evidence['response']['selectedOptionId']==a['selected']
  (out/f"attempt-{a['id']}.json").write_text(json.dumps(evidence,ensure_ascii=False,indent=2))
  media_ids=[('stem-audio',example.get('speechUrl')),('stem-image',example.get('cue'))]
  media_ids += [(f'option-{i}-image',o.get('picture')) for i,o in enumerate(example.get('options',[]))]
  for mid,url in media_ids:
   if not url:continue
   endpoint=f"http://localhost:19211/api/v1/children/{child}/knowledge/attempts/{a['id']}/media/{mid}"
   data,_=h.get_bytes(endpoint);digest=url.split('/')[-1].split('.')[0];assert hashlib.sha256(data).hexdigest()==digest
   checks.append({'url':endpoint,'sha256':digest,'bytes':len(data)})
 attempts.append({'kind':kind,'item':item['id'],'attempts':existing})
assert len(attempts)==2,attempts
old=h.get_json('http://localhost:19211/api/v1/children/5/knowledge/attempts/196')
assert old['mediaFidelity']=='mutable_reference'
assert baseline==fingerprint(),'Other children changed; investigate concurrent activity'
manifest.update({'attempts':attempts,'mediaChecks':checks,'protectedAttemptsUnchanged':True,'legacy196Fidelity':old['mediaFidelity']})
manifest_path.write_text(json.dumps(manifest,ensure_ascii=False,indent=2))
print(json.dumps({'child':child,'plan':plan['plan']['id'],'attempts':attempts,'mediaChecks':len(checks),'protectedAttemptsUnchanged':True},ensure_ascii=False))
