import { appPath } from '../appPath'
import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from './client'
import type { AnswerInput, AnswerResult, CreatePlanInput, Home, LearningItem, MathGeneratedQuiz, MathModule, MathQuizType, ModuleCode, PlanDetail, Progress, QuestionCode, StageCode, StudyPlan } from './types'

export const learningAudioURL = (childId: number, kpId: number, code: QuestionCode) =>
  appPath(`/api/v1/children/${childId}/math/items/${kpId}/audio/${code}.mp3`)

export const mathApi = {
  home: (childId: number) => api.get<Home>(`/children/${childId}/math/home`),
  progress: (childId: number) => api.get<Progress>(`/children/${childId}/math/progress`),
  modules: () => api.get<MathModule[]>('/math/modules'),
  module: (code: ModuleCode) => api.get<MathModule>(`/math/modules/${code}`),
  stage: (code: ModuleCode, stage: StageCode) => api.get<LearningItem[]>(`/math/modules/${code}/stages/${stage}`),
  createPlan: (childId: number, input: CreatePlanInput) => api.post<PlanDetail>(`/children/${childId}/math/plans`, input),
  plan: (childId: number, planId: number) => api.get<PlanDetail>(`/children/${childId}/math/plans/${planId}`),
  startPlan: (childId: number, planId: number) => api.post<PlanDetail>(`/children/${childId}/math/plans/${planId}/start`),
  answer: (childId: number, planId: number, itemId: number, input: AnswerInput) => api.post<AnswerResult>(`/children/${childId}/math/plans/${planId}/items/${itemId}/answer`, input),
  finish: (childId: number, planId: number) => api.post<StudyPlan>(`/children/${childId}/math/plans/${planId}/finish`),
  generateQuiz: (type: MathQuizType, excludeTargetIds: number[]) =>
    api.post<MathGeneratedQuiz>('/math/quiz/generate', { type, excludeTargetIds }),
}

export async function generateMathQuizSet(type: MathQuizType, count = 4) {
  const questions: MathGeneratedQuiz[] = []
  const excludeTargetIds: number[] = []
  for (let i = 0; i < count; i++) {
    const question = await mathApi.generateQuiz(type, excludeTargetIds)
    questions.push(question)
    excludeTargetIds.push(question.targetId)
  }
  return questions
}

export const useMathHome = (childId: number) => useQuery({ queryKey: ['math', 'home', childId], queryFn: () => mathApi.home(childId) })
export const useMathProgress = (childId: number) => useQuery({ queryKey: ['math', 'progress', childId], queryFn: () => mathApi.progress(childId) })
export const useMathModules = () => useQuery({ queryKey: ['math', 'modules'], queryFn: mathApi.modules })
export const useMathModule = (code: ModuleCode) => useQuery({ queryKey: ['math', 'module', code], queryFn: () => mathApi.module(code) })
export const useMathStage = (code: ModuleCode, stage: StageCode) => useQuery({ queryKey: ['math', 'stage', code, stage], queryFn: () => mathApi.stage(code, stage) })
export const useMathPlan = (childId: number, planId: number) => useQuery({ queryKey: ['math', 'plan', childId, planId], queryFn: () => mathApi.plan(childId, planId) })
export const useCreatePlan = (childId: number) => useMutation({ mutationFn: (input: CreatePlanInput) => mathApi.createPlan(childId, input) })
