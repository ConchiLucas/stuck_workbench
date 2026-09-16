import { NavLink } from 'react-router-dom'
import clsx from 'clsx'
import { useSubjects, visibleSubjects } from '../../api/dashboard'
import { diagnosisUrl } from '../../lib/diagnosisUrl'
import { useChildStore } from '../../store/childStore'

function navClass(isActive: boolean) {
  return clsx(
    'flex items-center gap-2.5 rounded-xl px-3 py-2 text-sm transition-all',
    isActive
      ? 'bg-gradient-to-r from-[#ff4e88] to-[#e63578] text-white shadow-[0_4px_16px_rgba(255,78,136,0.4)] font-medium'
      : 'text-white/60 hover:bg-white/5 hover:text-white',
  )
}

import { SubjectBadge } from '../overview/SubjectIcons'

export function Sidebar() {
  const childId = useChildStore((s) => s.childId)
  const { data: subjects } = useSubjects(childId)

  return (
    <aside className="flex min-h-screen w-56 shrink-0 flex-col gap-1 border-r border-white/[0.06] bg-[#110817]/90 px-3 py-4 backdrop-blur-md">
      {/* Brand / Logo */}
      <div className="mb-4 flex items-center gap-2.5 px-2 pt-1">
        <div className="grid h-9 w-9 shrink-0 place-items-center rounded-2xl bg-pink-500/20 text-lg shadow-[0_0_12px_rgba(255,78,136,0.35)]">
          🌸
        </div>
        <div>
          <div className="text-base font-bold tracking-wide text-white">启心学堂</div>
          <div className="text-[11px] text-white/40">陪伴成长 · 看见进步</div>
        </div>
      </div>

      {/* Overview Nav */}
      <NavLink to="/" end className={({ isActive }) => navClass(isActive)}>
        <span className="text-base">🏠</span>
        <span>进度总览</span>
      </NavLink>

      {/* Subjects Section */}
      <div className="mb-1 mt-4 flex items-center gap-1.5 px-3 text-[11px] font-medium text-white/30">
        <span>📖</span>
        <span>学习内容</span>
      </div>
      <div className="sidebar-subjects flex flex-col gap-0.5">
        {visibleSubjects(subjects).map((s) => (
          <NavLink key={s.code} to={`/subjects/${s.code}`} className={({ isActive }) => navClass(isActive)}>
            <span className="flex min-w-0 flex-1 items-center gap-2.5">
              <SubjectBadge code={s.code} size="sm" />
              <span className="truncate">{s.name}</span>
            </span>
          </NavLink>
        ))}
      </div>

      {/* Bottom Menu Items */}
      <div className="my-2 border-t border-white/5" />
      <NavLink to="/tasks" className={({ isActive }) => navClass(isActive)}>
        <span className="text-sm">📋</span>
        <span>任务列表</span>
      </NavLink>
      <NavLink to="/rewards" className={({ isActive }) => navClass(isActive)}>
        <span className="text-sm">🎁</span>
        <span>奖励商店</span>
      </NavLink>
      <a href={diagnosisUrl} className={navClass(false)} target="_blank" rel="noreferrer">
        <span className="text-sm">⚙️</span>
        <span>设置</span>
      </a>

      {/* Bottom Mascot/Quote Card */}
      <div className="mt-auto flex items-center justify-between rounded-2xl border border-white/[0.08] bg-[#1a0f24]/90 p-3 shadow-md hover:border-pink-500/20 transition-all cursor-pointer">
        <div className="flex items-center gap-2.5">
          <span className="grid h-8 w-8 place-items-center rounded-xl bg-emerald-500/20 text-sm text-emerald-400">
            🌱
          </span>
          <div className="text-[11px] font-medium leading-snug text-white/90">
            让学习<br />成为更好的自己
          </div>
        </div>
        <span className="text-white/30 text-xs font-bold">❯</span>
      </div>
    </aside>
  )
}
