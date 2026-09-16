export type ScienceKind = 'choice' | 'match' | 'sequence' | 'label'

export type ScienceNode = { id: string; label: string; icon?: string; imageUrl?: string }
export type ScienceChoice = { id: string; label: string; icon?: string; imageUrl?: string }
export type ScienceLabelTarget = { id: string; label: string; x: number; y: number }

export type ScienceExample = {
  kind: ScienceKind
  prompt: string
  visual?: string
  icon?: string
  imageUrl?: string
  options?: ScienceChoice[]
  answerId?: string
  optionOrder?: string[]
  matchSources?: ScienceNode[]
  matchTargets?: ScienceNode[]
  matchAnswers?: Record<string, string>
  matchSourceOrder?: string[]
  matchTargetOrder?: string[]
  matchSourceGroup?: string
  matchTargetGroup?: string
  sequenceItems?: ScienceNode[]
  sequenceDisplayOrder?: string[]
  correctSequence?: string[]
  sequenceLoop?: boolean
  sequenceStartId?: string
  diagramKey?: string
  diagramVersion?: number
  labelTargets?: ScienceLabelTarget[]
  labelTokens?: ScienceNode[]
  labelAnswers?: Record<string, string>
  explanation?: string
  tip?: string
  rationale?: string
  listLabel?: string
}

export type ScienceAnswer = {
  kind: ScienceKind
  selectedId?: string
  pairs?: Record<string, string>
  sequence?: string[]
  labels?: Record<string, string>
}

export const scienceKinds: ScienceKind[] = ['choice', 'match', 'sequence', 'label']
export const scienceTitles: Record<ScienceKind, string> = {
  choice: '选择题',
  match: '连线题',
  sequence: '排序题',
  label: '结构标注题',
}

export function isCorrect(example: ScienceExample, answer?: ScienceAnswer | null) {
  if (!answer) return false
  if (example.kind === 'choice') return Boolean(answer.selectedId) && answer.selectedId === example.answerId
  if (example.kind === 'match') {
    const want = example.matchAnswers ?? {}
    const got = answer.pairs ?? {}
    const keys = Object.keys(want)
    return keys.length > 0 && keys.every((id) => got[id] === want[id]) && Object.keys(got).length === keys.length
  }
  if (example.kind === 'sequence') {
    const want = example.correctSequence ?? []
    const got = answer.sequence ?? []
    return want.length > 0 && want.length === got.length && want.every((id, i) => got[i] === id)
  }
  const want = example.labelAnswers ?? {}
  const got = answer.labels ?? {}
  return Object.keys(want).length > 0 && Object.keys(want).every((id) => got[id] === want[id]) && Object.keys(got).length === Object.keys(want).length
}

export function errorFacts(example: ScienceExample, answer?: ScienceAnswer | null) {
  if (!answer) return [] as string[]
  if (example.kind === 'choice' && answer.selectedId && answer.selectedId !== example.answerId) {
    const got = example.options?.find((item) => item.id === answer.selectedId)?.label ?? answer.selectedId
    const want = example.options?.find((item) => item.id === example.answerId)?.label ?? example.answerId
    return [`孩子选了「${got}」，正确答案是「${want}」。`]
  }
  if (example.kind === 'match') {
    const src = Object.fromEntries((example.matchSources ?? []).map((item) => [item.id, item.label]))
    const tgt = Object.fromEntries((example.matchTargets ?? []).map((item) => [item.id, item.label]))
    return Object.entries(example.matchAnswers ?? {}).flatMap(([from, want]) => {
      const got = answer.pairs?.[from]
      if (!got) return [`「${src[from]}」没有连线。`]
      if (got === want) return []
      return [`「${src[from]}」连到了「${tgt[got]}」，应连到「${tgt[want]}」。`]
    })
  }
  if (example.kind === 'sequence' && answer.sequence?.length) {
    const labels = Object.fromEntries((example.sequenceItems ?? []).map((item) => [item.id, item.label]))
    const show = (ids: string[]) => ids.map((id) => labels[id] ?? id).join('→')
    if (!isCorrect(example, answer)) return [`孩子排出的顺序是 ${show(answer.sequence ?? [])}，正确顺序是 ${show(example.correctSequence ?? [])}。`]
  }
  if (example.kind === 'label') {
    const tokens = Object.fromEntries((example.labelTokens ?? []).map((item) => [item.id, item.label]))
    const targets = Object.fromEntries((example.labelTargets ?? []).map((item) => [item.id, item.label]))
    return Object.entries(example.labelAnswers ?? {}).flatMap(([id, want]) => {
      const got = answer.labels?.[id]
      if (!got) return [`「${targets[id]}」位置没有放标签。`]
      if (got === want) return []
      return [`「${targets[id]}」位置放了「${tokens[got]}」，应放「${tokens[want]}」。`]
    })
  }
  return [] as string[]
}
