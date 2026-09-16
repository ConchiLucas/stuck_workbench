export interface PinyinItem {
  kpId: number
  letter: string
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
  soloText: string
  wordText: string
	wordExamples?: string[]
  wordExampleSpeechUrls?: Record<string, string>
  soloSpeechUrl: string
  wordSpeechUrl: string
  glyphImageUrl: string
}

export interface PinyinGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: PinyinItem[]
}

export interface PinyinListResult {
  view: string
  total: number
  groups?: PinyinGroup[]
  items?: PinyinItem[]
}

export interface PinyinSyncResult {
  upserted: number
  total: number
}

export interface PinyinBatchResult {
  generated: number
  skipped: number
  failed: number
  errors?: string[]
}

export type PinyinGeneratedQuizType = 'listen' | 'inword' | 'shape' | 'blend'

export interface PinyinGeneratedQuizOption {
  id: number
  label?: string
  speechText?: string
  speechUrl?: string
}

export interface PinyinGeneratedQuiz {
  instanceId: string
  type: PinyinGeneratedQuizType
  stem: string
  targetId: number
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
  answerIndex: number
}
