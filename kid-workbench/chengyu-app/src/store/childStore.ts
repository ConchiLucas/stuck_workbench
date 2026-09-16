import { create } from 'zustand'

export const useChildStore = create<{ childId: number; setChildId: (id: number) => void }>((set) => ({
  childId: Number(localStorage.getItem('chengyu-child-id') ?? 1),
  setChildId: (childId) => { localStorage.setItem('chengyu-child-id', String(childId)); set({ childId }) },
}))
