import { create } from 'zustand'

type DemoAnswerStore = {
  solved: Record<string, boolean>
  mark: (questionId: string, correct: boolean) => void
  clearType: (slug: string) => void
}

export const useDemoAnswerStore = create<DemoAnswerStore>((set) => ({
  solved: {},
  mark: (questionId, correct) => set((state) => ({ solved: { ...state.solved, [questionId]: correct } })),
  clearType: (slug) => set((state) => {
    const solved = { ...state.solved }
    for (const key of Object.keys(solved)) {
      if (key.startsWith(`${slug}:`)) delete solved[key]
    }
    return { solved }
  }),
}))
