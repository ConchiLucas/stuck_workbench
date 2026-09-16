export interface MathItem {
  kpId: number
  title: string
  kind: string
  payload: string
  difficulty: number
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
  glyphImageUrl: string
  speechAudioUrl: string
  speechText: string
  enabled?: boolean
}

export interface MathGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: MathItem[]
}

export interface MathListResult {
  view: string
  total: number
  groups?: MathGroup[]
  items?: MathItem[]
}

export interface MathSyncResult {
  upserted: number
  total: number
}

export interface MathBatchResult {
  generated: number
  skipped: number
  failed: number
  errors?: string[]
}

export interface MathQuestionSpeechStatus {
  moduleCode: string
  total: number
  ready: number
  missing: number
}
