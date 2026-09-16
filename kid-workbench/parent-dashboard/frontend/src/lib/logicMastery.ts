import type { MatrixPoint } from '../api/types'

export const LOGIC_TYPES = [
  { code: 'pattern', name: '找规律', short: '律' },
  { code: 'classify', name: '分类', short: '类' },
  { code: 'order', name: '排序', short: '序' },
  { code: 'shape_reason', name: '图形推理', short: '形' },
  { code: 'diff', name: '找不同', short: '异' },
  { code: 'compare', name: '比较', short: '比' },
] as const

export type LogicCounts = { mastered: number; answered: number; unattempted: number; total: number }
export type LogicPoint = MatrixPoint & { module_code: string }

function finite(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
}

export function logicKindOf(point: { kind?: string; module_code?: string }) {
  const kind = point.kind || point.module_code || ''
  return LOGIC_TYPES.some((type) => type.code === kind) ? kind : ''
}

export function logicState(point: { status?: string; attempts?: number }) {
  const complete = point.status === 'mastered' || point.status === 'review_due'
  const started = finite(point.attempts) > 0 || Boolean(point.status && point.status !== 'not_started')
  return { complete, started, label: complete ? '已掌握' : started ? '尚未掌握' : '未练' }
}

export function logicSummary(points: { kind?: string; module_code?: string; status?: string; attempts?: number }[]) {
  const byKind = LOGIC_TYPES.map(({ code }) => {
    const applicable = points.filter((point) => logicKindOf(point) === code)
    return [code, {
      mastered: applicable.filter((point) => logicState(point).complete).length,
      answered: applicable.filter((point) => !logicState(point).complete && finite(point.attempts) > 0).length,
      unattempted: applicable.filter((point) => !logicState(point).complete && finite(point.attempts) === 0).length,
      total: applicable.length,
    }]
  })
  return Object.fromEntries(byKind) as Record<string, LogicCounts>
}
