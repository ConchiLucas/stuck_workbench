const labels: Record<string, string> = { not_started: '还没学', learning: '学习中', shaky: '再练练', mastered: '已掌握', review_due: '该复习' }
export function StatusBadge({ status }: { status: string }) { return <span className={`status status-${status}`}>{labels[status] ?? '还没学'}</span> }
