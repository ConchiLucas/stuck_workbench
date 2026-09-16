import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { useQuestionTypePracticeStore } from '../store/questionTypePracticeStore'
import { QuestionTypePracticePage } from './QuestionTypePracticePage'

function renderPractice() {
  return render(<MemoryRouter initialEntries={['/types/addition-equation/practice']}><Routes>
    <Route path="types/addition-equation/practice" element={<QuestionTypePracticePage />} />
    <Route path="types/addition-equation/result" element={<h1>模拟结果</h1>} />
  </Routes></MemoryRouter>)
}

describe('local question type practice', () => {
  beforeEach(() => useQuestionTypePracticeStore.getState().start())

  it('starts with the first equation and no audio control', () => {
    renderPractice()
    expect(screen.getByRole('heading', { name: '3 + 5 = ?' })).toBeInTheDocument()
    expect(screen.getByText('第 1 / 5 题')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /播放|朗读|听/ })).not.toBeInTheDocument()
  })

  it('offers one retry and then reveals the answer', () => {
    renderPractice()
    fireEvent.click(screen.getByRole('button', { name: 'A 7' }))
    expect(screen.getByText('再想一想，你还有一次机会')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'C 9' }))
    expect(screen.getByText('正确答案是 8')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^下一题/ })).toBeInTheDocument()
  })

  it('moves to the next equation after a correct answer', () => {
    renderPractice()
    fireEvent.click(screen.getByRole('button', { name: 'B 8' }))
    expect(screen.getByText('答对了')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /^下一题/ }))
    expect(screen.getByRole('heading', { name: '6 + 7 = ?' })).toBeInTheDocument()
  })

  it('navigates to the result after the fifth question', () => {
    useQuestionTypePracticeStore.setState({ index: 4, tries: 0, score: 4, answered: false, completed: false })
    renderPractice()
    fireEvent.click(screen.getByRole('button', { name: 'C 12' }))
    fireEvent.click(screen.getByRole('button', { name: /^查看结果/ }))
    expect(screen.getByRole('heading', { name: '模拟结果' })).toBeInTheDocument()
  })
})
