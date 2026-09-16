import { create } from 'zustand'
import { ApiError } from '../api/client'
import { pinyinApi } from '../api/pinyin'
import type { PinyinAnswerRequest, PinyinAnswerResult, PinyinGeneratedQuiz, PinyinGeneratedQuizType } from '../api/types'

export const sessionKey = (childId: number, type: PinyinGeneratedQuizType) => `${childId}:${type}`
const storageKey = (key: string) => `pinyin-practice:v1:${key}`
export type PracticeEntry = {
  question: PinyinGeneratedQuiz
  pending?: PinyinAnswerRequest
  result?: PinyinAnswerResult
  submitting?: boolean
  unavailable?: boolean
  error?: string
}
export type PracticeSession = {
  id: string
  childId: number
  type: PinyinGeneratedQuizType
  entries: PracticeEntry[]
  position: number
  status: 'loading' | 'ready' | 'error'
  verified: boolean
  error: string
}
type PracticeStore = {
  sessions: Record<string, PracticeSession | undefined>
  ensure: (childId: number, type: PinyinGeneratedQuizType) => Promise<void>
  restart: (childId: number, type: PinyinGeneratedQuizType, preserveConfirmed?: boolean) => void
  setPosition: (childId: number, type: PinyinGeneratedQuizType, id: string, position: number) => void
  submit: (childId: number, type: PinyinGeneratedQuizType, sessionId: string, instanceId: string, optionId: string, costMs: number) => Promise<PinyinAnswerResult | undefined>
}
type Dependencies = { api: Pick<typeof pinyinApi, 'generateQuiz' | 'quizInstance' | 'answerQuiz'>; storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>; count?: number }
const errorMessage = (error: unknown, fallback: string) => error instanceof ApiError ? error.message : fallback
const isUnavailable = (error: unknown) => error instanceof ApiError && [404, 410].includes(error.status)

