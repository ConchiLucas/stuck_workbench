import type { MatrixPoint, MatrixSkill } from '../api/types'

export const SCIENCE_TYPES = [
  { code: 'choice', name: '选择题', short: '选' },
  { code: 'match', name: '连线题', short: '连' },
  { code: 'sequence', name: '排序题', short: '排' },
  { code: 'label', name: '结构标注题', short: '标' },
] as const

export type ScienceCounts = { mastered: number; answered: number; unattempted: number; total: number }
export type SciencePoint = MatrixPoint & { module_code: string }

function finite(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
}

export function scienceState(point: { status?: string; skills?: MatrixSkill[] | null; attempts?: number }) {
  const skills = SCIENCE_TYPES.map((type) => {
    const raw = point.skills?.find((item) => item.code === type.code)
    const attempts = finite(raw?.attempts)
    const lit = raw?.status === 'mastered' || raw?.status === 'review_due'
    const started = attempts > 0 || Boolean(raw?.status && raw.status !== 'not_started')
    if (!raw || !started) return null
    const detail = raw?.status === 'review_due' ? '待复习' : lit ? '已掌握' : raw?.status === 'shaky' ? '待巩固' : attempts ? '练习中' : '未练'
    return { ...type, status: raw?.status ?? 'not_started', attempts, accuracy: finite(raw?.accuracy), lit, detail }
  }).filter((skill): skill is NonNullable<typeof skill> => skill != null)
  const complete = point.status === 'mastered' || point.status === 'review_due'
  const started = finite(point.attempts) > 0 || skills.some((skill) => skill.attempts > 0 || skill.status !== 'not_started')
  return { skills, complete, started, label: complete ? '已掌握' : started ? '尚未掌握' : '未练' }
}

export function scienceSummary(points: { status?: string; skills?: MatrixSkill[] | null; attempts?: number }[]) {
  const byKind = SCIENCE_TYPES.map(({ code }) => {
    const applicable = points.flatMap((point) => scienceState(point).skills.filter((skill) => skill.code === code))
    return [code, {
      mastered: applicable.filter((skill) => skill.lit).length,
      answered: applicable.filter((skill) => !skill.lit && skill.attempts > 0).length,
      unattempted: applicable.filter((skill) => !skill.lit && skill.attempts === 0).length,
      total: applicable.length,
    }]
  })
  return Object.fromEntries(byKind) as Record<string, ScienceCounts>
}
