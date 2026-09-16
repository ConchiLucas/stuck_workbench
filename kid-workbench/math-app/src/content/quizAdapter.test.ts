import { describe, expect, it } from 'vitest'
import { isRedundantVisual, quizToPractice } from './quizAdapter'

describe('isRedundantVisual', () => {
  it('hides an equation fragment that already appears in the stem', () => {
    expect(isRedundantVisual('5 + 7 = ?', '5 + 7')).toBe(true)
    expect(isRedundantVisual('3 + 5 = ?', '3 + 5')).toBe(true)
    expect(isRedundantVisual('3 + □ = 8', '3 + □ = 8')).toBe(true)
  })

  it('keeps story and shape pictures that are not the stem', () => {
    expect(isRedundantVisual('一共有几颗星星？', '★★★  ★★')).toBe(false)
    expect(isRedundantVisual('这是什么图形？', '△')).toBe(false)
  })
})

describe('quizToPractice', () => {
  it('turns a generated equation into a stem plus a short visual', () => {
    const question = quizToPractice({
      instanceId: 'eq-1',
      type: 'equation',
      stem: '5 + 7 = ?',
      targetId: 1,
      visual: { kind: 'add', a: 5, b: 7 },
      options: [{ id: 0, label: '12' }, { id: 1, label: '11' }, { id: 2, label: '10' }, { id: 3, label: '9' }],
      answerIndex: 1,
    })
    expect(question.prompt).toBe('5 + 7 = ?')
    expect(question.visual).toBe('5 + 7')
    expect(isRedundantVisual(question.prompt, question.visual)).toBe(true)
  })
})
