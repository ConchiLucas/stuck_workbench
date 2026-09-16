export function healthLabel(health: string): string {
  if (health === 'weak') return '需巩固'
  if (health === 'watch') return '观察'
  return '稳定'
}

export function statusLabel(status: string): string {
  switch (status) {
    case 'shaky':
      return '需巩固'
    case 'review_due':
      return '该复习'
    case 'learning':
      return '学习中'
    case 'mastered':
      return '已掌握'
    case 'not_started':
      return '未开始'
    default:
      return status
  }
}

export function pct(n: number): string {
  if (!Number.isFinite(n)) return '—'
  return `${Math.round(n * 100)}%`
}

export function patternKindLabel(kind: string): string {
  switch (kind) {
    case 'skill_imbalance':
      return '技能不平衡'
    case 'repeated_wrong':
      return '反复选错'
    case 'slow_wrong':
      return '超时答错'
    default:
      return kind
  }
}
