import test from 'node:test'
import assert from 'node:assert/strict'
import { pinyinState, pinyinSummary, normalizePinyin, searchPoints, pinyinGroups, normalizeModules, barWidths } from '../src/lib/pinyinMastery.ts'
const skill = (code, status, attempts = 4) => ({ code, status, attempts, accuracy: 1 })
const point = (id, title, kind = 'letter', skills = [], extra = {}) => ({ id, title, kind, skills, ...extra })

test('字母须听找认三项全过，review_due算掌握，blend不补字母技能', () => {
 const skills = [skill('listen', 'mastered'), skill('inword', 'review_due'), skill('blend', 'mastered')]
 const state = pinyinState({ kind: 'letter', skills })
 assert.equal(state.complete, false)
 assert.deepEqual(state.skills.map(s => s.short), ['听', '找', '认'])
 assert.equal(state.skills[2].detail, '未作答')
 assert.equal(pinyinState({ kind: 'letter', skills: [...skills, skill('shape', 'mastered')] }).complete, true)
})
test('未作答和作答未掌握均暗，音节只按blend计算', () => {
 const state = pinyinState({ kind: 'letter', skills: [skill('listen', 'learning')] })
 assert.equal(state.skills[0].lit, false)
 assert.equal(state.skills[0].detail, '已作答未掌握')
 assert.equal(state.skills[1].lit, false)
 assert.equal(state.skills[1].detail, '未作答')
 assert.equal(pinyinState({ kind: 'syllable', skills: [skill('blend', 'mastered')] }).complete, true)
 assert.equal(pinyinState({ kind: 'syllable', skills: [skill('shape', 'mastered')] }).complete, false)
})
test('题型分母分开且空分母不出现NaN', () => {
 const summary = pinyinSummary([point(1,'b'), point(2,'bā','syllable',[skill('blend','mastered')]), point(3,'bǎ','syllable',[skill('blend','learning')])])
 assert.deepEqual(summary.listen,{ mastered:0,answered:0,unattempted:1,total:1 })
 assert.deepEqual(summary.blend,{ mastered:1,answered:1,unattempted:0,total:2 })
 assert.deepEqual(barWidths(pinyinSummary([]).blend),{ mastered:0,answered:0 })
})
test('拼音专门去声调，保留ü并支持v/u:以及分解Unicode', () => {
 assert.equal(normalizePinyin('LǙ'), 'lü')
 assert.equal(normalizePinyin('lu:'), 'lü')
 assert.equal(normalizePinyin('lv'), 'lü')
 assert.equal(normalizePinyin('lu\u0308\u030c'), 'lü')
 assert.notEqual(normalizePinyin('lǚ'), normalizePinyin('lǔ'))
 const points = [point(1,'bā','syllable'),point(2,'bǎ','syllable'),point(3,'lǚ','syllable'),point(4,'lǔ','syllable')]
 assert.deepEqual(searchPoints(points,'ba').map(p=>p.id),[1,2])
 assert.deepEqual(searchPoints(points,'bǎ').map(p=>p.id),[2,1])
 assert.deepEqual(searchPoints(points,'lv').map(p=>p.id),[3])
})
test('音节按韵母分组再按声母声调ID排序，搜索不改变分母', () => {
 const modules = [{code:'shengmu',name:'声母',points:[point(1,'b'),point(2,'p')]},{code:'syllables',name:'音节',points:[point(4,'bǎ','syllable',[],{initial:'b',final:'ǎ',tone:3}),point(3,'bā','syllable',[],{initial:'b',final:'ā',tone:1}),point(5,'lǚ','syllable',[],{initial:'l',final:'ǚ',tone:3}),point(6,'lǔ','syllable',[],{initial:'l',final:'ǔ',tone:3})]}]
 const groups = pinyinGroups(modules,'b')
 assert.equal(groups.letters[0].points.length,2)
 assert.equal(groups.letters[0].visiblePoints.length,1)
 assert.deepEqual(groups.syllables.map(g=>g.code),['a','ü','u'])
 assert.deepEqual(groups.syllables[0].points.map(p=>p.id),[3,4])
})
test('null数据及未知kind安全处理，旧字母可推断但未知kind不冒充字母', () => {
 assert.deepEqual(normalizeModules(null),[])
 const modules = normalizeModules([{code:'shengmu',name:'声母',points:[null,{id:1,title:'b',skills:null},{id:2,title:'x',kind:'unknown'}]},{code:'syllables',name:'音节',points:null}])
 assert.equal(modules[0].points.length,1)
 assert.equal(modules[0].points[0].kind,'letter')
 assert.deepEqual(modules[0].points[0].skills,[])
 assert.equal(pinyinState({kind:'unknown'}).complete,false)
})
