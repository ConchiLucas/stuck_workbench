import type { MatrixPoint, MatrixSkill } from '../api/types'

export const PHRASE_TYPES = [
  { code: 'listen_zh', name: '听一听', short: '听' },
  { code: 'listen_en', name: '选句子', short: '选' },
  { code: 'scene', name: '什么时候说', short: '场' },
  { code: 'reply', name: '问与答', short: '答' },
] as const

export const PHRASE_CORE = ['listen_zh', 'listen_en', 'scene'] as const

export type PhraseCounts = { mastered: number; answered: number; unattempted: number; total: number }
export type PhrasePoint = MatrixPoint & { module_code: string }

function finite(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
}

export function phraseState(point: { skills?: MatrixSkill[] | null }) {
  const skills = PHRASE_TYPES.map((type) => {
    const raw = point.skills?.find((item) => item.code === type.code)
    const lit = raw?.status === 'mastered' || raw?.status === 'review_due'
    const attempts = finite(raw?.attempts)
    const detail = raw?.status === 'review_due' ? '待复习' : lit ? '已掌握' : raw?.status === 'shaky' ? '待巩固' : attempts ? '练习中' : '未练'
    return { ...type, status: raw?.status ?? 'not_started', attempts, accuracy: finite(raw?.accuracy), lit, detail }
  })
  const core = skills.filter((skill) => (PHRASE_CORE as readonly string[]).includes(skill.code))
  const complete = core.every((skill) => skill.lit)
  const started = skills.some((skill) => skill.attempts > 0 || skill.status !== 'not_started')
  return { skills, complete, started, label: complete ? '听、选、场景均已掌握' : started ? '尚未完全掌握' : '未练' }
}

export function phraseSummary(points: { skills?: MatrixSkill[] | null }[]) {
  const skills = points.flatMap((point) => phraseState(point).skills)
  return Object.fromEntries(PHRASE_TYPES.map(({ code }) => {
    const applicable = skills.filter((skill) => skill.code === code)
    return [code, {
      mastered: applicable.filter((skill) => skill.lit).length,
      answered: applicable.filter((skill) => !skill.lit && skill.attempts > 0).length,
      unattempted: applicable.filter((skill) => !skill.lit && skill.attempts === 0).length,
      total: applicable.length,
    }]
  })) as Record<string, PhraseCounts>
}
