import { useState } from 'react'
import clsx from 'clsx'
import { WEEKDAYS, type CalendarCell } from '../../lib/calendarGrid'

const LEVEL_CLASS: Record<number, string> = {
  0: 'bg-[#24132b]',
  1: 'bg-[#5e2246]',
  2: 'bg-[#aa3b72]',
  3: 'bg-[#f04e8d]',
  4: 'bg-[#ff7bb0]',
}

export function LearningCalendar({
  columns,
  monthLabels,
  selected,
  onSelect,
  today,
}: {
  columns: CalendarCell[][]
  monthLabels: { index: number; label: string }[]
  selected: string
  onSelect: (date: string) => void
  today: string
}) {
  const [tab, setTab] = useState<'attempts' | 'mastered'>('mastered')
  const marks = new Map(monthLabels.map((m) => [m.index, m.label]))

  const selectedMonth = selected.slice(5, 7).replace(/^0/, '')
  const selectedDay = selected.slice(8).replace(/^0/, '')

  return (
    <section className="dash-card flex flex-col justify-between p-5">
      {/* Header */}
      <header className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="flex items-center gap-2 text-sm font-bold text-white">
            <span className="grid h-6 w-6 place-items-center rounded-lg bg-pink-500/20 text-xs text-pink-400">
              🗓️
            </span>
            学习进度日历
          </h2>

          {/* Mode Switch Tabs */}
          <div className="flex items-center gap-1 rounded-full border border-white/10 bg-white/5 p-0.5 text-xs">
            <button
              type="button"
              onClick={() => setTab('attempts')}
              className={clsx(
                'rounded-full px-3 py-1 font-medium transition-all',
                tab === 'attempts'
                  ? 'bg-gradient-to-r from-pink-500 to-rose-500 text-white shadow-[0_0_12px_rgba(244,114,166,0.5)]'
                  : 'text-white/60 hover:text-white',
              )}
            >
              作答情况
            </button>
            <button
              type="button"
              onClick={() => setTab('mastered')}
              className={clsx(
                'rounded-full px-3 py-1 font-medium transition-all',
                tab === 'mastered'
                  ? 'bg-gradient-to-r from-pink-500 to-rose-500 text-white shadow-[0_0_12px_rgba(244,114,166,0.5)]'
                  : 'text-white/60 hover:text-white',
              )}
            >
              新增完全掌握
            </button>
          </div>
        </div>

        {/* Date Range Controls */}
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            className="grid h-7 w-7 place-items-center rounded-lg border border-white/10 bg-white/5 text-xs text-white/60 hover:bg-white/10 hover:text-white"
          >
            ❮
          </button>
          <button
            type="button"
            className="flex h-7 items-center gap-1.5 rounded-lg border border-white/10 bg-white/5 px-3 text-xs text-white/80 hover:bg-white/10"
          >
            <span>近三个月</span>
          </button>
          <button
            type="button"
            className="grid h-7 w-7 place-items-center rounded-lg border border-white/10 bg-white/5 text-xs text-white/60 hover:bg-white/10 hover:text-white"
          >
            ❯
          </button>
        </div>
      </header>

      {/* Month Labels */}
      <div className="flex">
        <div className="w-10 shrink-0" />
        <div className="mb-1 flex min-w-0 flex-1 text-[10px] text-white/40">
          {columns.map((col, i) => (
            <span key={col[0]?.date} className="min-w-0 flex-1 truncate">{marks.get(i) ?? ''}</span>
          ))}
        </div>
      </div>

      {/* Grid */}
      <div className="flex">
        <div className="flex w-10 shrink-0 flex-col gap-[3px] text-[10px] leading-[14px] text-white/40">
          {WEEKDAYS.map((d) => (
            <span key={d} className="h-[14px]">{d}</span>
          ))}
        </div>
        <div className="flex min-w-0 flex-1 gap-[3px]">
          {columns.map((col) => (
            <div key={col[0]?.date} className="flex min-w-0 flex-1 flex-col gap-[3px]">
              {col.map((cell) => {
                const cellVal = tab === 'mastered' ? cell.mastered : cell.attempts
                const cellLv = cell.level
                const isSelected = selected === cell.date

                return (
                  <button
                    key={cell.date}
                    type="button"
                    disabled={!cell.inRange}
                    title={`${cell.date} · ${tab === 'mastered' ? `新掌握 ${cell.mastered}` : `作答 ${cell.attempts} 次`}`}
                    onClick={() => onSelect(cell.date)}
                    className={clsx(
                      'h-[14px] w-full rounded-[3px] transition-all',
                      LEVEL_CLASS[cellLv],
                      cellLv === 0 && cellVal > 0 && 'bg-[#481c39]',
                      isSelected && 'ring-2 ring-pink-400 ring-offset-2 ring-offset-[#13081a] z-10 scale-110 shadow-[0_0_8px_rgba(244,114,166,0.6)]',
                      !cell.inRange && 'opacity-25',
                    )}
                  />
                )
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Footer */}
      <footer className="mt-4 flex flex-wrap items-center justify-between gap-3 text-xs text-white/50 pt-2 border-t border-white/5">
        <div className="flex items-center gap-1.5 text-[11px]">
          <span>新增完全掌握</span>
          <span className="text-white/40 ml-1">少</span>
          {[0, 1, 2, 3, 4].map((lv) => (
            <i key={lv} className={clsx('inline-block h-2.5 w-2.5 rounded-sm', LEVEL_CLASS[lv])} />
          ))}
          <span className="text-white/40">多</span>
        </div>

        <div className="flex items-center gap-3">
          <span className="text-white/40 text-[11px] hidden sm:inline">点击日期，联动右侧答题结果</span>
          <span className="rounded-full border border-white/10 bg-white/5 px-3 py-0.5 text-xs text-white font-medium">
            {selectedMonth}月{selectedDay}日
          </span>
          <button
            type="button"
            onClick={() => onSelect(today)}
            className="rounded-full bg-gradient-to-r from-pink-500 to-rose-500 px-3 py-0.5 text-xs font-medium text-white shadow-[0_0_10px_rgba(244,114,166,0.4)] hover:brightness-110 transition-all cursor-pointer"
          >
            今天
          </button>
        </div>
      </footer>
    </section>
  )
}


