export type StatusCounts = {
  not_started: number
  learning: number
  shaky: number
  mastered: number
  review_due: number
}

export type Child = {
  id: number
  name: string
  grade: string
}

export type SubjectHealth = {
  code: string
  name: string
  icon: string
  total: number
  counts: StatusCounts
  health: 'ok' | 'watch' | 'weak' | string
  summary: string
}

export type UrgentItem = {
  kp_id: number
  title: string
  subject_code: string
  subject_name: string
  module_name: string
  status: string
  accuracy: number
  wrong_count: number
  reason: string
}

export type Overview = {
  child: Child
  headline: string
  subjects: SubjectHealth[]
  urgent: UrgentItem[] | null
}

export type SkillGap = {
  kp_id: number
  title: string
  strong_skill: string
  strong_label: string
  weak_skill: string
  weak_label: string
}

export type ModuleHealth = {
  code: string
  name: string
  total: number
  counts: StatusCounts
}

export type SubjectDiagnosis = {
  code: string
  name: string
  icon: string
  health: string
  headline: string
  counts: StatusCounts
  total: number
  modules: ModuleHealth[] | null
  skill_gaps: SkillGap[] | null
}

export type HistoryItem = {
  at: string
  is_correct: boolean
  cost_ms: number
  source: string
  skill_code?: string
  skill_label?: string
}

export type SkillState = {
  code: string
  label: string
  status: string
  accuracy: number
  attempts: number
}

export type RecentWrong = {
  picks: string
  answer: string
  answered_at: string | null
}

export type KpArchive = {
  kp_id: number
  title: string
  subject_code: string
  subject_name: string
  module_name: string
  status: string
  attempts: number
  accuracy: number
  skills: SkillState[] | null
  history: HistoryItem[] | null
  recent_wrong: RecentWrong[] | null
}

export type ErrorPattern = {
  kind: string
  title: string
  detail: string
  kp_id?: number
  kp_title?: string
  subject_code?: string
}

export type ErrorPatternsResponse = {
  patterns: ErrorPattern[] | null
}
