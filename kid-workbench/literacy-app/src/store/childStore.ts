import { create } from 'zustand'

export const useChildStore = create<{ childId: number; setChildId: (id: number) => void }>((set) => ({
  childId: Number(localStorage.getItem('literacy-child-id') ?? import.meta.env.VITE_CHILD_ID ?? 1),
  setChildId: (childId) => { localStorage.setItem('literacy-child-id', String(childId)); set({ childId }) },
}))
