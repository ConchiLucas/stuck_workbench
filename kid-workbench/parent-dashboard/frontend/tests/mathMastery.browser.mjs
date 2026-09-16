// Isolated WebKit fixtures: no server or child ledger is contacted.
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { readFile, mkdir } from 'node:fs/promises'
import assert from 'node:assert/strict'
const { webkit } = createRequire(new URL('../../../pinyin-app/package.json', import.meta.url))('playwright')
const dist=fileURLToPath(new URL('../../backend/internal/http/dist',import.meta.url))
const output='/tmp/math-dashboard-qa';await mkdir(output,{recursive:true})
const sk=(code,status='not_started')=>({code,status,attempts:status==='not_started'?0:3,accuracy:1})
let id=0
const point=(title,module_code)=>{id++;return {id,title,module_code,last_at:'2026-09-12T03:04:00Z',skills:(module_code==='shape'?['find','name']:['calc','story']).map((c,i)=>sk(c,id%3>i?'mastered':'not_started'))}}
const adds=[],subs=[]
for(let a=1;a<=19;a++)for(let b=1;a+b<=20;b++)adds.push(point(`${a}+${b}`,'add10'))
for(let a=1;a<=20;a++)for(let b=1;b<=a;b++)subs.push(point(`${a}-${b}`,'sub10'))
const modules=[{code:'add10',name:'20以内加法',points:adds},{code:'sub10',name:'20以内减法',points:subs},{code:'shape',name:'认识图形',points:['圆形','正方形','长方形','三角形','椭圆形','梯形','菱形','五角星'].map(t=>point(t,'shape'))}]
const subject={code:'math',name:'算术',total:408,counts:{not_started:0,learning:0,shaky:0,mastered:0,review_due:0}}
const browser=await webkit.launch({headless:true})
const page=await browser.newPage();await page.clock.install()
const errors=[],counts={matrix:0,kp:0};let fail=false,empty=false
page.on('pageerror',e=>errors.push(String(e)))
await page.route('**/*',async route=>{
 assert.equal(route.request().method(),'GET','progress must remain read-only')
 const path=new URL(route.request().url()).pathname
 if(path.startsWith('/api/')){
  let data=null
  if(path.endsWith('/subjects'))data=[subject]
  else if(path.endsWith('/mastery/matrix')){counts.matrix++;if(fail)return route.fulfill({status:503,json:{error:'offline'}});data={subject,modules:empty?[]:modules,catalog_available:true}}
  else if(path.includes('/knowledge-points/')){counts.kp++;
   const kp=Number(path.split('/').pop()),p=modules.flatMap(m=>m.points).find(p=>p.id===kp)
   data={...p,kp_id:kp,subject_code:'math',module_name:p.module_code==='shape'?'认识图形':'加法',attempts:6,correct:6,accuracy:1,mastered_at:null,history:[{at:'2026-09-12T03:04:00Z',source:'plan',is_correct:true,skill_code:p.module_code==='shape'?'find':'calc'}]}
  }
  return route.fulfill({json:{data}})
 }
 const file=path.startsWith('/assets/')?`${dist}${path}`:`${dist}/index.html`
 return route.fulfill({body:await readFile(file),contentType:file.endsWith('.js')?'application/javascript':file.endsWith('.css')?'text/css':'text/html'})
})
for(const width of [1280,768,390]){
 await page.setViewportSize({width,height:900});await page.goto('http://math-ui.test/subjects/math')
 await page.getByRole('heading',{name:'算术掌握地图'}).waitFor()
 assert.equal(await page.locator('.math-type-bars button').count(),4)
 assert.equal(await page.locator('.math-group-bars .math-bar').count(),14)
 assert.equal(await page.locator('.math-card').count(),408)
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`overflow at ${width}`)
 await page.screenshot({path:`${output}/page-${width}.png`})
 const p=page.locator('.math-card').first()
 assert.match(await p.getAttribute('title'),/算式计算：已掌握.*情境应用：未作答/)
 await p.click();await page.getByRole('dialog').waitFor()
 assert.equal(await page.getByRole('button',{name:'标记为已学会'}).count(),0)
 assert.equal(await page.evaluate(()=>document.activeElement?.getAttribute('aria-label')),'关闭算术详情')
 await page.keyboard.press('Tab');assert.equal(await page.evaluate(()=>document.activeElement?.getAttribute('aria-label')),'关闭算术详情')
 assert.ok(await page.locator('.math-drawer').evaluate(e=>e.scrollWidth<=e.clientWidth),`drawer overflow ${width}`)
 await page.screenshot({path:`${output}/drawer-${width}.png`})
 await page.keyboard.press('Escape');await page.getByRole('dialog').waitFor({state:'detached'})
 assert.ok(await p.evaluate(e=>e===document.activeElement))
 const bar=await page.locator('.math-type-bars .math-bar').first().getAttribute('aria-label')
 await page.getByRole('textbox',{name:'查找算式或图形'}).fill('圆形')
 assert.equal(await page.locator('.math-card').count(),2)
 assert.equal(await page.locator('.math-type-bars .math-bar').first().getAttribute('aria-label'),bar)
 assert.equal(await page.locator('.math-group-bars .math-bar').count(),14)
 await page.getByRole('button',{name:'清空搜索'}).click()
 await page.getByRole('button',{name:'查看听音找图形掌握地图'}).click()
 assert.equal(await page.evaluate(()=>document.activeElement?.id),'math-shape')
 console.log(`PASS ${width}px: 408 catalog fixture points, all bars, search denominators, read-only drawer, focus, layout`)
}
const initial=counts.matrix;await page.clock.fastForward(30_100);await page.waitForTimeout(100);assert.ok(counts.matrix>initial)
fail=true;await page.clock.fastForward(120_000);await page.getByRole('alert').waitFor();assert.equal(await page.locator('.math-card').count(),408)
fail=false;await page.getByRole('button',{name:'重试',exact:true}).click();await page.getByRole('alert').waitFor({state:'detached'})
empty=true;await page.reload();await page.getByText('暂无算术内容。',{exact:true}).waitFor();assert.equal(await page.locator('.math-card').count(),0)
assert.deepEqual(errors,[])
await browser.close();console.log(`PASS polling, cached error/retry, empty state; screenshots ${output}`)
