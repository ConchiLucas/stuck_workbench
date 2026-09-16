import { chengyuState, chengyuSummary } from '../src/lib/chengyuMastery.ts'
import assert from 'node:assert/strict'
import test from 'node:test'

test('complete mastery requires all four skills', () => {
  const complete = chengyuState({
    skills: [
      { code: 'meaning', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'pick', status: 'review_due', accuracy: 1, attempts: 2 },
      { code: 'pinyin', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'example', status: 'mastered', accuracy: 1, attempts: 2 },
    ],
  })
  assert.equal(complete.complete, true)
  const missingExample = chengyuState({
    skills: [
      { code: 'meaning', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'pick', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'pinyin', status: 'mastered', accuracy: 1, attempts: 2 },
    ],
  })
  assert.equal(missingExample.complete, false)
})

test('summary keeps mastered, answered and unattempted separate', () => {
  const summary = chengyuSummary([
    { skills: [{ code: 'meaning', status: 'mastered', accuracy: 1, attempts: 3 }] },
    { skills: [{ code: 'meaning', status: 'learning', accuracy: 0.5, attempts: 2 }] },
    { skills: [{ code: 'meaning', status: 'not_started', accuracy: 0, attempts: 0 }] },
  ])
  assert.deepEqual(summary.meaning, { mastered: 1, answered: 1, unattempted: 1, total: 3 })
})
