import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const useChildStore = create<{ childId: number; setChildId: (id: number) => void }>()(persist(
  (set) => ({ childId: 1, setChildId: (childId) => set({ childId }) }),
  { name: 'math-child' },
))
