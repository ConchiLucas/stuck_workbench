import { appPath } from '../appPath'
import { api } from './client'
import type { EnglishGeneratedQuiz, EnglishQuizType } from './types'

export const englishApi = {
  generateQuiz: (type: EnglishQuizType, excludeTargetIds: number[]) =>
    api<EnglishGeneratedQuiz>(appPath('/api/v1/english/quiz/generate'), {
      method: 'POST',
      body: JSON.stringify({ type, excludeTargetIds }),
    }),
}

export async function generateEnglishQuizSet(type: EnglishQuizType, count = 4) {
  const questions: EnglishGeneratedQuiz[] = []
  const excludeTargetIds: number[] = []
  for (let i = 0; i < count; i++) {
    const question = await englishApi.generateQuiz(type, excludeTargetIds)
    questions.push(question)
    excludeTargetIds.push(question.targetId)
  }
  return questions
}
