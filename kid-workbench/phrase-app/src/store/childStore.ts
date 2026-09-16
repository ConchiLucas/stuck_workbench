import { create } from 'zustand'

export const useChildStore = create<{ childId: number; setChildId: (id: number) => void }>((set) => ({
  childId: Number(localStorage.getItem('phrase-child-id') ?? 1),
  setChildId: (childId) => { localStorage.setItem('phrase-child-id', String(childId)); set({ childId }) },
}))
