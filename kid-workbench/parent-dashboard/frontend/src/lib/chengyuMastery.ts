import type { MatrixPoint, MatrixSkill } from '../api/types'

export const CHENGYU_TYPES = [
  { code: 'meaning', name: '听释义', short: '听' },
  { code: 'pick', name: '选成语', short: '选' },
  { code: 'pinyin', name: '看拼音', short: '拼' },
  { code: 'example', name: '看句子', short: '句' },
] as const

export type ChengyuCounts = { mastered: number; answered: number; unattempted: number; total: number }
export type ChengyuPoint = MatrixPoint & { module_code: string }

function finite(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
}

export function chengyuState(point: { skills?: MatrixSkill[] | null }) {
  const skills = CHENGYU_TYPES.map((type) => {
    const raw = point.skills?.find((item) => item.code === type.code)
    const lit = raw?.status === 'mastered' || raw?.status === 'review_due'
    const attempts = finite(raw?.attempts)
    const detail = raw?.status === 'review_due' ? '待复习' : lit ? '已掌握' : raw?.status === 'shaky' ? '待巩固' : attempts ? '练习中' : '未练'
    return { ...type, status: raw?.status ?? 'not_started', attempts, accuracy: finite(raw?.accuracy), lit, detail }
  })
  const complete = skills.every((skill) => skill.lit)
  const started = skills.some((skill) => skill.attempts > 0 || skill.status !== 'not_started')
  return { skills, complete, started, label: complete ? '四项均已掌握' : started ? '尚未完全掌握' : '未练' }
}

export function chengyuSummary(points: { skills?: MatrixSkill[] | null }[]) {
  const skills = points.flatMap((point) => chengyuState(point).skills)
  return Object.fromEntries(CHENGYU_TYPES.map(({ code }) => {
    const applicable = skills.filter((skill) => skill.code === code)
    return [code, {
      mastered: applicable.filter((skill) => skill.lit).length,
      answered: applicable.filter((skill) => !skill.lit && skill.attempts > 0).length,
      unattempted: applicable.filter((skill) => !skill.lit && skill.attempts === 0).length,
      total: applicable.length,
    }]
  })) as Record<string, ChengyuCounts>
}
