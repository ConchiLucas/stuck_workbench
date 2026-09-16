import test from 'node:test'
import assert from 'node:assert/strict'
import { mathState, mathSummary, mathGroups } from '../src/lib/mathMastery.ts'
const skill = (code,status='mastered',attempts=3)=>({code,status,attempts,accuracy:1})
const point = (id,title,skills=[])=>({id,title,skills,status:'mastered',attempts:0,accuracy:0,due_at:null})
test('arithmetic needs calc/story, shapes need find/name; demo skills cannot complete either',()=>{
 assert.equal(mathState({module_code:'add10',skills:[skill('calc'),skill('shape-feature')]}).complete,false)
 assert.equal(mathState({module_code:'sub10',skills:[skill('calc'),skill('story','review_due')]}).complete,true)
 assert.deepEqual(mathState({module_code:'shape',skills:[skill('find'),skill('name','learning')]}).skills.map(s=>s.lit),[true,false])
 assert.equal(mathState({module_code:'shape',skills:[skill('find'),skill('name')]}).complete,true)
 assert.equal(mathState({module_code:'unknown',skills:[skill('calc')]}).complete,false)
})
test('summary retains distinct applicable denominators and answered unmastered states',()=>{
 const out=mathSummary([{module_code:'add10',skills:[skill('calc')]},{module_code:'sub10',skills:[skill('story','learning')]},{module_code:'shape',skills:[skill('find')]}])
 assert.deepEqual(out.calc,{mastered:1,answered:0,unattempted:1,total:2})
 assert.deepEqual(out.story,{mastered:0,answered:1,unattempted:1,total:2})
 assert.deepEqual(out.find,{mastered:1,answered:0,unattempted:0,total:1})
 assert.deepEqual(out.name,{mastered:0,answered:0,unattempted:1,total:1})
})
test('nonoverlapping 5/10/20 units use sum for addition, minuend for subtraction; searching preserves denominator',()=>{
 const modules=[{code:'add10',name:'加法',points:[point(1,'1+4'),point(2,'5+5'),point(3,'9+11')]},{code:'sub10',name:'减法',points:[point(4,'5-4'),point(5,'10−9'),point(6,'20-19')]},{code:'shape',name:'图形',points:[point(7,'圆形')]}]
 const groups=mathGroups(modules,'5')
 assert.deepEqual(groups.map(g=>g.code),['add10-5','add10-10','add10-20','sub10-5','sub10-10','sub10-20','shape'])
 assert.deepEqual(groups.map(g=>g.points.length),[1,1,1,1,1,1,1])
 assert.deepEqual(groups.map(g=>g.visiblePoints.length),[0,1,0,1,0,0,0])
 assert.equal(mathGroups(modules,'圆')[6].visiblePoints.length,1)
})
test('malformed input stays safe and unknown range stays visible without invented mastery',()=>{
 assert.deepEqual(mathGroups(null),[])
 const groups=mathGroups([{code:'add10',points:[null,point(1,'特殊算式')]}])
 assert.equal(groups.at(-1).name,'加法 · 其他算式')
 assert.equal(groups.at(-1).points.length,1)
 assert.equal(mathState(groups.at(-1).points[0]).complete,false)
})
