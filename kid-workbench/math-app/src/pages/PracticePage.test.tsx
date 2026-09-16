import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PracticePage } from './PracticePage'

const { answer, finish, planFetch } = vi.hoisted(() => ({ answer: vi.fn(), finish: vi.fn(), planFetch: vi.fn() }))
const pendingItem = {
  itemId: 21, seq: 1, kpId: 4, bucket: 'core', status: 'pending', tries: 0,
  question: { questionId: 7, code: 'calc', stem: '3 + 2 等于几？', options: [{ label: '4' }, { label: '5' }, { label: '6' }, { label: '7' }], visual: { kind: 'equation', a: 3, b: 2, operator: '+' }, audioUrl: '/audio' },
}
vi.mock('../api/math', async () => {
  const actual = await vi.importActual<object>('../api/math')
  return { ...actual, mathApi: {
    plan: planFetch,
    startPlan: vi.fn().mockResolvedValue(undefined), answer, finish,
  } }
})

describe('math practice', () => {
  beforeEach(() => {
    answer.mockReset(); finish.mockReset(); planFetch.mockReset()
    planFetch.mockResolvedValue({ plan: { id: 9, status: 'doing', targetCount: 1, doneCount: 0, correctCount: 0 }, items: [{ ...pendingItem }] })
  })
  it('submits an option with a stable client id and moves on when correct', async () => {
    answer.mockResolvedValue({ correct: true, answerIndex: 1, canRetry: false, tries: 1, status: 'correct' })
    finish.mockResolvedValue({ id: 9, stars: 3 })
    render(<MemoryRouter initialEntries={['/practice/9']}><Routes><Route path="practice/:planId" element={<PracticePage />} /><Route path="result/:planId" element={<h1>结果页</h1>} /></Routes></MemoryRouter>)
    fireEvent.click(await screen.findByRole('button', { name: /B\s*5/ }))
    await waitFor(() => expect(answer).toHaveBeenCalled())
    expect(answer.mock.calls[0][3].clientId).toMatch(/.+/)
    expect(await screen.findByRole('heading', { name: '结果页' })).toBeInTheDocument()
  })

  it('allows one retry after the first wrong answer', async () => {
    answer.mockResolvedValueOnce({ correct: false, answerIndex: 1, canRetry: true, tries: 1, status: 'pending' })
      .mockResolvedValueOnce({ correct: true, answerIndex: 1, canRetry: false, tries: 2, status: 'correct' })
    finish.mockResolvedValue({ id: 9, stars: 2 })
    render(<MemoryRouter initialEntries={['/practice/9']}><Routes><Route path="practice/:planId" element={<PracticePage />} /><Route path="result/:planId" element={<h1>结果页</h1>} /></Routes></MemoryRouter>)
    fireEvent.click(await screen.findByRole('button', { name: /A\s*4/ }))
    expect(await screen.findByText('再想一想，你还有一次机会')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /B\s*5/ }))
    await waitFor(() => expect(answer).toHaveBeenCalledTimes(2))
  })

  it('resumes at the first unfinished item', async () => {
    planFetch.mockResolvedValue({ plan: { id: 9, status: 'doing', targetCount: 2, doneCount: 1, correctCount: 1 }, items: [
      { ...pendingItem, itemId: 20, status: 'correct', question: { ...pendingItem.question, stem: '已经做完的题' } },
      { ...pendingItem, itemId: 21, seq: 2, question: { ...pendingItem.question, stem: '还没有做的题' } },
    ] })
    render(<MemoryRouter initialEntries={['/practice/9']}><Routes><Route path="practice/:planId" element={<PracticePage />} /></Routes></MemoryRouter>)
    expect(await screen.findByRole('heading', { name: '还没有做的题' })).toBeInTheDocument()
  })

  it('can retry settlement without resubmitting the final answer', async () => {
    answer.mockResolvedValue({ correct: true, answerIndex: 1, canRetry: false, tries: 1, status: 'correct' })
    finish.mockRejectedValueOnce(new Error('temporary')).mockResolvedValueOnce({ id: 9, stars: 3 })
    render(<MemoryRouter initialEntries={['/practice/9']}><Routes><Route path="practice/:planId" element={<PracticePage />} /><Route path="result/:planId" element={<h1>结果页</h1>} /></Routes></MemoryRouter>)
    fireEvent.click(await screen.findByRole('button', { name: /B\s*5/ }))
    fireEvent.click(await screen.findByRole('button', { name: '重试结算' }))
    expect(await screen.findByRole('heading', { name: '结果页' })).toBeInTheDocument()
    expect(answer).toHaveBeenCalledTimes(1)
    expect(finish).toHaveBeenCalledTimes(2)
  })

  it('shows an explicit retry when loading the plan fails', async () => {
    planFetch.mockRejectedValueOnce(new Error('database unavailable')).mockResolvedValueOnce({ plan: { id: 9, status: 'doing', targetCount: 1, doneCount: 0, correctCount: 0 }, items: [{ ...pendingItem }] })
    render(<MemoryRouter initialEntries={['/practice/9']}><Routes><Route path="practice/:planId" element={<PracticePage />} /></Routes></MemoryRouter>)
    fireEvent.click(await screen.findByRole('button', { name: '重试加载' }))
    expect(await screen.findByRole('heading', { name: '3 + 2 等于几？' })).toBeInTheDocument()
  })
})
