import { create } from 'zustand'
import { additionEquationType } from '../content/questionTypePrototype'

export type ChoiceResult = 'correct' | 'retry' | 'revealed' | 'locked'

interface QuestionTypePracticeState {
  index: number
  tries: number
  score: number
  answered: boolean
  completed: boolean
  start: () => void
  choose: (optionIndex: number) => ChoiceResult
  next: () => boolean
}

const cleanSession = { index: 0, tries: 0, score: 0, answered: false, completed: false }

export const useQuestionTypePracticeStore = create<QuestionTypePracticeState>((set, get) => ({
  ...cleanSession,
  start: () => set(cleanSession),
  choose: (optionIndex) => {
    const state = get()
    if (state.answered || state.completed) return 'locked'
    const question = additionEquationType.questions[state.index]
    if (optionIndex === question.answerIndex) {
      set({ answered: true, score: state.score + 1 })
      return 'correct'
    }
    if (state.tries === 0) {
      set({ tries: 1 })
      return 'retry'
    }
    set({ tries: 2, answered: true })
    return 'revealed'
  },
  next: () => {
    const state = get()
    if (!state.answered) return false
    if (state.index === additionEquationType.questions.length - 1) {
      set({ completed: true })
      return true
    }
    set({ index: state.index + 1, tries: 0, answered: false })
    return true
  },
}))
