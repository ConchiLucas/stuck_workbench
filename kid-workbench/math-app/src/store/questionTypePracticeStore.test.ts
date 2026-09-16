import { beforeEach, describe, expect, it } from 'vitest'
import { additionEquationType } from '../content/questionTypePrototype'
import { useQuestionTypePracticeStore } from './questionTypePracticeStore'

describe('question type prototype session', () => {
  beforeEach(() => useQuestionTypePracticeStore.getState().start())

  it('starts from a clean first question', () => {
    useQuestionTypePracticeStore.setState({ index: 3, tries: 2, score: 2, answered: true, completed: true })
    useQuestionTypePracticeStore.getState().start()
    expect(useQuestionTypePracticeStore.getState()).toMatchObject({ index: 0, tries: 0, score: 0, answered: false, completed: false })
  })

  it('allows one retry before revealing a wrong answer', () => {
    expect(useQuestionTypePracticeStore.getState().choose(0)).toBe('retry')
    expect(useQuestionTypePracticeStore.getState()).toMatchObject({ tries: 1, answered: false, score: 0 })
    expect(useQuestionTypePracticeStore.getState().choose(2)).toBe('revealed')
    expect(useQuestionTypePracticeStore.getState()).toMatchObject({ tries: 2, answered: true, score: 0 })
    expect(useQuestionTypePracticeStore.getState().choose(1)).toBe('locked')
  })

  it('counts a correct answer once and advances only after settlement', () => {
    expect(useQuestionTypePracticeStore.getState().next()).toBe(false)
    expect(useQuestionTypePracticeStore.getState().choose(1)).toBe('correct')
    expect(useQuestionTypePracticeStore.getState()).toMatchObject({ score: 1, answered: true })
    expect(useQuestionTypePracticeStore.getState().choose(1)).toBe('locked')
    expect(useQuestionTypePracticeStore.getState().next()).toBe(true)
    expect(useQuestionTypePracticeStore.getState()).toMatchObject({ index: 1, tries: 0, answered: false, score: 1 })
  })

  it('marks the session complete after all five questions', () => {
    for (const question of additionEquationType.questions) {
      expect(useQuestionTypePracticeStore.getState().choose(question.answerIndex)).toBe('correct')
      expect(useQuestionTypePracticeStore.getState().next()).toBe(true)
    }
    expect(useQuestionTypePracticeStore.getState()).toMatchObject({ score: 5, completed: true, index: 4 })
  })
})
