import { api } from './client'
import type { LogicGeneratedQuiz, LogicQuizType } from './types'

export const logicApi = {
  generateQuiz: (type: LogicQuizType, excludeTargetIds: number[]) =>
    api.post<LogicGeneratedQuiz>('/logic/quiz/generate', { type, excludeTargetIds }),
}

export async function generateLogicQuizSet(type: LogicQuizType, count = 4) {
  const questions: LogicGeneratedQuiz[] = []
  const excludeTargetIds: number[] = []
  for (let i = 0; i < count; i++) {
    const question = await logicApi.generateQuiz(type, excludeTargetIds)
    questions.push(question)
    excludeTargetIds.push(question.targetId)
  }
  return questions
}
