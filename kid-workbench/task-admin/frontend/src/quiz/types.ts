export type LiteracyQuizType = 'glyph_sense' | 'sense_char'
export type LiteracyQuizOptionKind = 'glyph' | 'sense' | 'char'

export interface LiteracyQuizOption {
  kpId: number
  charText: string
  kind: LiteracyQuizOptionKind
  imageUrl?: string
  speechUrl?: string
  correct: boolean
}

export interface LiteracyQuizQuestion {
  id: string
  type: LiteracyQuizType
  difficulty: 'medium' | 'easy'
  title: string
  target: {
    kpId: number
    charText: string
    glyphImageUrl?: string
    senseImageUrl?: string
    speechAudioUrl?: string
  }
  stemKind: 'glyph' | 'sense'
  optionKind: LiteracyQuizOptionKind
  speechUrl?: string
  options: LiteracyQuizOption[]
  available: boolean
  unavailableReason?: string
}
