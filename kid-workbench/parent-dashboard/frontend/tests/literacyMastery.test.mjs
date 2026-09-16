import test from 'node:test'
import assert from 'node:assert/strict'
import { literacyState, literacySummary, skillState, accuracyLabel } from '../src/lib/literacyMastery.ts'

const skill = (code, status, attempts = 4) => ({ code, status, attempts, accuracy: 1 })

test('两种认字题掌握，但手写缺失，仍不是完全掌握', () => {
  const state = literacyState({ skills: [skill('glyph_sense', 'mastered'), skill('sense_char', 'mastered')] })
  assert.equal(state.complete, false)
  assert.equal(state.mastered, 2)
  assert.deepEqual(state.skills.map(s => s.code), ['glyph_sense', 'sense_char', 'write_char'])
  assert.equal(state.skills[2].detail, '未作答')
  assert.equal(state.skills[2].lit, false)
})

test('未作答与答过未掌握都不亮，但保留不同的详细提示', () => {
  const empty = skillState(skill('write_char', 'not_started', 0))
  const answered = skillState(skill('write_char', 'learning', 3))
  assert.equal(empty.lit, false)
  assert.equal(answered.lit, false)
  assert.equal(empty.detail, '未作答')
  assert.equal(answered.detail, '已作答未掌握')
})

test('三项都过才完整掌握；待复习仍算已掌握，题型中有不稳则不完整', () => {
  const skills = [skill('glyph_sense', 'mastered'), skill('sense_char', 'review_due'), skill('write_char', 'mastered')]
  assert.equal(literacyState({ skills }).complete, true)
  skills[2] = skill('write_char', 'shaky')
  assert.equal(literacyState({ skills }).complete, false)
})

test('汇总使用各项技能，不把重复作答次数当成已掌握字数', () => {
  const summary = literacySummary([
    { skills: [skill('glyph_sense', 'mastered', 99), skill('sense_char', 'mastered'), skill('write_char', 'mastered')] },
    { skills: [skill('glyph_sense', 'mastered'), skill('write_char', 'learning')] },
    { skills: [] },
  ])
  assert.deepEqual(summary.overall, { mastered: 1, partial: 1, none: 1, total: 3 })
  assert.deepEqual(summary.types.write_char, { mastered: 1, answered: 1, unattempted: 1, total: 3 })
  assert.deepEqual(summary.types.glyph_sense, { mastered: 2, answered: 0, unattempted: 1, total: 3 })
})

test('空数据不生成示例进度，零作答正确率不显示为0%', () => {
  assert.equal(literacySummary([]).overall.total, 0)
  assert.equal(accuracyLabel(0, 0), '—')
  assert.equal(accuracyLabel(0, 3), '0%')
  assert.equal(accuracyLabel(0.4, 5), '40%')
})
