import { create } from 'zustand'
import type { PracticeType } from '../content/typePracticeBanks'

type TypePracticeStore = {
  picks: Record<string, number>
  setPick: (type: PracticeType, n: number, option: number) => void
  pick: (type: PracticeType, n: number) => number | undefined
  clearType: (type: PracticeType) => void
}

function key(type: PracticeType, n: number) {
  return `${type}:${n}`
}

export const useTypePracticeStore = create<TypePracticeStore>((set, get) => ({
  picks: {},
  setPick: (type, n, option) => set((state) => ({ picks: { ...state.picks, [key(type, n)]: option } })),
  pick: (type, n) => get().picks[key(type, n)],
  clearType: (type) => set((state) => {
    const picks = { ...state.picks }
    for (const name of Object.keys(picks)) {
      if (name.startsWith(`${type}:`)) delete picks[name]
    }
    return { picks }
  }),
}))
