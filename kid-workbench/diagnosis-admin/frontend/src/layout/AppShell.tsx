import type { PropsWithChildren } from 'react'
import { BarChart2, BookOpen, ExternalLink, FileText, LayoutGrid, Settings } from 'lucide-react'
import { NavLink, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { knowledge } from '../api/knowledge'
import type { Summary } from '../api/knowledgeTypes'
import { progressUrl } from '../lib/progressUrl'
import { getChildId } from '../store/childStore'
import '../styles/overview.css'

export function AppShell({ children }: PropsWithChildren) {
  const childId = getChildId()
  const { pathname } = useLocation()
  const home = pathname === '/'
  const overview = useQuery({ queryKey: ['knowledge', 'summary', childId], queryFn: () => knowledge<Summary>('/summary') })
  const child = overview.data?.child
  const libraryActive = pathname.startsWith('/library') || pathname.startsWith('/subjects/') || pathname.startsWith('/knowledge-points/')
  const wrongsActive = pathname.startsWith('/wrongs') || pathname.startsWith('/attempts/')
  const reviewsActive = pathname.startsWith('/reviews')

  return (
    <div className={`knowledge-workbench${home ? ' knowledge-workbench-home' : ''}`}>
      <aside className="workbench-sidebar">
        <div className="workbench-brand">
          <span className="brand-icon-box">
            <BookOpen size={22} aria-hidden="true" />
          </span>
          <div className="brand-text">
            <span className="brand-title">启心学堂</span>
            <span className="brand-sub">孩子知识库</span>
          </div>
        </div>
        <nav className="workbench-nav" aria-label="知识库导航">
          <NavLink to="/" end className={({ isActive }) => (isActive ? 'active' : '')}>
            <LayoutGrid size={18} aria-hidden="true" />
            <span>知识总览</span>
          </NavLink>
          <NavLink to="/library" className={libraryActive ? 'active' : ''} aria-current={libraryActive ? 'page' : undefined}>
            <BarChart2 size={18} aria-hidden="true" />
            <span>学习数据</span>
          </NavLink>
          <NavLink to="/wrongs" className={wrongsActive ? 'active' : ''} aria-current={wrongsActive ? 'page' : undefined}>
            <FileText size={18} aria-hidden="true" />
            <span>错题记录</span>
          </NavLink>
          <NavLink to="/reviews" className={reviewsActive ? 'active' : ''} aria-current={reviewsActive ? 'page' : undefined}>
            <Settings size={18} aria-hidden="true" />
            <span>设置</span>
          </NavLink>
        </nav>
        <div className="workbench-sidebar-bottom">
          <div className="sidebar-avatar-circle">
            <svg viewBox="0 0 36 36" width="36" height="36" fill="none">
              <circle cx="18" cy="18" r="18" fill="#e0e7ff" />
              <circle cx="18" cy="15" r="9" fill="#fbcfe8" />
              <path d="M9 16c0-6 4-10 9-10s9 4 9 10c0 1-1 3-2 3-2-2-4-3-7-3s-5 1-7 3c-1 0-2-2-2-3z" fill="#4338ca" />
              <path d="M10 13c1 3 3 5 5 5" stroke="#312e81" strokeWidth="1.5" strokeLinecap="round" />
              <path d="M26 13c-1 3-3 5-5 5" stroke="#312e81" strokeWidth="1.5" strokeLinecap="round" />
              <circle cx="14" cy="16" r="1.2" fill="#1e1b4b" />
              <circle cx="22" cy="16" r="1.2" fill="#1e1b4b" />
              <path d="M16 19.5c.8.8 3.2.8 4 0" stroke="#be185d" strokeWidth="1.2" strokeLinecap="round" />
              <circle cx="12.5" cy="18" r="1.5" fill="#f472b6" opacity="0.6" />
              <circle cx="23.5" cy="18" r="1.5" fill="#f472b6" opacity="0.6" />
              <path d="M7 34c1-6 6-10 11-10s10 4 11 10" fill="#a855f7" />
            </svg>
          </div>
          <div className="sidebar-user-info">
            <div className="sidebar-user-row">
              <span className="sidebar-user-name">{child?.name || '小朋友'}</span>
              <a href={progressUrl} target="_blank" rel="noreferrer" className="sidebar-progress-link" title="进度后台">
                <ExternalLink size={12} aria-hidden="true" />
              </a>
            </div>
            <span className="sidebar-user-motto">快乐学习 · 每天进步</span>
          </div>
        </div>
      </aside>
      <main className={`workbench-content${home ? ' workbench-overview' : ' workbench-detail'}`}>{children}</main>
    </div>
  )
}
