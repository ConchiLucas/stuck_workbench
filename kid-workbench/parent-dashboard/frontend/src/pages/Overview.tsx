import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCalendar, useOverview, useSubjects, visibleSubjects } from '../api/dashboard'
import { usePlanHistory } from '../api/plans'
import { useChildStore } from '../store/childStore'
import { localDate } from '../lib/date'
import { buildWeekGrid, gridStartDate, monthLabels as monthMarks } from '../lib/calendarGrid'
import { dailyQuizFromPlans, ledgerCalendarDays, overallMasteredPct, questionTypeBars, subjectPercents } from '../lib/ledgerFacts'
import { LearningCalendar } from '../components/overview/LearningCalendar'
import { ORDERED_SUBJECTS, SubjectBadge } from '../components/overview/SubjectIcons'
import type { QuestionTypeStat } from '../api/types'

export function Overview() {
  const childId = useChildStore((s) => s.childId)
  const nav = useNavigate()
  const today = localDate()
  const [picked, setPicked] = useState<string | null>(null)

  const { data: ov, isError: overviewFailed } = useOverview(childId)
  const { data: rawSubjects, isError: subjectsFailed } = useSubjects(childId)
  const { data: calDays, isError: calendarFailed } = useCalendar(childId, 4)
  const from = gridStartDate(12)
  const { data: plans, isError: plansFailed } = usePlanHistory(childId, '', { from, to: today })
  const backendSubjects = visibleSubjects(rawSubjects)

  const columns = useMemo(() => buildWeekGrid(ledgerCalendarDays(calDays), 12), [calDays])
  const months = useMemo(() => monthMarks(columns), [columns])
  const selected = picked ?? today

  const selectedMonth = selected.slice(5, 7).replace(/^0/, '')
  const selectedDay = selected.slice(8).replace(/^0/, '')

  const dailyResults = useMemo(() => dailyQuizFromPlans(plans, selected), [plans, selected])

  const subjectsList = useMemo(() => subjectPercents(ORDERED_SUBJECTS, backendSubjects), [backendSubjects])
  const questionTypeRows = useMemo(
    () => ORDERED_SUBJECTS.map((s) => {
      const found = backendSubjects.find((b) => b.code === s.code)
      return { code: s.code, name: s.name, types: (found?.question_types ?? []) as QuestionTypeStat[] }
    }),
    [backendSubjects],
  )

  const overallPct = overallMasteredPct(ov?.counts?.mastered, ov?.total_kp)
  const ledgerFailed = overviewFailed || subjectsFailed || calendarFailed || plansFailed

  return (
    <div className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-4 pb-1">
        <div className="flex items-center gap-3.5">
          <div className="grid h-12 w-12 shrink-0 place-items-center rounded-2xl border border-pink-500/30 bg-gradient-to-br from-pink-500/20 to-purple-600/20 text-2xl shadow-[0_0_16px_rgba(244,114,166,0.25)]">
            📖
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">学习进度总览</h1>
            <p className="mt-1 text-xs text-white/50">从答题结果，看见掌握的积累</p>
          </div>
        </div>

        <div className="flex items-center gap-6">
          <div className="flex items-center gap-3">
            <div className="relative grid h-11 w-11 place-items-center">
              <svg width={44} height={44} className="-rotate-90">
                <circle cx={22} cy={22} r={17} fill="none" stroke="#2a142c" strokeWidth={4.5} />
                <circle
                  cx={22}
                  cy={22}
                  r={17}
                  fill="none"
                  stroke="#ff4e88"
                  strokeWidth={4.5}
                  strokeLinecap="round"
                  strokeDasharray={2 * Math.PI * 17}
                  strokeDashoffset={2 * Math.PI * 17 * (1 - overallPct / 100)}
                  style={{ filter: 'drop-shadow(0 0 4px rgba(255,78,136,0.5))' }}
                />
              </svg>
              <span className="absolute text-xs text-pink-400">📊</span>
            </div>
            <div>
              <div className="text-sm font-bold text-white">总体掌握</div>
              <div className="text-[11px] text-white/50">全部题型掌握才计入</div>
            </div>
          </div>
        </div>
      </header>

      <section className="dash-card p-4">
        <header className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="grid h-6 w-6 place-items-center rounded-lg bg-pink-500/20 text-xs text-pink-400">
              📊
            </span>
            <h2 className="text-sm font-bold text-white">各科学学习进度</h2>
            <span className="text-xs text-white/40 ml-1">当前完全掌握</span>
          </div>
          <button
            type="button"
            className="flex items-center gap-1 text-xs text-white/50 transition-colors hover:text-white"
            onClick={() => nav(`/subjects/${subjectsList[0]?.code ?? 'literacy'}`)}
          >
            <span>查看全部学科</span>
            <span className="text-xs">➔</span>
          </button>
        </header>

        {ledgerFailed ? <p role="alert" className="mb-3 text-sm text-rose-300">进度账本加载失败，未使用合成数据代替。</p> : null}
        <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-3 md:grid-cols-5 xl:grid-cols-9">
          {subjectsList.map((s) => (
            <button
              key={s.code}
              type="button"
              onClick={() => nav(`/subjects/${s.code}`)}
              className="flex flex-col justify-between rounded-xl border border-white/[0.08] bg-[#1a1024]/90 p-3 text-left transition-all hover:border-pink-500/40 hover:bg-white/[0.04]"
            >
              <div className="flex items-center justify-between">
                <div className="flex min-w-0 items-center gap-2">
                  <SubjectBadge code={s.code} size="md" />
                  <span className="truncate text-xs font-semibold text-white/90">{s.name}</span>
                </div>
                <span className="text-xs text-white/30">❯</span>
              </div>

              <div className="mt-3 flex items-center justify-between gap-2">
                <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-[#27132e]">
                  <div
                    className="h-full rounded-full transition-all duration-500"
                    style={{
                      width: `${ledgerFailed ? 0 : s.pct}%`,
                      background: 'linear-gradient(90deg, #ff659c, #ff4081)',
                    }}
                  />
                </div>
                <span className="text-xs font-semibold tabular-nums text-white/80 font-mono">{ledgerFailed ? '—' : `${s.pct}%`}</span>
              </div>
            </button>
          ))}
        </div>
      </section>

      <section className="grid gap-4 lg:grid-cols-[minmax(0,1.3fr)_minmax(300px,0.95fr)]">
        <LearningCalendar
          columns={columns}
          monthLabels={months}
          selected={selected}
          onSelect={setPicked}
          today={today}
        />

        <section className="dash-card flex flex-col justify-between p-5">
          <div>
            <header className="mb-4 flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="grid h-6 w-6 place-items-center rounded-lg bg-pink-500/20 text-xs text-pink-400">
                  📊
                </span>
                <h2 className="text-sm font-bold text-white">
                  {selectedMonth}月{selectedDay}日 · 答题结果
                </h2>
              </div>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => nav('/tasks')}
                  className="flex items-center gap-1 text-xs text-white/50 transition-colors hover:text-white"
                >
                  <span>查看当日记录</span>
                  <span className="text-xs">➔</span>
                </button>
                <div className="flex items-center gap-2 text-[10px] text-white/50 border-l border-white/10 pl-2">
                  <span className="flex items-center gap-1">
                    <i className="h-2 w-2 rounded-full bg-[#ff4e88]" />答对
                  </span>
                  <span className="flex items-center gap-1">
                    <i className="h-2 w-2 rounded-full bg-[#6366f1]" />答错
                  </span>
                  <span className="flex items-center gap-1">
                    <i className="h-2 w-2 rounded-full bg-[#27132e] border border-white/20" />未作答
                  </span>
                </div>
              </div>
            </header>

            {ledgerFailed ? <p role="alert" className="text-sm text-rose-300">进度账本加载失败，未使用合成数据代替。</p> : null}
            {!ledgerFailed && dailyResults.length === 0 ? <p className="text-sm text-white/50">这一天还没有保存的练习记录。</p> : null}
            <ul className="space-y-3.5">
              {!ledgerFailed && dailyResults.map((row) => (
                <li key={row.code} className="flex items-center gap-3">
                  <div className="flex w-24 shrink-0 items-center gap-2">
                    <SubjectBadge code={row.code} size="sm" />
                    <span className="truncate text-xs font-semibold text-white/90">{row.name}</span>
                    {row.hasSparkle && <span className="text-xs text-amber-300">✨</span>}
                  </div>

                  <div
                    className="h-3.5 flex-1 overflow-hidden rounded-full bg-[#27132e] flex"
                    title={`答对 ${row.correct} / 已作答 ${row.done} / 计划 ${row.target}；未作答 ${Math.max(0, row.target - row.done)}`}
                  >
                    <div
                      style={{ width: `${row.correctShare}%` }}
                      className="h-full bg-gradient-to-r from-[#ff4e88] to-[#f472a6] shadow-[0_0_8px_rgba(255,78,136,0.3)] transition-all"
                    />
                    <div
                      style={{ width: `${row.wrongShare}%` }}
                      className="h-full bg-[#6366f1] transition-all"
                    />
                  </div>

                  <span className="w-10 shrink-0 text-right text-xs font-bold text-white/90 font-mono tabular-nums" title="正确率=答对题数/已作答题数">
                    {row.done > 0 ? `${row.correctPct}%` : '—'}
                  </span>
                </li>
              ))}
            </ul>
          </div>

          <footer className="mt-4 flex flex-wrap items-center justify-between gap-3 text-xs text-white/40 pt-3 border-t border-white/5">
            <div className="flex items-center gap-1 text-amber-300/90 text-xs">
              <span>✨</span>
              <span>当日有新增完全掌握</span>
            </div>
            <span>正确率=答对/已作答；完成率不是正确率；未作答不计错误；同题当场重试只计最终结果</span>
          </footer>
        </section>
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        <section className="dash-card flex flex-col justify-between p-5 rounded-2xl bg-[#170e26]/90 border border-white/[0.08] shadow-xl">
          <div>
            <header className="mb-4 flex h-7 items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="grid h-7 w-7 place-items-center rounded-lg bg-pink-500/15 border border-pink-500/30 text-pink-400">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
                    <circle cx="12" cy="12" r="10" />
                    <path d="M12 2a10 10 0 0 1 10 10h-10z" fill="currentColor" fillOpacity="0.2" />
                  </svg>
                </span>
                <h2 className="text-sm font-bold tracking-wide text-white">知识掌握总体概览</h2>
              </div>
              <div className="flex items-center gap-3 text-xs text-white/50">
                <span className="flex items-center gap-1.5">
                  <i className="h-2.5 w-2.5 rounded-full bg-[#ff4e88] shadow-[0_0_6px_rgba(255,78,136,0.6)]" />完全掌握
                </span>
                <span className="flex items-center gap-1.5">
                  <i className="h-2.5 w-2.5 rounded-full bg-[#9d4edd]" />已作答未掌握
                </span>
                <span className="flex items-center gap-1.5">
                  <i className="h-2.5 w-2.5 rounded-full bg-[#251433] border border-white/15" />尚无题型掌握
                </span>
              </div>
            </header>

            <div className="flex flex-col">
              {subjectsList.map((s) => (
                <div key={s.code} className="flex h-9 items-center gap-3">
                  <div className="flex w-20 shrink-0 items-center gap-2">
                    <SubjectBadge code={s.code} size="md" />
                    <span className="truncate text-xs font-semibold text-white/90">{s.name}</span>
                  </div>

                  <div className="h-2.5 flex-1 overflow-hidden rounded-full bg-[#251433] flex">
                    <div
                      style={{ width: `${ledgerFailed ? 0 : s.pct}%` }}
                      className="h-full bg-gradient-to-r from-[#ff4e88] to-[#ff2a8d] transition-all duration-500 shadow-[0_0_8px_rgba(255,78,136,0.3)]"
                    />
                    <div
                      style={{ width: `${ledgerFailed ? 0 : s.learningPct}%` }}
                      className="h-full bg-[#9d4edd] transition-all duration-500"
                    />
                  </div>

                  <span className="w-10 shrink-0 text-right text-xs font-bold text-white/90 font-mono tabular-nums">
                    {ledgerFailed ? '—' : `${s.pct}%`}
                  </span>
                </div>
              ))}
            </div>
          </div>

          <footer className="mt-4 flex h-8 items-center justify-between border-t border-white/5 pt-3 text-xs text-white/40">
            <span>全部必需题型掌握，才是完全掌握</span>
            <button
              type="button"
              onClick={() => nav(`/subjects/${subjectsList[0]?.code ?? 'literacy'}`)}
              className="flex items-center gap-1 text-xs font-medium text-pink-400 hover:text-pink-300 transition-colors"
            >
              <span>查看掌握详情</span>
              <span>➔</span>
            </button>
          </footer>
        </section>

        <section className="dash-card flex flex-col justify-between p-5 rounded-2xl bg-[#170e26]/90 border border-white/[0.08] shadow-xl">
          <div>
            <header className="mb-4 flex h-7 items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="grid h-7 w-7 place-items-center rounded-lg bg-pink-500/15 border border-pink-500/30 text-pink-400">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
                  </svg>
                </span>
                <h2 className="text-sm font-bold tracking-wide text-white">各科题型掌握</h2>
              </div>
              <div className="flex items-center gap-3 text-xs text-white/50">
                <span className="flex items-center gap-1.5">
                  <i className="h-2.5 w-2.5 rounded-full bg-[#ff4e88] shadow-[0_0_6px_rgba(255,78,136,0.6)]" />已掌握
                </span>
                <span className="flex items-center gap-1.5">
                  <i className="h-2.5 w-2.5 rounded-full bg-[#9d4edd]" />已作答未掌握
                </span>
                <span className="flex items-center gap-1.5">
                  <i className="h-2.5 w-2.5 rounded-full bg-[#251433] border border-white/15" />未作答
                </span>
              </div>
            </header>

            {ledgerFailed ? <p role="alert" className="mb-3 text-sm text-rose-300">题型掌握加载失败，未使用示意数字。</p> : null}
            <div className="flex flex-col">
              {questionTypeRows.map((sqt) => (
                <div key={sqt.code} className="flex h-9 items-center gap-3">
                  <div className="flex w-20 shrink-0 items-center gap-2">
                    <SubjectBadge code={sqt.code} size="md" />
                    <span className="truncate text-xs font-semibold text-white/90">{sqt.name}</span>
                  </div>

                  <div className="grid grid-cols-5 flex-1 gap-2 min-w-[450px]">
                    {[0, 1, 2, 3, 4].map((slotIdx) => {
                      const t = sqt.types[slotIdx]
                      if (!t) {
                        return <div key={slotIdx} className="invisible" />
                      }
                      const bars = questionTypeBars(t.total, t.mastered, t.attempted)
                      return (
                        <div key={t.code} className="flex items-center gap-1.5 min-w-0">
                          <span
                            className="w-[48px] shrink-0 text-[10.5px] text-slate-300 font-normal truncate"
                            title={`${t.name}：掌握 ${t.mastered} / 已作答未掌握 ${t.attempted} / 共 ${t.total}`}
                          >
                            {t.name}
                          </span>
                          <div className="h-2 w-9 shrink-0 overflow-hidden rounded-full bg-[#251433] flex">
                            <div
                              style={{ width: `${ledgerFailed ? 0 : bars.masteredPct}%` }}
                              className="h-full bg-[#ff4e88]"
                            />
                            <div
                              style={{ width: `${ledgerFailed ? 0 : bars.attemptedPct}%` }}
                              className="h-full bg-[#9d4edd]"
                            />
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <footer className="mt-4 flex h-8 items-center justify-between border-t border-white/5 pt-3 text-xs text-white/40">
            <span>英语为听音选词/看图选词/组句子/写单词/读一读；无记录为 0</span>
            <span>按技能账本统计，不使用示意数字</span>
          </footer>
        </section>
      </section>
    </div>
  )
}
