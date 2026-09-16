export type PlanSummary = {
  id: number; planDate: string; seqNo: number; subjectCode: 'pinyin'; status: string
  targetCount: number; doneCount: number; correctCount: number; stars: number; durationSec: number
}
export type Home = {
  child: { id: number; name: string; grade: string; avatarUrl: string; flowers: number }
  currentPlan: PlanSummary | null; dueCount: number
  modules: { code: string; name: string; mastered: number; total: number }[]
}
export type Module = { code: string; name: string; orderNo: number; itemCount: number }
export type PinyinItem = {
  kpId: number; letter: string; moduleCode: string; moduleName: string; soloText: string; wordText: string
  hasSoloSpeech: boolean; hasWordSpeech: boolean; hasGlyph: boolean; orderNo: number
}
export type SkillProgress = { code: PinyinGeneratedQuizType; status: string; attempts: number; accuracy: number; streak: number }
export type ItemProgress = { kpId: number; letter: string; status: string; skills: SkillProgress[] }
export type QuestionOption = { label?: string; emoji?: string; image?: string }
export type PlanItem = {
  id: number; seq: number; kpId: number; letter: string; bucket: string; status: string; tries: number; picks: string
  optionOrder: string; question: { id: number; code: string; type: string; stem: string; options: QuestionOption[]; visual: Record<string, unknown>; speech: Record<string, unknown> }
}
export type PlanDetail = { plan: PlanSummary; items: PlanItem[] }
export type AnswerResult = { correct: boolean; answerIndex: number; canRetry: boolean; tries: number; status: string; mastery: Record<string, unknown> }

export type PinyinGeneratedQuizType = 'listen' | 'inword' | 'shape' | 'blend'

export type PinyinGeneratedQuizOption = {
  id: string
  label?: string
  speechText?: string
  speechUrl?: string
}

export type PinyinGeneratedQuiz = {
  instanceId: string
  type: PinyinGeneratedQuizType
  stem: string
  targetId: number
  kpId: number
  expiresAt: string
  speechText?: string
  speechUrl?: string
  visual: {
    kind: string
    text?: string
    imageUrl?: string
    initial?: string
    final?: string
    syllable?: string
  }
  options: PinyinGeneratedQuizOption[]
}

export type PinyinAnswerRequest = { clientId: string; optionId: string; costMs: number }
export type PinyinAnswerResult = {
  instanceId: string
  attemptId: number
  selectedOptionId: string
  correct: boolean
  answerOptionId: string
  skill: { code: PinyinGeneratedQuizType; status: string }
  knowledge: { kpId: number; status: string; newlyMastered: boolean }
}
export type PinyinInstanceSnapshot = PinyinGeneratedQuiz & { acceptedResult?: PinyinAnswerResult }
