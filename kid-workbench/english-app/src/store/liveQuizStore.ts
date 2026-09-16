import { create } from 'zustand'
import { generateEnglishQuizSet } from '../api/english'
import type { EnglishGeneratedQuiz } from '../api/types'

export type LiveQuizCard = 'audio-choice' | 'image-text'

const quizTypeByCard: Record<LiveQuizCard, 'listen' | 'look'> = {
  'audio-choice': 'listen',
  'image-text': 'look',
}

type LiveQuizStore = {
  questionsByType: Partial<Record<LiveQuizCard, EnglishGeneratedQuiz[]>>
  loadingType: LiveQuizCard | null
  error: string
  fallback: Partial<Record<LiveQuizCard, boolean>>
  ensure: (type: LiveQuizCard) => Promise<void>
  invalidate: (type: LiveQuizCard) => void
  useFallback: (type: LiveQuizCard) => void
}

const inflight = new Map<LiveQuizCard, Promise<void>>()

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
        const questions = await generateEnglishQuizSet(quizTypeByCard[type])
        set((state) => ({
          loadingType: state.loadingType === type ? null : state.loadingType,
          questionsByType: { ...state.questionsByType, [type]: questions },
        }))
      } catch (error) {
        set({
          loadingType: null,
          error: error instanceof Error ? error.message : '出题失败',
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
