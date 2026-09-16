export type PlanSummary = {
  id: number; planDate: string; seqNo: number; subjectCode: 'science'; status: string
  targetCount: number; doneCount: number; correctCount: number; stars: number; durationSec: number
}
export type Home = {
  child: { id: number; name: string; grade: string; avatarUrl: string; flowers: number }
  currentPlan: PlanSummary | null; dueCount: number; exploredCount: number
  modules: { code: string; name: string; mastered: number; total: number }[]
}
export type Module = { code: string; name: string; order: number; total: number }
export type ScienceItem = {
  kpId: number; title: string; moduleCode: string; moduleName: string; difficulty: number
  summary: string; explanation: string; funFact: string; contentVersion: number
  senseImageUrl?: string; glyphImageUrl?: string; speechUrl?: string; hasPractice: boolean; orderNo: number
}
export type SkillProgress = { code: 'recognize'; status: string; attempts: number; accuracy: number; streak: number }
export type ItemProgress = { kpId: number; title: string; status: string; skills: SkillProgress[] }
export type QuestionOption = { label?: string; emoji?: string; image?: string }
export type PlanItem = {
  id: number; seq: number; kpId: number; title: string; bucket: string; status: string; tries: number; picks: string
  optionOrder: string; question: { id: number; code: 'recognize'; type: string; stem: string; options: QuestionOption[]; visual: Record<string, unknown>; speech: Record<string, unknown> }
}
export type PlanDetail = { plan: PlanSummary; items: PlanItem[] }
export type AnswerResult = { correct: boolean; answerIndex: number; canRetry: boolean; tries: number; status: string; explanation?: string; mastery: Record<string, unknown> }
export type ScienceQuizType = 'choice'
export type ScienceGeneratedQuiz = {
  instanceId: string
  type: ScienceQuizType
  stem: string
  targetId: number
  visual: { kind: string; key?: string; text?: string; emoji?: string }
  options: { id: number; label?: string }[]
  answerIndex: number
}
