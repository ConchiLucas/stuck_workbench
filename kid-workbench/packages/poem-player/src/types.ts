export type PoemKind = 'title' | 'fill' | 'couplet' | 'recite'

export type PoemChoice = { id: string; label: string }
export type PoemNode = { id: string; label: string }

export type PoemExample = {
  kind: PoemKind
  prompt: string
  line?: string
  speechUrl?: string
  speechText?: string
  audioMissingReason?: string
  options?: PoemChoice[]
  answerId?: string
  optionOrder?: string[]
  workId?: string
  workTitle?: string
  author?: string
  dynasty?: string
  edition?: string
  lineId?: string
  sourceLine?: string
  gapIndexes?: number[]
  upperLineId?: string
  nextLineId?: string
  sequenceItems?: PoemNode[]
  sequenceDisplayOrder?: string[]
  correctSequence?: string[]
  explanation?: string
  tip?: string
}

export type PoemAnswer = {
  kind: PoemKind
  selectedId?: string
  sequence?: string[]
}

export const poemKinds: PoemKind[] = ['title', 'fill', 'couplet', 'recite']
export const poemTitles: Record<PoemKind, string> = {
  title: '选诗名',
  fill: '补字',
  couplet: '选下一句',
  recite: '排顺序',
}

export function isCorrect(example: PoemExample, answer?: PoemAnswer | null) {
  if (!answer) return false
  if (example.kind === 'recite') {
    const got = answer.sequence ?? []
    const want = example.correctSequence ?? []
    return got.length === want.length && got.every((id, i) => id === want[i])
  }
  return Boolean(answer.selectedId) && answer.selectedId === example.answerId
}

export function errorFacts(example: PoemExample, answer?: PoemAnswer | null) {
  if (!answer) return []
  const label = (id?: string) =>
    example.options?.find((o) => o.id === id)?.label
    || example.sequenceItems?.find((n) => n.id === id)?.label
    || id
    || ''
  if (example.kind === 'title' && answer.selectedId && answer.selectedId !== example.answerId) {
    return [`孩子选了「${label(answer.selectedId)}」，这首作品是「${label(example.answerId)}」。`]
  }
  if (example.kind === 'fill' && answer.selectedId && answer.selectedId !== example.answerId) {
    return [`孩子选了「${label(answer.selectedId)}」，缺的字是「${label(example.answerId)}」。`]
  }
  if (example.kind === 'couplet' && answer.selectedId && answer.selectedId !== example.answerId) {
    return [`孩子选了「${label(answer.selectedId)}」，下一句是「${label(example.answerId)}」。`]
  }
  if (example.kind === 'recite' && !isCorrect(example, answer) && (answer.sequence?.length ?? 0) > 0) {
    const join = (ids: string[]) => ids.map((id) => label(id)).join('→')
    return [`孩子排出的顺序是 ${join(answer.sequence ?? [])}，正确顺序是 ${join(example.correctSequence ?? [])}。`]
  }
  return []
}

export function displayFillLine(source: string, gaps: number[] = []) {
  const marks = new Set(gaps)
  return Array.from(source).map((ch, i) => (marks.has(i) ? '□' : ch)).join('')
}

export function exampleFromClue(clue: {
  kind: string
  prompt: string
  line: string
  speech?: string
  answerId: string
  options: { id: string; label: string }[]
  sequence?: string[]
  workId?: string
  lineId?: string
  sourceLine?: string
  gapIndexes?: number[]
  speechUrl?: string
}): PoemExample {
  const kind = clue.kind as PoemKind
  const source = clue.sourceLine || clue.line
  const items = (clue.sequence ?? []).map((text, i) => ({
    id: clue.options.find((o) => o.label === text || o.id === text)?.id || `${clue.workId || 'poem'}:L${i + 1}`,
    label: text,
  }))
  return {
    kind,
    prompt: clue.prompt,
    line: clue.line,
    speechUrl: clue.speechUrl,
    speechText: clue.speech,
    options: clue.options,
    answerId: kind === 'recite' ? (clue.sequence?.join(',') ? undefined : clue.answerId) : clue.answerId,
    workId: clue.workId,
    lineId: clue.lineId,
    sourceLine: source,
    gapIndexes: kind === 'fill' ? (clue.gapIndexes ?? inferGaps(source, clue.line)) : undefined,
    sequenceItems: kind === 'recite' ? (items.length ? items : clue.options.map((o) => ({ id: o.id, label: o.label }))) : undefined,
    sequenceDisplayOrder: kind === 'recite' ? clue.options.map((o) => o.id) : undefined,
    correctSequence: kind === 'recite' ? (clue.sequence ?? []) : undefined,
  }
}

function inferGaps(source: string, display: string) {
  const a = Array.from(source)
  const b = Array.from(display)
  const gaps: number[] = []
  for (let i = 0; i < Math.min(a.length, b.length); i++) {
    if (b[i] === '□') gaps.push(i)
  }
  return gaps
}
