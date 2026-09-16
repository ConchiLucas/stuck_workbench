import { create } from 'zustand'

export const useChildStore = create<{ childId: number; setChildId: (id: number) => void }>((set) => ({
  childId: Number(localStorage.getItem('pinyin-child-id') ?? 1),
  setChildId: (childId) => { localStorage.setItem('pinyin-child-id', String(childId)); set({ childId }) },
}))
