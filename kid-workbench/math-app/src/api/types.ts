export type MasteryStatus = 'not_started' | 'learning' | 'shaky' | 'review_due' | 'mastered'
export type ModuleCode = 'add10' | 'sub10' | 'shape'
export type StageCode = 'within5' | 'within10' | 'within20' | 'basic-shapes'
export type QuestionCode = 'calc' | 'story' | 'find' | 'name'
export type PlanKind = 'daily' | 'module' | 'review'
export type ShapeKey = 'circle' | 'square' | 'rect' | 'triangle' | 'oval' | 'trapezoid' | 'rhombus' | 'star'

export type MathVisual =
  | { kind: 'equation'; a: number; b: number; operator: '+' | '-' }
  | { kind: 'add' | 'sub'; leftCount: number; rightCount: number; object: string }
  | { kind: 'shape'; shape: ShapeKey }
  | { kind: 'none' }

export interface Stage { code: StageCode; name: string; itemCount: number }
export interface MathModule { code: ModuleCode; name: string; orderNo: number; itemCount: number; stages: Stage[] }
export interface LearningItem { kpId: number; title: string; moduleCode: ModuleCode; stageCode: StageCode; kind: 'add' | 'sub' | 'shape'; a?: number; b?: number; shape?: ShapeKey }
export interface SkillProgress { code: QuestionCode; status: MasteryStatus }
export interface ItemProgress { kpId: number; title: string; status: MasteryStatus; skills: SkillProgress[] }
export interface StageProgress { code: StageCode; items: ItemProgress[] }
export interface ModuleProgress { code: ModuleCode; name: string; stages: StageProgress[] }
export interface Progress { modules: ModuleProgress[] }

export interface ChildSummary { id: number; name: string; flowers: number }
export interface HomePlan { id: number; status: string; targetCount: number; doneCount: number; correctCount: number }
export interface HomeModule { code: ModuleCode; name: string; itemCount: number; masteredCount: number }
export interface Home { child: ChildSummary; plan: HomePlan | null; dueCount: number; modules: HomeModule[] }

export interface PlanQuestion { questionId: number; code: QuestionCode; stem: string; options: Array<{ label?: string; shape?: ShapeKey }>; visual: MathVisual; audioUrl: string }
export interface StudyPlan { id: number; planDate: string; seqNo: number; subjectCode: 'math'; planKind: PlanKind; moduleCode?: ModuleCode; stageCode?: StageCode; status: string; targetCount: number; doneCount: number; correctCount: number; stars: number; durationSec: number }
export interface PlanItem { itemId: number; seq: number; kpId: number; bucket: string; status: string; tries: number; question: PlanQuestion }
export interface PlanDetail { plan: StudyPlan; items: PlanItem[] }
export interface CreatePlanInput { kind: PlanKind; moduleCode?: ModuleCode; stageCode?: StageCode }
export interface AnswerInput { clientId: string; optionIndex: number; costMs: number }
export interface AnswerResult { correct: boolean; answerIndex: number; canRetry: boolean; tries: number; status: string; mastery: { kp_id: number; status: MasteryStatus }; plan: { status: string; targetCount: number; doneCount: number; correctCount: number } }

export type MathQuizType = 'equation' | 'story' | 'missing' | 'judge' | 'shape'
export type MathGeneratedQuiz = {
  instanceId: string
  type: MathQuizType
  stem: string
  targetId: number
  visual: { kind: string; text?: string; a?: number; b?: number; emoji?: string }
  options: { id: number; label?: string }[]
  answerIndex: number
}
