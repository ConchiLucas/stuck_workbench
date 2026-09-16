import { appPath } from '../appPath'
import { api } from './client'
import type { AnswerResult, PinyinAnswerRequest, PinyinAnswerResult, PinyinInstanceSnapshot, Home, ItemProgress, Module, PinyinGeneratedQuiz, PinyinGeneratedQuizType, PinyinItem, PlanDetail, PlanSummary } from './types'

export const pinyinApi = {
  home: (childId: number) => api.get<Home>(`/children/${childId}/pinyin/home`),
  modules: () => api.get<Module[]>('/pinyin/modules'),
  items: (moduleCode: string) => api.get<PinyinItem[]>(`/pinyin/modules/${moduleCode}/items`),
  item: (kpId: number) => api.get<PinyinItem>(`/pinyin/items/${kpId}`),
  progress: (childId: number) => api.get<ItemProgress[]>(`/children/${childId}/pinyin/progress`),
  createPlan: (childId: number) => api.post<PlanDetail>(`/children/${childId}/pinyin/plans`, {}),
  plan: (childId: number, planId: number) => api.get<PlanDetail>(`/children/${childId}/pinyin/plans/${planId}`),
  startPlan: (childId: number, planId: number) => api.post<PlanDetail>(`/children/${childId}/pinyin/plans/${planId}/start`),
  answer: (childId: number, planId: number, itemId: number, input: { clientId: string; optionIndex: number; costMs: number }) =>
    api.post<AnswerResult>(`/children/${childId}/pinyin/plans/${planId}/items/${itemId}/answer`, input),
  finish: (childId: number, planId: number) => api.post<PlanSummary>(`/children/${childId}/pinyin/plans/${planId}/finish`),
  generateQuiz: (childId: number, type: PinyinGeneratedQuizType, excludeTargetIds: number[]) =>
    api.post<PinyinGeneratedQuiz>(`/children/${childId}/pinyin/quiz/generate`, { type, excludeTargetIds }),
  quizInstance: (childId: number, instanceId: string) =>
    api.get<PinyinInstanceSnapshot>(`/children/${childId}/pinyin/quiz/${encodeURIComponent(instanceId)}`),
  answerQuiz: (childId: number, instanceId: string, input: PinyinAnswerRequest) =>
    api.post<PinyinAnswerResult>(`/children/${childId}/pinyin/quiz/${encodeURIComponent(instanceId)}/answer`, input),
  glyphUrl: (kpId: number) => appPath(`/api/v1/pinyin/items/${kpId}/glyph.png`),
  speechUrl: (kpId: number, kind: 'solo' | 'word') => appPath(`/api/v1/pinyin/items/${kpId}/speech/${kind}.mp3`),
}

