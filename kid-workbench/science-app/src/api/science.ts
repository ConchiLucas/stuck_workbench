import { appPath } from '../appPath'
import { api } from './client'
import type { AnswerResult, Home, ItemProgress, Module, ScienceGeneratedQuiz, ScienceItem, PlanDetail, PlanSummary } from './types'

export const scienceApi = {
  home: (childId: number) => api.get<Home>(`/children/${childId}/science/home`),
  modules: () => api.get<Module[]>('/science/modules'),
  items: (moduleCode: string) => api.get<ScienceItem[]>(`/science/modules/${moduleCode}/items`),
  item: (kpId: number) => api.get<ScienceItem>(`/science/items/${kpId}`),
  progress: (childId: number) => api.get<ItemProgress[]>(`/children/${childId}/science/progress`),
  createPlan: (childId: number, mode: 'daily' | 'module' | 'review' = 'daily', moduleCode = '') => api.post<PlanDetail>(`/children/${childId}/science/plans`, { mode, moduleCode }),
  plan: (childId: number, planId: number) => api.get<PlanDetail>(`/children/${childId}/science/plans/${planId}`),
  startPlan: (childId: number, planId: number) => api.post<PlanDetail>(`/children/${childId}/science/plans/${planId}/start`),
  answer: (childId: number, planId: number, itemId: number, input: { clientId: string; optionIndex: number; costMs: number }) =>
    api.post<AnswerResult>(`/children/${childId}/science/plans/${planId}/items/${itemId}/answer`, input),
  finish: (childId: number, planId: number) => api.post<PlanSummary>(`/children/${childId}/science/plans/${planId}/finish`),
  senseUrl: (kpId: number) => appPath(`/api/v1/science/items/${kpId}/sense.png`),
  glyphUrl: (kpId: number) => appPath(`/api/v1/science/items/${kpId}/glyph.png`),
  speechUrl: (kpId: number) => appPath(`/api/v1/science/items/${kpId}/speech.mp3`),
  generateQuiz: (type: 'choice', excludeTargetIds: number[]) =>
    api.post<ScienceGeneratedQuiz>('/science/quiz/generate', { type, excludeTargetIds }),
}

export async function generateScienceQuizSet(type: 'choice' = 'choice', count = 4) {
  const questions: ScienceGeneratedQuiz[] = []
  const excludeTargetIds: number[] = []
  for (let i = 0; i < count; i++) {
    const question = await scienceApi.generateQuiz(type, excludeTargetIds)
    questions.push(question)
    excludeTargetIds.push(question.targetId)
  }
  return questions
}
