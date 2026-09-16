export type ChengyuCode = 'meaning' | 'pick' | 'pinyin' | 'example'

export type QuestionOption = { id?: string; label: string }

export interface PlanQuestion {
  id: number
  code: ChengyuCode
  type: string
  stem: string
  options: QuestionOption[]
  visual: { kind?: string; text?: string; full?: string; blanked?: string; target?: string; start?: number; length?: number }
  speech: { text?: string; lang?: string; url?: string }
}

export type PlanSummary = {
  id: number
  planDate: string
  seqNo: number
  subjectCode: 'chengyu'
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
  chengyu: string
  pinyin: string
  meaning: string
  example: string
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
  weakChengyu: { kpId: number; chengyu: string; meaning: string }[]
}

export type CreatePlanInput = { mode: 'type'; questionCode: ChengyuCode; count?: number }
