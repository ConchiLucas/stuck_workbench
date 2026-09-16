import type { MatrixPoint, MatrixSkill } from '../api/types'

export const MATH_TYPES = [
  { code: 'calc', short: '算', name: '算式计算' },
  { code: 'story', short: '用', name: '情境应用' },
  { code: 'find', short: '找', name: '听音找图形' },
  { code: 'name', short: '认', name: '看图认名称' },
] as const
export type MathSkillCode = typeof MATH_TYPES[number]['code']
export type MathPoint = MatrixPoint & { module_code: string }
export type MathCounts = { mastered: number; answered: number; unattempted: number; total: number }
export type MathGroup = { code: string; name: string; points: MathPoint[]; visiblePoints: MathPoint[] }
const finite = (value: unknown) => typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
const record = (value: unknown): value is Record<string, unknown> => !!value && typeof value === 'object'

export function mathState(point: { module_code?: string; skills?: MatrixSkill[] | null }) {
  const types = point.module_code === 'shape' ? MATH_TYPES.slice(2) : ['add10', 'sub10'].includes(point.module_code ?? '') ? MATH_TYPES.slice(0, 2) : []
  const skills = types.map(type => {
    const raw = Array.isArray(point.skills) ? point.skills.find(s => s?.code === type.code) : undefined
    const lit = raw?.status === 'mastered' || raw?.status === 'review_due'
    const attempts = finite(raw?.attempts)
    return { ...type, status: raw?.status ?? 'not_started', attempts, accuracy: Math.min(1, finite(raw?.accuracy)), lit,
      detail: lit ? '已掌握' : attempts > 0 ? '已作答未掌握' : '未作答' }
  })
  const mastered = skills.filter(s => s.lit).length
  const complete = skills.length > 0 && mastered === skills.length
  return { skills, mastered, complete, label: complete ? '完全掌握' : mastered ? '部分题型掌握' : '尚未掌握' }
}

export function mathSummary(points: { module_code?: string; skills?: MatrixSkill[] | null }[]) {
  const skills = points.flatMap(p => mathState(p).skills)
  return Object.fromEntries(MATH_TYPES.map(({ code }) => {
    const applicable = skills.filter(s => s.code === code)
    return [code, { mastered: applicable.filter(s => s.lit).length, answered: applicable.filter(s => !s.lit && s.attempts > 0).length,
      unattempted: applicable.filter(s => !s.lit && s.attempts === 0).length, total: applicable.length }]
  })) as Record<MathSkillCode, MathCounts>
}

const normalize = (value: string) => value.trim().replace(/\s/g, '').replace(/[−－]/g, '-').replace(/＋/g, '+')
function range(point: MathPoint) {
  const match = normalize(point.title).match(/^(\d+)([+-])(\d+)$/)
  if (!match || (match[2] === '+') !== (point.module_code === 'add10')) return 0
  const a = Number(match[1]), b = Number(match[3])
  const max = point.module_code === 'add10' ? a + b : a
  if (point.module_code === 'sub10' && b > a) return 0
  return max <= 5 ? 5 : max <= 10 ? 10 : max <= 20 ? 20 : 0
}

export function mathGroups(value: unknown, search = ''): MathGroup[] {
  if (!Array.isArray(value)) return []
  const groups: MathGroup[] = []
  for (const module of value.filter(record)) {
    if (!['add10', 'sub10', 'shape'].includes(String(module.code))) continue
    const points = (Array.isArray(module.points) ? module.points : []).filter(record)
      .filter(p => typeof p.id === 'number' && Number.isFinite(p.id) && typeof p.title === 'string')
      .map(p => ({ ...p, module_code: String(module.code), id: Number(p.id), title: String(p.title), status: 'not_started',
        attempts: finite(p.attempts), accuracy: finite(p.accuracy), due_at: typeof p.due_at === 'string' ? p.due_at : null,
        skills: Array.isArray(p.skills) ? p.skills.filter(record).filter(s => typeof s.code === 'string') as unknown as MatrixSkill[] : [],
      }) as MathPoint)
    const name = module.code === 'add10' ? '加法' : module.code === 'sub10' ? '减法' : '认识图形'
    const add = (code: string, title: string, items: MathPoint[]) => groups.push({ code, name: title, points: items,
      visiblePoints: items.filter(p => normalize(p.title).includes(normalize(search))) })
    if (module.code === 'shape') add('shape', name, points)
    else {
      for (const max of [5, 10, 20]) add(`${module.code}-${max}`, `${max}以内${name}`, points.filter(p => range(p) === max))
      const rest = points.filter(p => range(p) === 0)
      if (rest.length) add(`${module.code}-other`, `${name} · 其他算式`, rest)
    }
  }
  return groups
}
