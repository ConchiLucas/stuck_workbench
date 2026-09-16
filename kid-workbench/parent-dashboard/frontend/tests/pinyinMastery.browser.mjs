// Build first with npm run build, then run: node tests/pinyinMastery.browser.mjs
// Uses the repository's existing pinyin-app Playwright; every request is intercepted.
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
const { webkit } = createRequire(new URL('../../../pinyin-app/package.json', import.meta.url))('playwright')
import { readFile, mkdir } from 'node:fs/promises'
import assert from 'node:assert/strict'
const dist=fileURLToPath(new URL('../../backend/internal/http/dist', import.meta.url))
const output='/tmp/pinyin-dashboard-qa'
await mkdir(output,{recursive:true})
const sk=(code,status='not_started')=>({code,status,attempts:status==='not_started'?0:8,accuracy:.875})
const letter=(id,title,done=0)=>({id,title,kind:'letter',skills:['listen','inword','shape'].map((s,i)=>sk(s,i<done?'mastered':'not_started'))})
const syllable=(id,initial,final,title,tone,done=false)=>({id,title,kind:'syllable',initial,final,syllable:title,tone,last_at:'2026-09-12T03:04:00Z',skills:[sk('blend',done?'mastered':'learning')]})
const modules=[{code:'shengmu',name:'声母',points:['b','p','m','f','d','t','n','l','g','k','h','j','q','x','zh','ch','sh','r','z','c','s','y','w'].map((s,i)=>letter(i+1,s,i%4))},{code:'yunmu',name:'韵母',points:['a','o','e','i','u','ü','ai','ei','ui','ao','ou','iu','ie','üe','er','an','en','in','un','ün','ang','eng','ing','ong'].map((s,i)=>letter(i+30,s,i%4))},{code:'syllables',name:'音节拼读',points:[syllable(70,'b','ā','bā',1,true),syllable(71,'b','ǎ','bǎ',3),syllable(72,'p','á','pá',2),syllable(73,'m','à','mà',4),syllable(74,'l','ǚ','lǚ',3,true),syllable(75,'l','ǔ','lǔ',3)]}]
const subject={code:'pinyin',name:'拼音',total:53,counts:{not_started:0,learning:0,shaky:0,mastered:0,review_due:0}}
const browser=await webkit.launch({headless:true})
const page=await browser.newPage()
await page.clock.install()
const counts={matrix:0,kp:0};
const errors=[];page.on('pageerror',e=>errors.push(String(e)))
await page.route('**/*',async route=>{
 const url=new URL(route.request().url());const path=url.pathname
 if(path.startsWith('/api/')){
  let data=null
  if(path.endsWith('/subjects'))data=[subject]
  else if(path.endsWith('/mastery/matrix')){counts.matrix++;data={subject,modules,rule_version:2,rule_effective_at:'2026-09-12T00:00:00Z',catalog_available:true}}
  else if(path.includes('/knowledge-points/')){counts.kp++;
   const id=Number(path.split('/').pop()),p=modules.flatMap(m=>m.points).find(p=>p.id===id)
   data={...p,kp_id:id,subject_code:'pinyin',module_code:p.kind==='syllable'?'syllables':'shengmu',module_name:p.kind==='syllable'?'音节拼读':'声母',mastered_at:p.skills.every(s=>s.status==='mastered')?'2026-09-12T03:00:00Z':null,history:[{at:'2026-09-12T03:04:00Z',is_correct:true,skill_code:p.kind==='syllable'?'blend':'shape',source:'practice'}]}
  }
  return route.fulfill({json:{data}})
 }
 const file=path.startsWith('/assets/')?`${dist}${path}`:`${dist}/index.html`
 return route.fulfill({body:await readFile(file),contentType:file.endsWith('.js')?'application/javascript':file.endsWith('.css')?'text/css':'text/html'})
})
for(const width of [1280,768,390]){
 await page.setViewportSize({width,height:900});await page.goto('http://pinyin-ui.test/subjects/pinyin')
 await page.getByRole('heading',{name:'音节拼读地图'}).waitFor()
 assert.equal(await page.locator('.pinyin-type-bars button').count(),4)
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth),`page overflow at ${width}`)
 await page.screenshot({path:`${output}/page-${width}.png`,fullPage:true})
 const b=page.locator('.pinyin-card').filter({has:page.locator('.pinyin-glyph',{hasText:/^b$/})})
 await b.click();await page.getByRole('dialog').waitFor()
 assert.equal(await page.getByRole('button',{name:'标记为已学会'}).count(),0)
 assert.equal(await page.evaluate(()=>document.activeElement?.getAttribute('aria-label')),'关闭拼音详情')
 await page.keyboard.press('Tab');assert.equal(await page.evaluate(()=>document.activeElement?.getAttribute('aria-label')),'关闭拼音详情')
 assert.ok(await page.locator('.pinyin-drawer').evaluate(e=>e.scrollWidth<=e.clientWidth),`drawer overflow at ${width}`)
 await page.screenshot({path:`${output}/drawer-${width}.png`})
 await page.keyboard.press('Escape');await page.getByRole('dialog').waitFor({state:'detached'})
 assert.ok(await b.evaluate(e=>e===document.activeElement),'focus restored')
 const search=page.getByRole('textbox',{name:'查找字母或音节'});await search.fill('lv')
 assert.equal(await page.locator('.pinyin-card').count(),1)
 assert.equal(await page.locator('.pinyin-glyph').textContent(),'lǚ')
 await search.fill('bǎ')
 assert.equal(await page.locator('.pinyin-syllable-grid .pinyin-glyph').first().textContent(),'bǎ')
 await page.getByRole('button',{name:'清空搜索'}).click()
 console.log(`PASS ${width}px: layout, drawer, focus trap/restore, search`)
}
const matrixStart=counts.matrix
await page.clock.fastForward(30_100)
await page.waitForFunction(()=>true)
assert.ok(counts.matrix>matrixStart,'visible matrix polls')
const b=page.locator('.pinyin-card').filter({has:page.locator('.pinyin-glyph',{hasText:/^b$/})})
await b.click();await page.getByRole('dialog').waitFor()
const kpStart=counts.kp
await page.clock.fastForward(30_100)
await page.waitForFunction(()=>true)
assert.ok(counts.kp>kpStart,'open detail polls')
await page.keyboard.press('Escape')
await page.getByRole('dialog').waitFor({state:'detached'})
const closed=counts.kp
await page.clock.fastForward(30_100)
await page.waitForFunction(()=>true)
assert.equal(counts.kp,closed,'closed detail stops polling')
await page.evaluate(()=>{Object.defineProperty(document,'visibilityState',{value:'hidden',configurable:true});window.dispatchEvent(new Event('visibilitychange'))})
const hidden=counts.matrix
await page.clock.fastForward(60_100)
await page.waitForFunction(()=>true)
assert.equal(counts.matrix,hidden,'hidden page stops polling')
await page.evaluate(()=>{Object.defineProperty(document,'visibilityState',{value:'visible',configurable:true});window.dispatchEvent(new Event('visibilitychange'))})
await page.waitForFunction(()=>true)
await page.clock.fastForward(1)
assert.ok(counts.matrix>hidden,'focus refreshes immediately')
await page.getByRole('link',{name:'奖励商店'}).click()
await page.getByText('小红花余额').waitFor()
const left=counts.matrix
await page.clock.fastForward(60_100)
await page.waitForFunction(()=>true)
assert.equal(counts.matrix,left,'leaving pinyin stops polling')
console.log('PASS visible 30s polling, hidden pause, focus refresh, drawer close, page exit')
assert.deepEqual(errors,[])
await browser.close()
