export function shanghaiToday() {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date())
}
export function shiftMonth(month: string, offset: number) {
  const [year, m] = month.split('-').map(Number)
  return new Date(Date.UTC(year, m - 1 + offset, 1)).toISOString().slice(0, 7)
}
export function monthDates(month: string) {
  const [year, m] = month.split('-').map(Number)
  return Array.from({ length: new Date(Date.UTC(year, m, 0)).getUTCDate() }, (_, i) => `${month}-${String(i + 1).padStart(2, '0')}`)
}
export function shiftDay(date: string, offset: number) {
  const d = new Date(`${date}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() + offset)
  return d.toISOString().slice(0, 10)
}
export const dateCaption = (date: string) => `${Number(date.slice(5, 7))}月${Number(date.slice(8))}日`
