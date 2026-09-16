export type LogicQuizType = 'pattern' | 'classify' | 'order' | 'shape_reason' | 'diff' | 'compare'

export type LogicGeneratedQuiz = {
  instanceId: string
  type: LogicQuizType
  stem: string
  targetId: number
  visual: { kind?: string; items?: string[] }
  options: { id: number; label?: string; emoji?: string }[]
  answerIndex: number
}
