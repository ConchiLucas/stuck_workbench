export type PlanSummary = {
  id: number; planDate: string; seqNo: number; subjectCode: 'literacy'; status: string
  targetCount: number; doneCount: number; correctCount: number; stars: number; durationSec: number
}
export type Home = {
  child: { id: number; name: string; grade: string; avatarUrl: string; flowers: number }
  currentPlan: PlanSummary | null; dueCount: number
  modules: { code: string; name: string; mastered: number; total: number }[]
}
export type Module = { code: string; name: string; orderNo: number; itemCount: number }
export type LiteracyItem = {
  kpId: number; character: string; moduleCode: string; moduleName: string
  hasGlyph: boolean; hasSense: boolean; hasSpeech: boolean; orderNo: number
}
export type SkillProgress = { code: 'glyph_sense' | 'sense_char' | 'write_char'; status: string; attempts: number; accuracy: number; streak: number }
export type ItemProgress = { kpId: number; character: string; status: string; skills: SkillProgress[] }
export type QuestionOption = { label?: string; emoji?: string; image?: string; kpId?: number; assetKind?: string; id?: string; audio?: string }
export type PlanItem = {
  id: number; seq: number; kpId: number; character: string; bucket: string; status: string; tries: number; picks: string
  optionOrder: string
  question: { versionId?: number; snapshot?: { schemaVersion?:number; interaction?:'choice'|'handwriting'; responseSchemaVersion?:number; stem: {text?: string; image?: MediaRef; audio?: MediaRef} }; id: number; code: string; type: string; stem: string; options: QuestionOption[]; visual: Record<string, unknown>; speech: Record<string, unknown> }
}
export type PlanDetail = { plan: PlanSummary; items: PlanItem[] }
export type AnswerResult = { correct: boolean; answerIndex?: number; answerOptionId?:string; canRetry: boolean; tries: number; status: string; mastery: Record<string, unknown> }

export type MediaRef = {revisionId: string; kind: "glyph" | "sense" | "speech"; sha256: string}
export type PracticeTask = {id: number; title: string; kind: string; revisionId: number; targetCount: number; planId?: number; planStatus?: string; questionTypes?:string[]}
