import type { PropsWithChildren } from 'react'
import { ClipboardList } from 'lucide-react'
import { NavLink } from 'react-router-dom'

export function AppShell({ children }: PropsWithChildren) {
  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand-mark" aria-hidden="true">
          <ClipboardList size={18} />
        </div>
        <div className="brand-block">
          <div className="brand-name">题目后台</div>
          <div className="brand-sub">Question Studio</div>
        </div>
        <span className="environment-label">INTERNAL</span>
        <nav className="top-primary-nav" aria-label="主导航">
          <NavLink to="/" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`} end>
            识字任务
          </NavLink>
          <NavLink to="/pinyin" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>拼音任务</NavLink>
          <NavLink to="/math" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>算术任务</NavLink>
          <NavLink to="/english" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>英语任务</NavLink>
          <NavLink to="/phrase" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>短句任务</NavLink>
          <NavLink to="/chengyu" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>成语任务</NavLink>
          <NavLink to="/science" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>科普任务</NavLink>
          <NavLink to="/poem" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>古诗任务</NavLink>
          <NavLink to="/logic" className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}>逻辑任务</NavLink>
        </nav>
      </header>
      <div className="workspace full">
        <main className="application-main">{children}</main>
      </div>
    </div>
  )
}
