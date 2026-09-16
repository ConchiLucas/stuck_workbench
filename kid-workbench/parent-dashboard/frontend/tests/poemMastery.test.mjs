import { poemState, poemSummary } from '../src/lib/poemMastery.ts'
import assert from 'node:assert/strict'
import test from 'node:test'

test('poem complete uses four-skill rollup status, not a single title correct', () => {
  const incomplete = poemState({
    status: 'learning',
    attempts: 2,
    skills: [{ code: 'title', status: 'mastered', accuracy: 1, attempts: 2 }],
  })
  assert.equal(incomplete.complete, false)
  const complete = poemState({
    status: 'mastered',
    attempts: 8,
    skills: [
      { code: 'title', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'fill', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'couplet', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'recite', status: 'mastered', accuracy: 1, attempts: 2 },
    ],
  })
  assert.equal(complete.complete, true)
})

test('summary keeps mastered, answered and unattempted separate', () => {
  const summary = poemSummary([
    { skills: [{ code: 'title', status: 'mastered', accuracy: 1, attempts: 3 }] },
    { skills: [{ code: 'title', status: 'learning', accuracy: 0.5, attempts: 2 }] },
    { skills: [{ code: 'title', status: 'not_started', accuracy: 0, attempts: 0 }] },
  ])
  assert.deepEqual(summary.title, { mastered: 1, answered: 1, unattempted: 1, total: 3 })
})
