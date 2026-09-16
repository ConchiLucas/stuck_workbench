import { create } from 'zustand'
import { pickKey } from './garden'

type PoemState = {
  picks: Record<string, string>
  sequence: Record<string, string[]>
  wrongs: Record<string, string>
  setPick: (code: string, n: number, answerId: string) => void
  recite: (code: string, n: number, id: string, expected: string[], answerId: string) => void
  clearType: (code: string) => void
}

function dropPrefix(record: Record<string, unknown>, code: string) {
  const next = { ...record }
  for (const name of Object.keys(next)) {
    if (name.startsWith(`${code}:`)) delete next[name]
  }
  return next
}

export const usePoemStore = create<PoemState>((set, get) => ({
  picks: {},
  sequence: {},
  wrongs: {},
  setPick: (code, n, answerId) => set((state) => ({
    picks: { ...state.picks, [pickKey(code, n)]: answerId },
    wrongs: { ...state.wrongs, [pickKey(code, n)]: '' },
  })),
  recite: (code, n, id, expected, answerId) => {
    const key = pickKey(code, n)
    const soFar = get().sequence[key] ?? []
    if (soFar.includes(id) || soFar.length >= expected.length) return
    const next = [...soFar, id]
    const complete = next.length === expected.length
    const ok = complete && next.every((value, index) => value === expected[index])
    set((state) => ({
      sequence: { ...state.sequence, [key]: next },
      wrongs: { ...state.wrongs, [key]: ok || !complete ? '' : id },
      picks: complete ? { ...state.picks, [key]: ok ? answerId : next.join(',') } : state.picks,
    }))
  },
  clearType: (code) => set((state) => ({
    picks: dropPrefix(state.picks, code) as Record<string, string>,
    sequence: dropPrefix(state.sequence, code) as Record<string, string[]>,
    wrongs: dropPrefix(state.wrongs, code) as Record<string, string>,
  })),
}))
