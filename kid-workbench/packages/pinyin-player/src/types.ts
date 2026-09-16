export type PinyinQuestionType = 'listen' | 'inword' | 'shape' | 'blend'

export type PinyinOption = {
  id: string
  label?: string
  speechText?: string
  speechUrl?: string
}

export type PinyinVisual = {
  kind: string
  text?: string
  imageUrl?: string
  initial?: string
  final?: string
  syllable?: string
}

export type PinyinQuestionView = {
  id: string
  type: PinyinQuestionType
  stem?: string
  speechText?: string
  speechUrl?: string
  visual: PinyinVisual
  options: PinyinOption[]
}
