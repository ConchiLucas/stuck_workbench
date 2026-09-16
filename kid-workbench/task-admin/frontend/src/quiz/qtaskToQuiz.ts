import type { OptionAsset, QTaskItem } from '../api/qtaskTypes'
import type {
  LiteracyQuizOption,
  LiteracyQuizOptionKind,
  LiteracyQuizQuestion,
  LiteracyQuizType,
} from './types'

const CODE_META: Record<
  string,
  {
    type: LiteracyQuizType
    difficulty: 'medium' | 'easy'
    title: string
    stemKind: LiteracyQuizQuestion['stemKind']
    optionKind: LiteracyQuizOptionKind
  }
> = {
  glyph_sense: {
    type: 'glyph_sense',
    difficulty: 'easy',
    title: '看字图选义图',
    stemKind: 'glyph',
    optionKind: 'sense',
  },
  sense_char: {
    type: 'sense_char',
    difficulty: 'easy',
    title: '看义图选字',
    stemKind: 'sense',
    optionKind: 'char',
  },
}

function optionLabel(opt: { label?: string } | string | undefined): string {
  if (typeof opt === 'string') return opt
  if (opt && typeof opt.label === 'string') return opt.label
  return ''
}

function optionImage(kind: LiteracyQuizOptionKind, asset: OptionAsset | undefined): string | undefined {
  if (!asset) return undefined
  if (kind === 'glyph' || kind === 'char') return asset.glyphImageUrl || undefined
  return asset.senseImageUrl || undefined
}

export function qtaskItemToQuizQuestion(item: QTaskItem): LiteracyQuizQuestion | null {
  const meta = CODE_META[item.code]
  if (!meta) return null

  const byLabel = new Map<string, OptionAsset>()
  for (const asset of item.optionAssets ?? []) {
    if (asset.label) byLabel.set(asset.label, asset)
  }
  const targetAsset =
    (item.optionAssets ?? []).find((a) => a.kpId === item.kpId) ?? byLabel.get(item.charText)

  const rawOptions = Array.isArray(item.options) ? item.options : []
  const options: LiteracyQuizOption[] = rawOptions.map((opt, i) => {
    const label = optionLabel(opt)
    const asset = byLabel.get(label)
    const kpId = asset?.kpId || (i === item.answerIndex ? item.kpId : -1000 - i)
    return {
      kpId,
      charText: label || asset?.label || '?',
      kind: meta.optionKind,
      imageUrl: optionImage(meta.optionKind, asset),
      speechUrl: asset?.speechAudioUrl || (i === item.answerIndex ? item.speechAudioUrl : undefined),
      correct: i === item.answerIndex,
    }
  })

  return {
    id: `qtask-${item.questionId}-${item.seq}`,
    type: meta.type,
    difficulty: meta.difficulty,
    title: meta.title,
    target: {
      kpId: item.kpId,
      charText: item.charText || targetAsset?.label || '',
      glyphImageUrl: item.glyphImageUrl || targetAsset?.glyphImageUrl,
      senseImageUrl: item.senseImageUrl || targetAsset?.senseImageUrl,
      speechAudioUrl: item.speechAudioUrl || targetAsset?.speechAudioUrl,
    },
    stemKind: meta.stemKind,
    optionKind: meta.optionKind,
    speechUrl: item.speechAudioUrl || targetAsset?.speechAudioUrl,
    options,
    available: options.length > 0,
    unavailableReason: options.length > 0 ? undefined : '无选项',
  }
}
