import { create } from 'zustand'

const KEY = 'progress-child-id'

function readChildId() {
  if (typeof window === 'undefined') return 1
  const fromQuery = Number(new URLSearchParams(window.location.search).get('child'))
  if (Number.isFinite(fromQuery) && fromQuery > 0) {
    localStorage.setItem(KEY, String(fromQuery))
    return fromQuery
  }
  const n = Number(localStorage.getItem(KEY) || 1)
  return Number.isFinite(n) && n > 0 ? n : 1
}

interface ChildState {
  childId: number
  kpDrawerId: number | null
  setChild: (id: number) => void
  openKp: (id: number) => void
  closeKp: () => void
}

export const useChildStore = create<ChildState>((set) => ({
  childId: readChildId(),
  kpDrawerId: null,
  setChild: (childId) => {
    localStorage.setItem(KEY, String(childId))
    set({ childId })
  },
  openKp: (kpDrawerId) => set({ kpDrawerId }),
  closeKp: () => set({ kpDrawerId: null }),
}))
