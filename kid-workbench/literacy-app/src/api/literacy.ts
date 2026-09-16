import { appPath } from '../appPath'
import { api } from './client'
import type { AnswerResult, Home, ItemProgress, LiteracyItem, Module, PlanDetail, PlanSummary, PracticeTask, MediaRef } from './types'

export const literacyApi = {
  tasks: (childId: number) => api.get<{items: PracticeTask[]}>(`/children/${childId}/literacy/question-tasks`),
  claimTask: (childId: number, taskId: number, revisionId: number, claimKey: string) => api.post<PlanDetail>(`/children/${childId}/literacy/question-tasks/${taskId}/claim`, {revisionId, claimKey}),
  revisionUrl: (ref: MediaRef) => appPath(`/api/v1/literacy/material-revisions/${ref.revisionId}/media/${ref.kind}`),
  frozenUrl: (path: string) => appPath(path),
  home: (childId: number) => api.get<Home>(`/children/${childId}/literacy/home`),
  modules: () => api.get<Module[]>('/literacy/modules'),
  items: (moduleCode: string) => api.get<LiteracyItem[]>(`/literacy/modules/${moduleCode}/items`),
  item: (kpId: number) => api.get<LiteracyItem>(`/literacy/items/${kpId}`),
  progress: (childId: number) => api.get<ItemProgress[]>(`/children/${childId}/literacy/progress`),
  createPlan: (childId: number) => api.post<PlanDetail>(`/children/${childId}/literacy/plans`, {}),
  plan: (childId: number, planId: number) => api.get<PlanDetail>(`/children/${childId}/literacy/plans/${planId}`),
  startPlan: (childId: number, planId: number) => api.post<PlanDetail>(`/children/${childId}/literacy/plans/${planId}/start`),
  answer: (childId: number, planId: number, itemId: number, input: { clientId: string; optionIndex?: number; response?: import('@kid-workbench/literacy-player').PlayerResponse; costMs: number }) =>
    api.post<AnswerResult>(`/children/${childId}/literacy/plans/${planId}/items/${itemId}/answer`, input),
  finish: (childId: number, planId: number) => api.post<PlanSummary>(`/children/${childId}/literacy/plans/${planId}/finish`),
  glyphUrl: (kpId: number) => appPath(`/api/v1/literacy/items/${kpId}/glyph.png`),
  senseUrl: (kpId: number) => appPath(`/api/v1/literacy/items/${kpId}/sense.png`),
  speechUrl: (kpId: number) => appPath(`/api/v1/literacy/items/${kpId}/speech.mp3`),
}
