export type PhraseCode = 'listen_zh' | 'listen_en' | 'scene' | 'reply'

export type QuestionOption = { id?: string; label: string }

export interface PlanQuestion {
  id: number
  code: PhraseCode
  type: string
  stem: string
  options: QuestionOption[]
  visual: { kind?: string; text?: string }
  speech: { text?: string; lang?: string; url?: string }
}

export type PlanSummary = {
  id: number
  planDate: string
  seqNo: number
  subjectCode: 'phrase'
  status: string
  targetCount: number
  doneCount: number
  correctCount: number
  stars: number
  durationSec: number
}

export type PlanItem = {
  id: number
  seq: number
  kpId: number
  tries: number
  phrase: string
  meaningZh: string
  scene: string
  replyTo: string
  bucket: string
  status: string
  picks: string
  optionOrder: string
  question: PlanQuestion
}

export type PlanDetail = { plan: PlanSummary; items: PlanItem[] }

export type AnswerResult = {
  correct: boolean
  answerIndex?: number
  canRetry: boolean
  tries: number
  status: string
  mastery: Record<string, unknown>
}

export type FinishResult = {
  plan: PlanSummary
  stars: number
  flowers: number
  weakPhrases: { kpId: number; phrase: string; meaningZh: string }[]
}

export type CreatePlanInput = { mode: 'type'; questionCode: PhraseCode; count?: number }
