import { phraseState, phraseSummary } from '../src/lib/phraseMastery.ts'
import assert from 'node:assert/strict'
import test from 'node:test'

test('complete mastery requires the three core skills, not reply', () => {
  const complete = phraseState({
    skills: [
      { code: 'listen_zh', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'listen_en', status: 'review_due', accuracy: 1, attempts: 2 },
      { code: 'scene', status: 'mastered', accuracy: 1, attempts: 2 },
    ],
  })
  assert.equal(complete.complete, true)
  const missingScene = phraseState({
    skills: [
      { code: 'listen_zh', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'listen_en', status: 'mastered', accuracy: 1, attempts: 2 },
      { code: 'reply', status: 'mastered', accuracy: 1, attempts: 2 },
    ],
  })
  assert.equal(missingScene.complete, false)
})

test('summary keeps mastered, answered and unattempted separate', () => {
  const summary = phraseSummary([
    { skills: [{ code: 'listen_zh', status: 'mastered', accuracy: 1, attempts: 3 }] },
    { skills: [{ code: 'listen_zh', status: 'learning', accuracy: 0.5, attempts: 2 }] },
    { skills: [{ code: 'listen_zh', status: 'not_started', accuracy: 0, attempts: 0 }] },
  ])
  assert.deepEqual(summary.listen_zh, { mastered: 1, answered: 1, unattempted: 1, total: 3 })
})
