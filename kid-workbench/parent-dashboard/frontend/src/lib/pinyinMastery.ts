import type { MatrixModule, MatrixPoint, MatrixSkill } from '../api/types'

export const PINYIN_TYPES = [
  { code: 'listen', short: '听', name: '听音选字母' },
  { code: 'inword', short: '找', name: '字中找拼音' },
  { code: 'shape', short: '认', name: '看形认读' },
  { code: 'blend', short: '拼', name: '声韵拼读' },
] as const
export const LETTER_TYPES = PINYIN_TYPES.slice(0, 3)
export type PinyinSkillCode = typeof PINYIN_TYPES[number]['code']
export type PinyinView = 'all' | Exclude<PinyinSkillCode, 'blend'>
export type PinyinPoint = MatrixPoint & { kind: 'letter' | 'syllable' }
export type PinyinGroup = { code: string; name: string; points: PinyinPoint[]; visiblePoints: PinyinPoint[] }
export type PinyinCounts = { mastered: number; answered: number; unattempted: number; total: number }
const record = (value: unknown): value is Record<string, unknown> => !!value && typeof value === 'object'
const finite = (value: unknown) => typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0

export function pinyinState(point: { kind?: string; skills?: MatrixSkill[] | null }) {
  const types = point.kind === 'syllable' ? PINYIN_TYPES.slice(3) : !point.kind || point.kind === 'letter' ? LETTER_TYPES : []
  const skills = types.map(type => {
    const raw = Array.isArray(point.skills) ? point.skills.find(s => s?.code === type.code) : undefined
    const lit = raw?.status === 'mastered' || raw?.status === 'review_due'
    const attempts = finite(raw?.attempts)
    return { ...type, status: raw?.status ?? 'not_started', accuracy: Math.min(1, finite(raw?.accuracy)), attempts, lit,
      detail: lit ? '已掌握' : attempts > 0 ? '已作答未掌握' : '未作答' }
  })
  const mastered = skills.filter(s => s.lit).length
  const complete = skills.length > 0 && mastered === skills.length
  return { skills, mastered, complete, label: complete ? '完全掌握' : mastered ? '部分题型掌握' : '尚未掌握' }
}

export function pinyinSummary(points: { kind?: string; skills?: MatrixSkill[] | null }[]) {
  const skills = points.flatMap(p => pinyinState(p).skills)
  return Object.fromEntries(PINYIN_TYPES.map(({ code }) => {
    const applicable = skills.filter(s => s.code === code)
    return [code, { mastered: applicable.filter(s => s.lit).length,
      answered: applicable.filter(s => !s.lit && s.attempts > 0).length,
      unattempted: applicable.filter(s => !s.lit && s.attempts === 0).length, total: applicable.length }]
  })) as Record<PinyinSkillCode, PinyinCounts>
}

export function barWidths(counts: PinyinCounts) {
  const total = finite(counts.total)
  const mastered = total ? Math.min(total, finite(counts.mastered)) / total * 100 : 0
  const answered = total ? Math.min(100 - mastered, finite(counts.answered) / total * 100) : 0
  return { mastered, answered }
}

// Explicit tone mapping preserves the diaeresis: ü is a vowel distinct from u.
const TONES: Record<string, string> = Object.fromEntries([
  ['āáǎà', 'a'], ['ēéěè', 'e'], ['īíǐì', 'i'], ['ōóǒò', 'o'], ['ūúǔù', 'u'], ['ǖǘǚǜ', 'ü'], ['ńňǹ', 'n'], ['ḿ', 'm'],
].flatMap(([chars, base]) => [...chars].map(char => [char, base])))
const canonical = (value: string) => value.trim().toLowerCase().normalize('NFC').replace(/u:|v/g, 'ü')
export function normalizePinyin(value: string) {
  return [...canonical(value)].map(char => TONES[char] ?? char).join('')
}
export function searchPoints<T extends { title: string; syllable?: string }>(points: T[], search: string): T[] {
  const query = normalizePinyin(search)
  if (!query) return points
  const exact = canonical(search)
  return points.filter(p => normalizePinyin(p.syllable || p.title).includes(query))
    .sort((a, b) => Number(canonical(b.syllable || b.title) === exact) - Number(canonical(a.syllable || a.title) === exact))
}

export function normalizeModules(value: unknown): (Omit<MatrixModule, 'points'> & { points: PinyinPoint[] })[] {
  if (!Array.isArray(value)) return []
  return value.filter(record).filter(m => typeof m.code === 'string').map(m => {
    const points: PinyinPoint[] = []
    for (const raw of Array.isArray(m.points) ? m.points : []) {
      if (!record(raw) || typeof raw.id !== 'number' || !Number.isFinite(raw.id) || typeof raw.title !== 'string') continue
      const kind = raw.kind ?? (m.code === 'syllables' ? 'syllable' : 'letter')
      if (kind !== 'letter' && kind !== 'syllable') continue
      points.push({ ...raw, id: raw.id, title: raw.title, kind, status: 'not_started',
        accuracy: finite(raw.accuracy), attempts: finite(raw.attempts), due_at: typeof raw.due_at === 'string' ? raw.due_at : null,
        initial: typeof raw.initial === 'string' ? raw.initial : undefined,
        final: typeof raw.final === 'string' ? raw.final : undefined,
        tone: typeof raw.tone === 'number' ? raw.tone : undefined,
        syllable: typeof raw.syllable === 'string' ? raw.syllable : undefined,
        last_at: typeof raw.last_at === 'string' ? raw.last_at : null,
        skills: Array.isArray(raw.skills) ? raw.skills.filter(record).filter(s => typeof s.code === 'string') as unknown as MatrixSkill[] : [],
      })
    }
    return { code: String(m.code), name: typeof m.name === 'string' ? m.name : String(m.code), total: points.length,
      mastered: points.filter(p => pinyinState(p).complete).length, points }
  })
}

export function pinyinGroups(value: unknown, search = '') {
  const modules = normalizeModules(value)
  const letters: PinyinGroup[] = modules.map(m => ({ ...m, points: m.points.filter(p => p.kind === 'letter') }))
    .filter(m => m.points.length > 0).map(m => ({ ...m, visiblePoints: searchPoints(m.points, search) }))
  const groups = new Map<string, PinyinPoint[]>()
  for (const point of modules.flatMap(m => m.points).filter(p => p.kind === 'syllable')) {
    const final = point.final ? normalizePinyin(point.final) : '未分组'
    groups.set(final, [...(groups.get(final) ?? []), point])
  }
  const syllables: PinyinGroup[] = [...groups].map(([code, points]) => {
    points.sort((a, b) => normalizePinyin(a.initial ?? '').localeCompare(normalizePinyin(b.initial ?? ''), 'en') || (a.tone ?? 0) - (b.tone ?? 0) || a.id - b.id)
    return { code, name: code === '未分组' ? code : `${code} 韵母`, points, visiblePoints: searchPoints(points, search) }
  })
  return { letters, syllables }
}

export function formatPinyinDate(value: string, withTime = false) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '日期未记录'
  return new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', ...(withTime ? { hour: '2-digit', minute: '2-digit' } as const : {}) }).format(date)
}
