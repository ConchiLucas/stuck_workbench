/**
 * 本地时区的 YYYY-MM-DD。
 *
 * 不能用 toISOString().slice(0,10)：那是 UTC 日期，东八区晚上 8 点之后
 * 会算成前一天，和后端按本地日期存的 plan_date 对不上。
 */
export function localDate(d: Date = new Date()): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

export function parseLocalDate(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(y, (m ?? 1) - 1, d ?? 1)
}

export function addDays(s: string, n: number): string {
  const d = parseLocalDate(s)
  d.setDate(d.getDate() + n)
  return localDate(d)
}

/** 周一为一周的第一天。 */
export function startOfMonday(d: Date = new Date()): Date {
  const x = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const day = x.getDay()
  x.setDate(x.getDate() + (day === 0 ? -6 : 1 - day))
  return x
}

export function formatMonthDay(s: string): string {
  const [, m, d] = s.split('-')
  return `${Number(m)}月${Number(d)}日`
}
