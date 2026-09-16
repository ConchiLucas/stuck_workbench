import { createStore } from 'zustand/vanilla'

const key = (planId: number, itemId: number, tryNumber: number) => `${planId}:${itemId}:${tryNumber}`
const storageKey = 'math-pending-answers'

type PendingState = {
  ids: Record<string, string>
  clientId: (planId: number, itemId: number, tryNumber: number) => string
  acknowledge: (planId: number, itemId: number, tryNumber: number) => void
}

export function createPendingAnswerStore(storage: Storage = localStorage) {
  let initial: Record<string, string> = {}
  try { initial = JSON.parse(storage.getItem(storageKey) ?? '{}') as Record<string, string> } catch { initial = {} }
  return createStore<PendingState>((set, get) => ({
    ids: initial,
    clientId(planId, itemId, tryNumber) {
      const idKey = key(planId, itemId, tryNumber)
      const existing = get().ids[idKey]
      if (existing) return existing
      const value = crypto.randomUUID()
      const ids = { ...get().ids, [idKey]: value }
      storage.setItem(storageKey, JSON.stringify(ids))
      set({ ids })
      return value
    },
    acknowledge(planId, itemId, tryNumber) {
      const ids = { ...get().ids }
      delete ids[key(planId, itemId, tryNumber)]
      storage.setItem(storageKey, JSON.stringify(ids))
      set({ ids })
    },
  }))
}

export const pendingAnswerStore = createPendingAnswerStore()
