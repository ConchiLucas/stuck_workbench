/** Reproduce legacy App / server handwriting differences without changing either implementation.
 * Run: node scripts/verify-handwriting-alignment.mjs
 * Requires literacy-app/node_modules and Go matching shared-go/go.mod. No network fixtures.
 */
import { readFileSync, writeFileSync, mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, resolve, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import assert from 'node:assert/strict'
const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const require = createRequire(join(root, 'literacy-app/package.json'))
const ts = require('typescript')
const templates = Object.fromEntries(['一', '山', '水', '的'].map(char => [char, JSON.parse(readFileSync(join(root, 'shared-go/handwriting/testdata', `${char}.json`), 'utf8'))]))
// Only replace the remote loader: compile and execute the actual App implementation.
const source = readFileSync(join(root, 'literacy-app/src/lib/handwriting.ts'), 'utf8').replace("import HanziWriter from 'hanzi-writer'", 'const HanziWriter = globalThis.__verificationHanziWriter')
globalThis.__verificationHanziWriter = { loadCharacterData: async char => templates[char] }
const compiled = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } }).outputText
const app = await import(`data:text/javascript;base64,${Buffer.from(compiled).toString('base64')}`)
const cases = []
for (const [character, template] of Object.entries(templates)) {
  const standard = template.medians.map(s => s.map(([x,y]) => ({ x:x/1024, y:1-y/1024 })))
  const rows = count => Array.from({length:count}, (_,i) => [{x:0,y:i/(count-1)}, {x:1,y:i/(count-1)}])
  const fragmented = standard.flatMap(stroke => stroke.slice(1).flatMap((to, i) => {
    const from = stroke[i]
    return Array.from({length:3}, (_,j) => [j/3,(j+1)/3].map(f => ({x:from.x+(to.x-from.x)*f, y:from.y+(to.y-from.y)*f})))
  }))
  for (const [sample, points] of Object.entries({ blank:[], standard, 'vertical-mirror':standard.map(s=>s.map(p=>({...p,y:1-p.y}))), ...Object.fromEntries(standard.map((_,i)=>[`missing-stroke-${i+1}`,standard.filter((_,j)=>i!==j)])), 'translated-scaled':standard.map(s=>s.map(p=>({x:p.x*.5+.2,y:p.y*.5+.1}))), 'missing-last-stroke':standard.slice(0,-1), 'only-first-stroke':standard.slice(0,1), 'dense-scribble':rows(32), 'sparse-scribble':rows(4), ...(fragmented.length <= 64 ? {'fragmented-standard':fragmented} : {}) })) {
    let t=0
    const strokes=points.map(s=>s.map(p=>({...p,t:t++})))
    // Actual canvas path: pixel coordinates with downwards Y, passed to looksLikeCharacter unchanged.
    const appCanvas = await app.looksLikeCharacter(points.map(s=>s.map(p=>({x:p.x*320,y:p.y*320}))), character)
    // Control: feed both algorithms equivalent upward coordinates to isolate policy differences.
    const legacyCanvas = app.inkMatch(points.map(s=>s.map(p=>({x:p.x*320,y:p.y*320}))), template.medians.map(s=>s.map(([x,y])=>({x,y}))))
    const appSameAxis = app.inkMatch(points.map(s=>s.map(p=>({x:p.x*1024,y:(1-p.y)*1024}))), template.medians.map(s=>s.map(([x,y])=>({x,y}))))
    cases.push({character,sample,template,strokes,appCanvas,legacyCanvas,appSameAxis})
  }
}
const dir=mkdtempSync(join(tmpdir(),'handwriting-alignment-'))
try {
  const input=join(dir,'cases.json'), runner=join(dir,'main.go')
  writeFileSync(input,JSON.stringify(cases))
  writeFileSync(runner,`package main
import("encoding/json";"os"; "github.com/conchi/study-learning/handwriting")
func main(){var cases []struct{Character string \x60json:"character"\x60; Sample string \x60json:"sample"\x60; Template handwriting.Template \x60json:"template"\x60; Strokes [][]handwriting.Point \x60json:"strokes"\x60}; b,e:=os.ReadFile(os.Args[1]);if e!=nil{panic(e)};if e=json.Unmarshal(b,&cases);e!=nil{panic(e)};out:=[]handwriting.EvaluationResult{};for _,c:=range cases{r,e:=handwriting.Evaluate(c.Strokes,c.Template,handwriting.PolicyV1);if e!=nil{panic(e)};out=append(out,r)};json.NewEncoder(os.Stdout).Encode(out)}
`)
  const results=JSON.parse(execFileSync('go',['run',runner,input],{cwd:join(root,'shared-go'),encoding:'utf8',env:{...process.env,GOCACHE:process.env.GOCACHE||join(tmpdir(),'handwriting-go-cache')}}))
  const report=cases.map((c,i)=>({character:c.character,sample:c.sample,appCanvas:c.appCanvas,legacyCanvas:c.legacyCanvas,appSameAxis:c.appSameAxis,server:results[i]}))
  for(const r of report){
    assert.equal(r.appCanvas.ok,r.appSameAxis.ok,`${r.character}/${r.sample}: canvas orientation differs from same-axis control`)
    if(['standard','translated-scaled'].includes(r.sample)){assert.equal(r.appSameAxis.ok,true);assert.equal(r.appCanvas.score,1);assert.equal(r.server.outcome,'passed')}
    if(r.sample==='blank'){assert.equal(r.appCanvas.ok,false);assert.equal(r.server.outcome,'not_passed')}
    if(r.sample.includes('scribble'))assert.equal(r.server.outcome,'not_passed')
  }
  assert.ok(report.some(r=>r.appSameAxis.ok !== (r.server.outcome==='passed')), 'Expected current policy boundary to remain visible')
  assert.ok(report.some(r=>r.legacyCanvas.ok !== r.appSameAxis.ok), 'Expected legacy coordinate boundary to remain visible')
  console.log(JSON.stringify({fixtures:'shared-go/handwriting/testdata/{一,山,水,的}.json',cases:report.length,results:report},null,2))
} finally {rmSync(dir,{recursive:true,force:true});delete globalThis.__verificationHanziWriter}
