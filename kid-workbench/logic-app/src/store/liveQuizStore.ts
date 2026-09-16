import { create } from 'zustand'
import { generateLogicQuizSet } from '../api/logic'
import type { LogicGeneratedQuiz } from '../api/types'
import type { PracticeType } from '../content/typePracticeBanks'

type LiveQuizStore = {
  questionsByType: Partial<Record<PracticeType, LogicGeneratedQuiz[]>>
  loadingType: PracticeType | null
  error: string
  fallback: Partial<Record<PracticeType, boolean>>
  ensure: (type: PracticeType) => Promise<void>
  invalidate: (type: PracticeType) => void
  useFallback: (type: PracticeType) => void
}

const inflight = new Map<PracticeType, Promise<void>>()

export const useLiveQuizStore = create<LiveQuizStore>((set, get) => ({
  questionsByType: {},
  loadingType: null,
  error: '',
  fallback: {},
  ensure: async (type) => {
    if (get().fallback[type] || get().questionsByType[type]?.length) return
    const pending = inflight.get(type)
    if (pending) return pending
    const request = (async () => {
      set({ loadingType: type, error: '' })
      try {
        const questions = await generateLogicQuizSet(type)
        set((state) => ({
          loadingType: state.loadingType === type ? null : state.loadingType,
          questionsByType: { ...state.questionsByType, [type]: questions },
        }))
      } catch (error) {
        set({
          loadingType: null,
          error: error instanceof Error ? error.message : '出题没有成功，点下面再试一次',
        })
      } finally {
        inflight.delete(type)
      }
    })()
    inflight.set(type, request)
    return request
  },
  invalidate: (type) => {
    inflight.delete(type)
    set((state) => {
      const questionsByType = { ...state.questionsByType }
      const fallback = { ...state.fallback }
      delete questionsByType[type]
      delete fallback[type]
      return { questionsByType, fallback, error: '' }
    })
  },
  useFallback: (type) => {
    inflight.delete(type)
    set((state) => ({
      loadingType: state.loadingType === type ? null : state.loadingType,
      fallback: { ...state.fallback, [type]: true },
      error: '',
    }))
  },
}))
