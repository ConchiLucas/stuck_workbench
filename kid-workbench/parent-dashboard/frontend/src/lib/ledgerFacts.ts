import type { CalendarDay } from '../api/types'

export function ledgerCalendarDays(calDays: CalendarDay[] | undefined): CalendarDay[] {
  return [...(calDays ?? [])]
}

export function subjectPercents(
  ordered: { code: string; name: string }[],
  backend: { code: string; total: number; counts: { mastered: number; review_due: number; learning?: number; shaky?: number } }[],
): { code: string; name: string; pct: number; learningPct: number }[] {
  return ordered.map((item) => {
    const found = backend.find((b) => b.code === item.code)
    const masteredLike = found ? found.counts.mastered + found.counts.review_due : 0
    const learningLike = found ? (found.counts.learning ?? 0) + (found.counts.shaky ?? 0) : 0
    const total = found?.total ?? 0
    const pct = found && total > 0 && masteredLike > 0
      ? Math.round((masteredLike / total) * 100)
      : 0
    const learningPct = found && total > 0 && learningLike > 0
      ? Math.round((learningLike / total) * 100)
      : 0
    return { code: item.code, name: item.name, pct, learningPct }
  })
}

export function overallMasteredPct(mastered: number | undefined, total: number | undefined): number {
  if (!mastered || mastered <= 0) return 0
  return Math.round((mastered / Math.max(1, total ?? 0)) * 100)
}

export type DailyQuizRow = {
  code: string
  name: string
  hasSparkle?: boolean
  correctPct: number
  wrongPct: number
  correctShare: number
  wrongShare: number
  unansweredShare: number
  done: number
  target: number
  correct: number
  wrong: number
}

type PlanFact = {
  plan_date: string
  done_count: number
  target_count: number
  correct_count?: number
  status: string
  subjects?: { code: string; name: string; count?: number; done_count?: number; correct_count?: number }[]
}

// 当日条形按题目最终结果：答对/答错/未作答分母是该学科题量。
// 正确率 = 答对 / 已作答；未作答不计错误。同一题当场重试不重复计入。
export function dailyQuizFromPlans(plans: PlanFact[] | undefined, selected: string): DailyQuizRow[] {
  const selectedDayPlans = (plans ?? []).filter((p) => p.plan_date === selected)
  if (!selectedDayPlans.length) return []
  const byCode = new Map<string, { code: string; name: string; target: number; done: number; correct: number; hasSparkle: boolean }>()
  for (const p of selectedDayPlans) {
    const subjects = p.subjects ?? []
    if (!subjects.length) continue
    for (const s of subjects) {
      const single = subjects.length === 1
      const target = s.count ?? (single ? p.target_count : 0)
      const done = s.done_count ?? (single ? p.done_count : 0)
      const correct = s.correct_count ?? (single ? p.correct_count ?? 0 : 0)
      const cur = byCode.get(s.code) ?? { code: s.code, name: s.name, target: 0, done: 0, correct: 0, hasSparkle: false }
      cur.target += target
      cur.done += done
      cur.correct += correct
      cur.hasSparkle = cur.hasSparkle || p.status === 'done'
      byCode.set(s.code, cur)
    }
  }
  return [...byCode.values()].map((s) => {
    const wrong = Math.max(0, s.done - s.correct)
    const unanswered = Math.max(0, s.target - s.done)
    const denom = Math.max(1, s.target)
    return {
      code: s.code,
      name: s.name,
      hasSparkle: s.hasSparkle,
      correctPct: s.done > 0 ? Math.round((s.correct / s.done) * 100) : 0,
      wrongPct: s.done > 0 ? Math.round((wrong / s.done) * 100) : 0,
      correctShare: Math.round((s.correct / denom) * 100),
      wrongShare: Math.round((wrong / denom) * 100),
      unansweredShare: Math.round((unanswered / denom) * 100),
      done: s.done,
      target: s.target,
      correct: s.correct,
      wrong,
    }
  })
}

export function questionTypeBars(
  total: number,
  mastered: number,
  attempted: number,
): { masteredPct: number; attemptedPct: number } {
  if (total <= 0) return { masteredPct: 0, attemptedPct: 0 }
  return {
    masteredPct: Math.round((Math.max(0, mastered) / total) * 100),
    attemptedPct: Math.round((Math.max(0, attempted) / total) * 100),
  }
}
