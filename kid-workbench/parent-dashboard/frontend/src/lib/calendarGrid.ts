import type { CalendarDay } from '../api/types'
import { localDate, startOfMonday } from './date'

export const WEEKDAYS = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

export type HeatLevel = 0 | 1 | 2 | 3 | 4

export interface CalendarCell {
  date: string
  mastered: number
  attempts: number
  level: HeatLevel
  inRange: boolean
}

export function heatLevel(mastered: number): HeatLevel {
  if (mastered <= 0) return 0
  if (mastered <= 3) return 1
  if (mastered <= 6) return 2
  if (mastered <= 9) return 3
  return 4
}

/** 近 `weeks` 周、周一到周日的格子。 */
export function buildWeekGrid(days: CalendarDay[], weeks = 12): CalendarCell[][] {
  const byDate = new Map(days.map((d) => [d.date, d]))
  const monday = startOfMonday()
  monday.setDate(monday.getDate() - (weeks - 1) * 7)
  const columns: CalendarCell[][] = []
  for (let w = 0; w < weeks; w++) {
    const col: CalendarCell[] = []
    for (let i = 0; i < 7; i++) {
      const date = localDate(new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + w * 7 + i))
      const hit = byDate.get(date)
      col.push({
        date,
        mastered: hit?.mastered ?? 0,
        attempts: hit?.attempts ?? 0,
        level: heatLevel(hit?.mastered ?? 0),
        inRange: date <= localDate(),
      })
    }
    columns.push(col)
  }
  return columns
}

export function gridStartDate(weeks = 12): string {
  const monday = startOfMonday()
  monday.setDate(monday.getDate() - (weeks - 1) * 7)
  return localDate(monday)
}

export function monthLabels(columns: CalendarCell[][]): { index: number; label: string }[] {
  const out: { index: number; label: string }[] = []
  let last = -1
  columns.forEach((col, i) => {
    const month = Number(col[0]?.date.slice(5, 7))
    if (month !== last) {
      out.push({ index: i, label: `${month}月` })
      last = month
    }
  })
  return out
}
