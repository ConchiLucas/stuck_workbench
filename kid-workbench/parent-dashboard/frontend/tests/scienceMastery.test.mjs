import { scienceState, scienceSummary } from '../src/lib/scienceMastery.ts'
import assert from 'node:assert/strict'
import test from 'node:test'

test('science cards use knowledge-point status, not four-skill rollup', () => {
  const complete = scienceState({
    status: 'mastered',
    attempts: 2,
    skills: [{ code: 'choice', status: 'mastered', accuracy: 1, attempts: 2 }],
  })
  assert.equal(complete.complete, true)
  assert.equal(complete.skills.find((skill) => skill.code === 'choice')?.lit, true)
})

test('summary keeps mastered, answered and unattempted separate', () => {
  const summary = scienceSummary([
    { skills: [{ code: 'choice', status: 'mastered', accuracy: 1, attempts: 3 }] },
    { skills: [{ code: 'choice', status: 'learning', accuracy: 0.5, attempts: 2 }] },
    { skills: [{ code: 'choice', status: 'not_started', accuracy: 0, attempts: 0 }] },
  ])
  assert.deepEqual(summary.choice, { mastered: 1, answered: 1, unattempted: 0, total: 2 })
})
