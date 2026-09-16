import { create } from 'zustand'
import { generateScienceQuizSet } from '../api/science'
import type { ScienceGeneratedQuiz } from '../api/types'

type LiveQuizStore = {
  questions: ScienceGeneratedQuiz[]
  loading: boolean
  error: string
  fallback: boolean
  ensure: () => Promise<void>
  invalidate: () => void
  useFallback: () => void
}

let inflight: Promise<void> | undefined

export const useLiveQuizStore = create<LiveQuizStore>((set, get) => ({
  questions: [],
  loading: false,
  error: '',
  fallback: false,
  ensure: async () => {
    if (get().fallback || get().questions.length) return
    if (inflight) return inflight
    const request = (async () => {
      set({ loading: true, error: '' })
      try {
        const questions = await generateScienceQuizSet()
        set({ loading: false, questions })
      } catch (error) {
        set({
          loading: false,
          error: error instanceof Error ? error.message : '出题没有成功，点下面再试一次',
        })
      } finally {
        inflight = undefined
      }
    })()
    inflight = request
    return request
  },
  invalidate: () => {
    inflight = undefined
    set({ questions: [], fallback: false, error: '', loading: false })
  },
  useFallback: () => {
    inflight = undefined
    set({ loading: false, fallback: true, error: '' })
  },
}))
