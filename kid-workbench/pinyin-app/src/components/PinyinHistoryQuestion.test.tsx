import { fireEvent, render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { PinyinHistoryQuestion } from '../../../parent-dashboard/frontend/src/components/question/PinyinHistoryQuestion'
import { PinyinDetailDrawer } from '../../../parent-dashboard/frontend/src/components/mastery/pinyin/PinyinDetailDrawer'

const review = {
  question: {
    id: 'listen-1',
    type: 'listen' as const,
    stem: '听一听，选出你听到的拼音',
    speechUrl: '/api/pinyin/items/100/speech/solo.mp3',
    visual: { kind: 'sound' },
    options: [
      { id: 'option-0', label: 'b' },
      { id: 'option-1', label: 'p' },
      { id: 'option-2', label: 'm' },
      { id: 'option-3', label: 'f' },
    ],
  },
  selected_option_id: 'option-1',
}

it('expands frozen listen history into the same read-only view with the saved selection', () => {
  render(<PinyinHistoryQuestion review={review} correct={false} />)
  fireEvent.click(screen.getByText('查看当时题目'))
  expect(screen.getByLabelText('听一听，选出你听到的拼音题面')).toBeInTheDocument()
  const picked = screen.getByRole('button', { name: 'p' })
  expect(picked).toHaveAttribute('aria-pressed', 'true')
  expect(picked).toBeDisabled()
  expect(screen.getByText('当时答错')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '播放读音' })).toBeEnabled()
})

it('explains absent historical media without creating a question from today’s materials', () => {
  render(<PinyinHistoryQuestion review={{ unavailable_reason: '未保存可信题目快照，无法还原历史题面' }} correct />)
  expect(screen.getByText('未保存可信题目快照，无法还原历史题面')).toBeInTheDocument()
  expect(screen.queryByLabelText('听一听，选出你听到的拼音题面')).not.toBeInTheDocument()
})

it('does not fall back to browser speech when a saved listen clip is missing', () => {
  const speak = vi.fn()
  vi.stubGlobal('speechSynthesis', { cancel: vi.fn(), speak })
  render(<PinyinHistoryQuestion review={{
    question: { ...review.question, speechUrl: undefined },
    selected_option_id: 'option-0',
  }} correct />)
  fireEvent.click(screen.getByText('查看当时题目'))
  fireEvent.click(screen.getByRole('button', { name: '播放读音' }))
  expect(screen.getByRole('alert')).toHaveTextContent('读音素材暂不可用')
  expect(speak).not.toHaveBeenCalled()
  vi.unstubAllGlobals()
})

it('keeps keyboard focus on the closed history disclosure instead of its hidden audio controls', () => {
  render(<PinyinDetailDrawer close={() => {}} retry={() => {}} isError={false} data={{
    kp_id: 1, title: 'b', module_name: '声母', subject_code: 'pinyin', kind: 'letter', skills: [],
    history: [{ at: '2026-09-12T10:00:00Z', is_correct: false, cost_ms: 1, source: 'quiz', skill_code: 'listen', pinyin_review: review }],
  } as any} />)
  expect(screen.getByRole('button', { name: '关闭拼音详情' })).toHaveFocus()
  fireEvent.keyDown(document, { key: 'Tab', shiftKey: true })
  expect(screen.getByText('查看当时题目')).toHaveFocus()
})
