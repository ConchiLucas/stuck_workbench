import { logicState, logicSummary } from '../src/lib/logicMastery.ts'
import assert from 'node:assert/strict'
import test from 'node:test'

test('logic cards use knowledge-point status, not six-entry rollup', () => {
  const complete = logicState({ status: 'mastered', attempts: 2 })
  assert.equal(complete.complete, true)
  const learning = logicState({ status: 'learning', attempts: 1 })
  assert.equal(learning.complete, false)
  assert.equal(learning.started, true)
})

test('summary keeps mastered, answered and unattempted separate', () => {
  const summary = logicSummary([
    { kind: 'classify', status: 'mastered', attempts: 3 },
    { kind: 'classify', status: 'learning', attempts: 2 },
    { kind: 'classify', status: 'not_started', attempts: 0 },
    { kind: 'pattern', status: 'not_started', attempts: 0 },
  ])
  assert.deepEqual(summary.classify, { mastered: 1, answered: 1, unattempted: 1, total: 3 })
  assert.deepEqual(summary.pattern, { mastered: 0, answered: 0, unattempted: 1, total: 1 })
})
