export interface ChengyuItem {
  kpId: number
  title: string
  pinyin: string
  meaning: string
  example: string
  wrong: string[]
  difficulty: number
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
  hasChengyuSpeech?: boolean
  chengyuSpeechUrl?: string
  chengyuSpeechSha256?: string
  hasMeaningSpeech?: boolean
  meaningSpeechUrl?: string
  meaningSpeechSha256?: string
  hasExampleSpeech?: boolean
  exampleSpeechUrl?: string
  exampleSpeechSha256?: string
}

export interface ChengyuGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: ChengyuItem[]
}

export interface ChengyuListResult {
  view: string
  total: number
  groups?: ChengyuGroup[]
  items?: ChengyuItem[]
}