export function createPinyinPracticeStore({ api, storage, count = 4 }: Dependencies) {
  const inflight = new Map<string, Promise<void>>()
  const save = (key: string, session: PracticeSession) => {
    try {
      storage.setItem(storageKey(key), JSON.stringify({
        ...session,
        // Verification is deliberately never trusted across a page reload.
        verified: false,
        entries: session.entries.map(({ question, pending, result }) => ({ question, pending, result })),
      }))
      return true
    } catch { return false }
  }
  const fresh = (childId: number, type: PinyinGeneratedQuizType): PracticeSession => ({
    id: crypto.randomUUID(), childId, type, entries: [], position: 1, status: 'loading', verified: false, error: '',
  })
  const restore = (childId: number, type: PinyinGeneratedQuizType): PracticeSession | undefined => {
    try {
      const raw = JSON.parse(storage.getItem(storageKey(sessionKey(childId, type))) || 'null') as PracticeSession | null
      if (!raw || raw.childId !== childId || raw.type !== type || typeof raw.id !== 'string' || !Array.isArray(raw.entries) || raw.entries.length > count) return
      if (!raw.entries.every(entry => entry.question?.type === type && typeof entry.question.instanceId === 'string' && Array.isArray(entry.question.options) && entry.question.options.every(option => typeof option.id === 'string'))) return
      return { ...raw, position: Number.isInteger(raw.position) && raw.position > 0 ? raw.position : 1, status: 'loading', verified: false, error: '', entries: raw.entries.map(({ question, pending }) => ({ question, pending })) }
    } catch { return }
  }
  return create<PracticeStore>((set, get) => {
    const update = (key: string, id: string, change: (session: PracticeSession) => PracticeSession) => {
      const current = get().sessions[key]
      if (!current || current.id !== id) return false
      const next = change(current)
      const saved = save(key, next)
      set(state => ({ sessions: { ...state.sessions, [key]: next } }))
      return saved
    }
    return {
      sessions: {},
      ensure: (childId, type) => {
        const key = sessionKey(childId, type)
        let session = get().sessions[key]
        if (session?.status === 'ready' && session.verified) return Promise.resolve()
        session ??= restore(childId, type) ?? fresh(childId, type)
        const { id } = session
        const existing = inflight.get(id)
        if (existing) return existing
        set(state => ({ sessions: { ...state.sessions, [key]: { ...session!, status: 'loading', error: '' } } }))
        const request = (async () => {
          try {
            if (!session.verified && session.entries.length) {
              const entries = await Promise.all(session.entries.map(async entry => {
                try {
                  const snapshot = await api.quizInstance(childId, entry.question.instanceId)
                  const { acceptedResult, ...question } = snapshot
                  return { question, result: acceptedResult, pending: acceptedResult ? undefined : entry.pending, error: !acceptedResult && entry.pending ? '上次提交尚未确认，请重试本次提交' : '' }
                } catch (error) {
                  if (!isUnavailable(error)) throw error
                  return { ...entry, result: undefined, unavailable: true, error: errorMessage(error, '这道题已失效，请重新练习') }
                }
              }))
              update(key, id, current => ({ ...current, entries, verified: true }))
            }
            while (get().sessions[key]?.id === id && get().sessions[key]!.entries.length < count) {
              const excludeTargetIds = get().sessions[key]!.entries.map(entry => entry.question.targetId)
              const question = await api.generateQuiz(childId, type, excludeTargetIds)
              update(key, id, current => ({ ...current, entries: [...current.entries, { question }] }))
            }
            update(key, id, current => ({ ...current, status: 'ready', verified: true }))
          } catch (error) {
            update(key, id, current => ({ ...current, status: 'error', error: errorMessage(error, '出题没有成功，点下面再试一次') }))
          } finally { inflight.delete(id) }
        })()
        inflight.set(id, request)
        return request
      },
      restart: (childId, type, preserveConfirmed = false) => {
        const key = sessionKey(childId, type)
        const previous = get().sessions[key]
        if (preserveConfirmed && previous?.entries.some(entry => entry.pending || entry.submitting)) return
        const session = fresh(childId, type)
        if (preserveConfirmed && previous?.verified) {
          session.entries = previous.entries.filter(entry => entry.result)
          session.position = Math.min(session.entries.length + 1, count)
          session.verified = true
        }
        save(key, session)
        set(state => ({ sessions: { ...state.sessions, [key]: session } }))
      },
      setPosition: (childId, type, id, position) => {
        const key = sessionKey(childId, type)
        if (get().sessions[key]?.position === position) return
        update(key, id, session => ({ ...session, position }))
      },
      submit: async (childId, type, id, instanceId, optionId, costMs) => {
        const key = sessionKey(childId, type)
        const session = get().sessions[key]
        const entry = session?.entries.find(item => item.question.instanceId === instanceId)
        if (!session || session.id !== id || !session.verified || session.status !== 'ready' || !entry || entry.submitting || entry.result || entry.unavailable) return
        if (!entry.question.options.some(option => option.id === optionId)) return
        const pending = entry.pending ?? { clientId: crypto.randomUUID(), optionId, costMs: Math.min(3_600_000, Math.max(0, Math.round(costMs))) }
        const changeEntry = (change: Partial<PracticeEntry>) => update(key, id, current => ({ ...current, entries: current.entries.map(item => item.question.instanceId === instanceId ? { ...item, ...change } : item) }))
        // Persist the exact idempotent payload before the request can leave the browser.
        if (!changeEntry({ pending, submitting: true, error: '' })) {
          changeEntry({ submitting: false, error: '无法保存本次选择，请允许浏览器存储后重试' })
          return
        }
        try {
          let result: PinyinAnswerResult
          try { result = await api.answerQuiz(childId, instanceId, pending) }
          catch (error) {
            if (!(error instanceof ApiError) || error.status !== 409) throw error
            const snapshot = await api.quizInstance(childId, instanceId)
            if (!snapshot.acceptedResult) throw error
            result = snapshot.acceptedResult
          }
          if (get().sessions[key]?.id !== id) return
          changeEntry({ result, pending: undefined, submitting: false, error: '' })
          return result
        } catch (error) {
          changeEntry({ submitting: false, unavailable: isUnavailable(error), error: errorMessage(error, '答案还未确认，请重试本次提交') })
          return
        }
      },
    }
  })
}

export const usePinyinPracticeSession = createPinyinPracticeStore({ api: pinyinApi, storage: localStorage })
