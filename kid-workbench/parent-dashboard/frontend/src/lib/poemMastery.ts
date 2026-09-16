import type { MatrixPoint, MatrixSkill } from '../api/types'

export const POEM_TYPES = [
  { code: 'title', name: '选诗名', short: '名' },
  { code: 'fill', name: '补字', short: '字' },
  { code: 'couplet', name: '选下一句', short: '句' },
  { code: 'recite', name: '排顺序', short: '序' },
] as const

export type PoemCounts = { mastered: number; answered: number; unattempted: number; total: number }
export type PoemPoint = MatrixPoint & { module_code: string }

function finite(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
}

export function poemState(point: { status?: string; skills?: MatrixSkill[] | null; attempts?: number }) {
  const skills = POEM_TYPES.map((type) => {
    const raw = point.skills?.find((item) => item.code === type.code)
    const attempts = finite(raw?.attempts)
    const lit = raw?.status === 'mastered' || raw?.status === 'review_due'
    const detail = raw?.status === 'review_due' ? '待复习' : lit ? '已掌握' : raw?.status === 'shaky' ? '待巩固' : attempts ? '练习中' : '未练'
    return { ...type, status: raw?.status ?? 'not_started', attempts, accuracy: finite(raw?.accuracy), lit, detail }
  })
  const complete = point.status === 'mastered' || point.status === 'review_due'
  const started = finite(point.attempts) > 0 || skills.some((skill) => skill.attempts > 0 || skill.status !== 'not_started')
  return { skills, complete, started, label: complete ? '已掌握' : started ? '尚未掌握' : '未练' }
}

export function poemSummary(points: { status?: string; skills?: MatrixSkill[] | null; attempts?: number }[]) {
  const byKind = POEM_TYPES.map(({ code }) => {
    const applicable = points.flatMap((point) => poemState(point).skills.filter((skill) => skill.code === code))
    return [code, {
      mastered: applicable.filter((skill) => skill.lit).length,
      answered: applicable.filter((skill) => !skill.lit && skill.attempts > 0).length,
      unattempted: applicable.filter((skill) => !skill.lit && skill.attempts === 0).length,
      total: applicable.length,
    }]
  })
  return Object.fromEntries(byKind) as Record<string, PoemCounts>
}
