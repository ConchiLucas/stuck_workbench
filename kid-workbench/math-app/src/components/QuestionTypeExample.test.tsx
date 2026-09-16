import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { getQuestionTypeDetail } from '../content/questionTypePrototype'
import { QuestionTypeExample } from './QuestionTypeExample'

function renderExample(id: string) {
  const detail = getQuestionTypeDetail(id)
  if (!detail) throw new Error(`Missing fixture: ${id}`)
  render(<QuestionTypeExample example={detail.example} />)
}

describe('question type example', () => {
  it('shows a choice example without revealing the answer', () => {
    renderExample('addition-equation')
    expect(screen.getByLabelText('3 + 5 = ?')).toBeInTheDocument()
    expect(screen.getByText('8')).toBeInTheDocument()
    expect(screen.queryByText('正确答案')).not.toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('shows object groups without making them interactive', () => {
    renderExample('addition-story')
    expect(screen.getByText('一共有几颗星星？')).toBeInTheDocument()
    expect(screen.getByText('★★★')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('shows a true or false choice for a single judgement equation', () => {
    renderExample('addition-judge')
    expect(screen.queryByText('判断下面的算式')).not.toBeInTheDocument()
    expect(screen.getByLabelText('7 + 6 = 12')).toBeInTheDocument()
    expect(screen.getByLabelText('对')).toBeInTheDocument()
    expect(screen.getByLabelText('错')).toBeInTheDocument()
    expect(screen.queryByText('对')).not.toBeInTheDocument()
    expect(screen.queryByText('错')).not.toBeInTheDocument()
    expect(screen.queryByText(/7 \+ 6 = 13/)).not.toBeInTheDocument()
  })

  it('shows the candidate equations without explaining the answer', () => {
    renderExample('subtraction-error')
    expect(screen.getByLabelText('14 − 6 = 9')).toBeInTheDocument()
    expect(screen.queryByText(/正确结果是 8/)).not.toBeInTheDocument()
  })

  it('shows both buckets in a shape classification example', () => {
    renderExample('shape-sort')
    expect(screen.getByText('没有角')).toBeInTheDocument()
    expect(screen.getByText('有角')).toBeInTheDocument()
    expect(screen.getByText('△')).toBeInTheDocument()
  })
})
