import test from 'node:test'
import assert from 'node:assert/strict'
import { dailyQuizFromPlans, ledgerCalendarDays, overallMasteredPct, questionTypeBars, subjectPercents } from '../src/lib/ledgerFacts.ts'

test('empty calendar days stay zero and are not synthesized', () => {
  assert.deepEqual(ledgerCalendarDays(undefined), [])
  assert.deepEqual(ledgerCalendarDays([]), [])
  assert.deepEqual(ledgerCalendarDays([{ date: '2026-09-12', practice_min: 8, attempts: 2, mastered: 1, checked_in: true }]), [
    { date: '2026-09-12', practice_min: 8, attempts: 2, mastered: 1, checked_in: true },
  ])
})

test('subject percents stay zero without saved ledger activity', () => {
  const ordered = [{ code: 'english', name: '英语' }, { code: 'math', name: '算数' }]
  assert.deepEqual(subjectPercents(ordered, []), [
    { code: 'english', name: '英语', pct: 0, learningPct: 0 },
    { code: 'math', name: '算数', pct: 0, learningPct: 0 },
  ])
  assert.equal(overallMasteredPct(undefined, 10), 0)
  assert.equal(overallMasteredPct(0, 10), 0)
  assert.equal(overallMasteredPct(5, 10), 50)
})

test('ten finished items with two wrong are 80% correct not 100%', () => {
  const row = dailyQuizFromPlans([
    {
      plan_date: '2026-09-14', done_count: 10, target_count: 10, correct_count: 8, status: 'done',
      subjects: [{ code: 'english', name: '英语', count: 10, done_count: 10, correct_count: 8 }],
    },
  ], '2026-09-14')[0]
  assert.equal(row.correctPct, 80)
  assert.equal(row.wrongShare, 20)
  assert.equal(row.unansweredShare, 0)
})

test('unanswered items are not counted as wrong', () => {
  const row = dailyQuizFromPlans([
    {
      plan_date: '2026-09-14', done_count: 4, target_count: 10, correct_count: 4, status: 'doing',
      subjects: [{ code: 'english', name: '英语', count: 10, done_count: 4, correct_count: 4 }],
    },
  ], '2026-09-14')[0]
  assert.equal(row.correctPct, 100)
  assert.equal(row.wrongShare, 0)
  assert.equal(row.correctShare, 40)
  assert.equal(row.unansweredShare, 60)
})

test('mixed-subject plans split by subject item counts', () => {
  const rows = dailyQuizFromPlans([
    {
      plan_date: '2026-09-14', done_count: 10, target_count: 10, correct_count: 8, status: 'done',
      subjects: [
        { code: 'literacy', name: '识字', count: 6, done_count: 6, correct_count: 6 },
        { code: 'english', name: '英语', count: 4, done_count: 4, correct_count: 2 },
      ],
    },
  ], '2026-09-14')
  const english = rows.find((r) => r.code === 'english')
  const literacy = rows.find((r) => r.code === 'literacy')
  assert.equal(english?.correctPct, 50)
  assert.equal(literacy?.correctPct, 100)
})

test('same-item retry uses final item result not attempt count', () => {
  const row = dailyQuizFromPlans([
    {
      plan_date: '2026-09-14', done_count: 1, target_count: 1, correct_count: 1, status: 'done',
      subjects: [{ code: 'english', name: '英语', count: 1, done_count: 1, correct_count: 1 }],
    },
  ], '2026-09-14')[0]
  assert.equal(row.correctPct, 100)
  assert.equal(row.done, 1)
})

test('question type bars stay zero without records', () => {
  assert.deepEqual(questionTypeBars(0, 0, 0), { masteredPct: 0, attemptedPct: 0 })
  assert.deepEqual(questionTypeBars(10, 2, 1), { masteredPct: 20, attemptedPct: 10 })
})

test('daily quiz uses saved plans only', () => {
  assert.deepEqual(dailyQuizFromPlans([], '2026-09-13'), [])
  assert.deepEqual(dailyQuizFromPlans([
    { plan_date: '2026-09-13', done_count: 2, target_count: 4, correct_count: 1, status: 'done', subjects: [{ code: 'english', name: '英语', count: 4, done_count: 2, correct_count: 1 }] },
  ], '2026-09-12'), [])
  assert.equal(dailyQuizFromPlans([
    { plan_date: '2026-09-13', done_count: 2, target_count: 4, correct_count: 1, status: 'done', subjects: [{ code: 'english', name: '英语', count: 4, done_count: 2, correct_count: 1 }] },
  ], '2026-09-13')[0].correctPct, 50)
})
