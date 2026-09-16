import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { useQuestionTypePracticeStore } from '../store/questionTypePracticeStore'
import { QuestionTypeResultPage } from './QuestionTypeResultPage'

describe('local question type result', () => {
  beforeEach(() => useQuestionTypePracticeStore.getState().start())

  it('shows the completed score and can reset for another round', () => {
    useQuestionTypePracticeStore.setState({ completed: true, score: 4 })
    render(<MemoryRouter><QuestionTypeResultPage /></MemoryRouter>)
    expect(screen.getByText('答对 4 / 5 题')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('link', { name: '再练一次' }))
    expect(useQuestionTypePracticeStore.getState()).toMatchObject({ completed: false, score: 0, index: 0 })
  })

  it('does not invent a score for a direct visit', () => {
    render(<MemoryRouter><QuestionTypeResultPage /></MemoryRouter>)
    expect(screen.getByRole('heading', { name: '先完成一次练习' })).toBeInTheDocument()
    expect(screen.queryByText(/答对 .* 题/)).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '查看题型详情' })).toHaveAttribute('href', '/types/addition-equation')
  })
})
