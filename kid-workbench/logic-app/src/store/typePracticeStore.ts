import { create } from 'zustand'
import type { PracticeType } from '../content/typePracticeBanks'

type RejectedTap = { id: string; atIndex: number }

type TypePracticeStore = {
  picks: Record<string, string>
  sequences: Record<string, string[]>
  wrongs: Record<string, string>
  rejected: Record<string, RejectedTap[]>
  setPick: (type: PracticeType, n: number, option: string) => void
  pick: (type: PracticeType, n: number) => string | undefined
  sequence: (type: PracticeType, n: number) => string[]
  wrong: (type: PracticeType, n: number) => string
  rejectedTaps: (type: PracticeType, n: number) => RejectedTap[]
  setSequence: (type: PracticeType, n: number, sequence: string[], rejected?: RejectedTap[]) => void
  appendOrder: (type: PracticeType, n: number, id: string, expected: string[]) => boolean
  clearType: (type: PracticeType) => void
}

function key(type: PracticeType, n: number) {
  return `${type}:${n}`
}

const emptySequence: string[] = []
const emptyRejected: RejectedTap[] = []

function dropType(record: Record<string, unknown>, type: PracticeType) {
  const next = { ...record }
  for (const name of Object.keys(next)) {
    if (name.startsWith(`${type}:`)) delete next[name]
  }
  return next
}

export const useTypePracticeStore = create<TypePracticeStore>((set, get) => ({
  picks: {},
  sequences: {},
  wrongs: {},
  rejected: {},
  setPick: (type, n, option) => set((state) => ({ picks: { ...state.picks, [key(type, n)]: option } })),
  pick: (type, n) => get().picks[key(type, n)],
  sequence: (type, n) => get().sequences[key(type, n)] ?? emptySequence,
  wrong: (type, n) => get().wrongs[key(type, n)] ?? '',
  rejectedTaps: (type, n) => get().rejected[key(type, n)] ?? emptyRejected,
  setSequence: (type, n, sequence, rejectedTaps) => {
    const k = key(type, n)
    set((state) => ({
      sequences: { ...state.sequences, [k]: sequence },
      rejected: rejectedTaps ? { ...state.rejected, [k]: rejectedTaps } : state.rejected,
      picks: sequence.length ? { ...state.picks, [k]: sequence.join(',') } : state.picks,
    }))
  },
  appendOrder: (type, n, id, expected) => {
    const k = key(type, n)
    const soFar = get().sequences[k] ?? []
    if (soFar.length >= expected.length) return true
    if (soFar.includes(id)) return false
    if (id === expected[soFar.length]) {
      const next = [...soFar, id]
      const done = next.length === expected.length
      set((state) => ({
        sequences: { ...state.sequences, [k]: next },
        wrongs: { ...state.wrongs, [k]: '' },
        picks: done ? { ...state.picks, [k]: id } : state.picks,
      }))
      return done
    }
    set((state) => ({
      wrongs: { ...state.wrongs, [k]: id },
      rejected: { ...state.rejected, [k]: [...(state.rejected[k] ?? []), { id, atIndex: soFar.length }] },
    }))
    return false
  },
  clearType: (type) => set((state) => ({
    picks: dropType(state.picks, type) as Record<string, string>,
    sequences: dropType(state.sequences, type) as Record<string, string[]>,
    wrongs: dropType(state.wrongs, type) as Record<string, string>,
    rejected: dropType(state.rejected, type) as Record<string, RejectedTap[]>,
  })),
}))
