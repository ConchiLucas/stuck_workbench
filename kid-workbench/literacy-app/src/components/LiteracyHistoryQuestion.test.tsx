import { fireEvent, render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { LiteracyHistoryQuestion } from '../../../parent-dashboard/frontend/src/components/question/LiteracyHistoryQuestion'
import { LiteracyDetailDrawer } from '../../../parent-dashboard/frontend/src/components/mastery/LiteracyDetailDrawer'

const review = { question: { id: 'attempt-9', questionType: 'glyph_sense', interaction: 'choice' as const,
  stem: { image: '/saved/glyph', text: '山' }, options: [{ id: 'water', text: '水', image: '/saved/water', audio: '/saved/audio' }] }, selected_option_id: 'water' }

it('expands frozen history into the same read-only view with the saved selection', () => {
  render(<LiteracyHistoryQuestion review={review} correct={false} />)
  fireEvent.click(screen.getByText('查看当时题目'))
  expect(screen.getByLabelText('看字选义题面')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '水' })).toHaveAttribute('aria-disabled', 'true')
  expect(screen.getByRole('button', { name: '水' })).toHaveAttribute('aria-pressed', 'true')
  expect(screen.getByText('当时答错')).toBeInTheDocument()
})

it('explains absent historical media without creating a question from today’s materials', () => {
  render(<LiteracyHistoryQuestion review={{ unavailable_reason: '这条早期记录未保存原题素材' }} correct />)
  expect(screen.getByText('这条早期记录未保存原题素材')).toBeInTheDocument()
  expect(screen.queryByLabelText('看字选义题面')).not.toBeInTheDocument()
})

it('keeps keyboard focus on the closed history disclosure instead of its hidden audio controls', () => {
  render(<LiteracyDetailDrawer close={() => {}} retry={() => {}} isError={false} data={{
    kp_id: 1, title: '山', module_name: '第一组', subject_code: 'literacy', skills: [],
    history: [{ at: '2026-09-12T10:00:00Z', is_correct: false, cost_ms: 1, source: 'quiz', skill_code: 'glyph_sense', literacy_review: review }],
  } as any} />)
  expect(screen.getByRole('button', { name: '关闭字详情' })).toHaveFocus()
  fireEvent.keyDown(document, { key: 'Tab', shiftKey: true })
  expect(screen.getByText('查看当时题目')).toHaveFocus()
})
