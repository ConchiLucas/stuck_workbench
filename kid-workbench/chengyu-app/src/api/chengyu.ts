import { api } from './client'
import type { AnswerResult, CreatePlanInput, FinishResult, PlanDetail } from './types'

export const chengyuApi = {
  createPlan: (childId: number, body: CreatePlanInput) =>
    api.post<PlanDetail>(`/children/${childId}/chengyu/plans`, body),
  plan: (childId: number, planId: number) =>
    api.get<PlanDetail>(`/children/${childId}/chengyu/plans/${planId}`),
  startPlan: (childId: number, planId: number) =>
    api.post<PlanDetail>(`/children/${childId}/chengyu/plans/${planId}/start`),
  answer: (childId: number, planId: number, itemId: number, input: { clientId: string; optionIndex: number; costMs: number }) =>
    api.post<AnswerResult>(`/children/${childId}/chengyu/plans/${planId}/items/${itemId}/answer`, input),
  finish: (childId: number, planId: number) =>
    api.post<FinishResult>(`/children/${childId}/chengyu/plans/${planId}/finish`),
}
