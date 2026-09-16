export interface PhraseItem {
  kpId: number
  title: string
  zh: string
  wrong: string[]
  scene: string
  replyTo: string
  difficulty: number
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
  hasSpeech?: boolean
  speechAudioUrl?: string
  speechSha256?: string
}

export interface PhraseGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: PhraseItem[]
}

export interface PhraseListResult {
  view: string
  total: number
  groups?: PhraseGroup[]
  items?: PhraseItem[]
}
