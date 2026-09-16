export interface EnglishWord {
  kpId: number
  wordText: string
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
  meaningZh?: string
}

export interface EnglishGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  words: EnglishWord[]
}

export interface EnglishSentence {
  id: number
  code: string
  text: string
  tokens: string[]
  targetKpId: number
  targetWord?: string
  speechAudioUrl?: string
  contentHash?: string
}

export interface EnglishPassage {
  id: number
  code: string
  passage: string
  prompt: string
  answerKpId: number
  answerWord?: string
  optionKpIds: number[]
  contentHash?: string
}

export interface EnglishListResult {
  view: string
  total: number
  groups?: EnglishGroup[]
  words?: EnglishWord[]
  sentences?: EnglishSentence[]
  passages?: EnglishPassage[]
}

export interface EnglishSyncResult {
  upserted: number
  total: number
}
