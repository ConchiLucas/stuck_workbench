import type { PlanSummary } from '../../api/plans'

export interface DaySubjectRow {
  code: string
  name: string
  icon: string
  total: number
  done: number
  complete: boolean
}

export function subjectRowsForDay(plans: PlanSummary[]): DaySubjectRow[] {
  const active = plans.filter((p) => p.done_count > 0 || p.status === 'doing' || p.status === 'done')
  const source = active.length > 0 ? active : plans
  const map = new Map<string, DaySubjectRow>()
  for (const p of source) {
    const items = p.subjects ?? []
    const itemTotal = items.reduce((n, s) => n + s.count, 0) || p.target_count || 1
    for (const s of items) {
      const cur = map.get(s.code) ?? {
        code: s.code,
        name: s.name,
        icon: s.icon,
        total: 0,
        done: 0,
        complete: true,
      }
      cur.total += s.count
      cur.done += itemTotal > 0 ? (p.done_count * s.count) / itemTotal : 0
      cur.complete = cur.complete && p.status === 'done'
      map.set(s.code, cur)
    }
  }
  return [...map.values()].map((s) => ({
    ...s,
    done: Math.min(s.total, Math.round(s.done)),
    complete: s.complete && s.done >= s.total && s.total > 0,
  }))
}

export function dayTone(plans: PlanSummary[]): { label: string; cheer: string } {
  if (plans.length === 0) return { label: '尚未开始', cheer: '这一天还没有练习记录。' }
  if (plans.every((p) => p.status === 'done')) {
    return { label: '完成良好', cheer: '今天也很棒！继续加油！' }
  }
  if (plans.some((p) => p.status === 'doing' || p.done_count > 0)) {
    return { label: '进行中', cheer: '按计划慢慢推进就好。' }
  }
  return { label: '待开始', cheer: '打开孩子端，开始今天的练习。' }
}

export function insightLine(params: {
  weekNew: { name: string; week_new: number }[]
  weekMastered: number
}): string {
  const hot = params.weekNew.filter((s) => s.week_new > 0).sort((a, b) => b.week_new - a.week_new)
  if (hot.length === 0) {
    return params.weekMastered > 0
      ? `最近 7 天新掌握了 ${params.weekMastered} 个知识点，继续保持这节节奏。`
      : '最近几天还没有新掌握，打开孩子端练一小份就有记录。'
  }
  const names = hot.slice(0, 2).map((s) => s.name).join('和')
  return `最近 7 天，${names}学得较多，已有可见进步。继续保持这节节奏吧！`
}
