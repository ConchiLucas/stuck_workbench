import { create } from 'zustand'

type State = { ids: Record<string, string>; get: (key: string) => string; clear: (key: string) => void }

export const usePendingAnswerStore = create<State>((set, get) => ({
  ids: {},
  get: (key) => {
    const existing = get().ids[key]
    if (existing) return existing
    const id = crypto.randomUUID()
    set((state) => ({ ids: { ...state.ids, [key]: id } }))
    return id
  },
  clear: (key) => set((state) => { const ids = { ...state.ids }; delete ids[key]; return { ids } }),
}))
