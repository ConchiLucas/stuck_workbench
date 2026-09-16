export type ScienceReviewStatus = 'draft' | 'reviewed' | 'published'

export interface ScienceItem {
  kpId: number
  title: string
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
  needsSenseImage: boolean
  needsSenseImageOverride: boolean | null
  effectiveNeedsSenseImage: boolean
  glyphImageUrl: string
  senseImageUrl: string
  speechAudioUrl: string
  summary: string
  explanation: string
  funFact: string
  reviewStatus: ScienceReviewStatus
  reviewedAt: string | null
  contentVersion: number
  kind?: string
  prompt?: string
  diagramUrl?: string
  hasDiagram?: boolean
  hasSpeech?: boolean
  rationale?: string
  citation?: string
  example?: unknown
}

export interface ScienceGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: ScienceItem[]
}

export interface ScienceListResult {
  view: string
  total: number
  groups?: ScienceGroup[]
  items?: ScienceItem[]
}

export interface ScienceSyncResult {
  upserted: number
  total: number
}
