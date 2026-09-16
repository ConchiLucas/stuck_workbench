export type LogicKind = 'pattern' | 'classify' | 'order' | 'shape_reason' | 'diff' | 'compare'

export type LogicObject = {
  id: string
  caption: string
  glyph: string
  fill?: string
  scale?: number
  rotate?: number
  count?: number
  attrs?: Record<string, string>
}

export type LogicRule = {
  type: string
  dimension?: string
  direction?: string
  period?: number
  inGroup?: string
  explain: string
}

export type LogicExample = {
  kind: LogicKind
  prompt: string
  rule: LogicRule
  objects: LogicObject[]
  sequence?: string[]
  options?: string[]
  answerId?: string
  correctSequence?: string[]
  displayOrder?: string[]
  kpId?: number
  glyphVersion?: string
  imageUrls?: Record<string, string>
}

export type LogicAnswer = {
  kind: LogicKind
  selectedId?: string
  sequence?: string[]
  rejected?: { id: string; atIndex: number }[]
}

export const logicKinds: LogicKind[] = ['pattern', 'classify', 'order', 'shape_reason', 'diff', 'compare']
export const logicTitles: Record<LogicKind, string> = {
  pattern: '找规律',
  classify: '分类',
  order: '排序',
  shape_reason: '图形推理',
  diff: '找不同',
  compare: '比较',
}

export function objectById(objects: LogicObject[], id: string) {
  return objects.find((item) => item.id === id)
}

export function isCorrect(example: LogicExample, answer?: LogicAnswer | null) {
  if (!answer) return false
  if (example.kind === 'order') {
    const want = example.correctSequence ?? []
    const got = answer.sequence ?? []
    return want.length > 0 && want.length === got.length && want.every((id, i) => got[i] === id)
  }
  return Boolean(answer.selectedId) && answer.selectedId === example.answerId
}

export function errorFacts(example: LogicExample, answer?: LogicAnswer | null) {
  if (!answer) return [] as string[]
  if (example.kind === 'order') {
    const facts: string[] = []
    if ((answer.sequence ?? []).length) {
      facts.push(`孩子排出了「${(answer.sequence ?? []).map((id) => objectById(example.objects, id)?.caption ?? id).join(' → ')}」。`)
    }
    facts.push(`正确顺序是「${(example.correctSequence ?? []).map((id) => objectById(example.objects, id)?.caption ?? id).join(' → ')}」。`)
    for (const tap of answer.rejected ?? []) {
      facts.push(`第 ${tap.atIndex + 1} 位误点了「${objectById(example.objects, tap.id)?.caption ?? tap.id}」。`)
    }
    return facts
  }
  if (answer.selectedId && answer.selectedId !== example.answerId) {
    const got = objectById(example.objects, answer.selectedId)?.caption ?? answer.selectedId
    const want = objectById(example.objects, example.answerId ?? '')?.caption ?? example.answerId
    return [`孩子选了「${got}」，正确答案是「${want}」。`]
  }
  if (answer.selectedId && answer.selectedId === example.answerId) {
    return [`孩子选了「${objectById(example.objects, answer.selectedId)?.caption ?? answer.selectedId}」。`]
  }
  return []
}
