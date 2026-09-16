import { X } from '@phosphor-icons/react'
import { Link, Outlet, useLocation } from 'react-router-dom'

export function Shell() {
  const location = useLocation()
  const gallery = location.pathname === '/'
  const immersive = gallery || location.pathname.startsWith('/practice')
  const onMap = location.pathname === '/map'
  return <div className={immersive ? 'app-shell immersive' : 'app-shell'}>
    {immersive ? null : <header className="topbar">
      <Link className="brand" to="/" aria-label="回到首页"><span className="brand-mark">b</span><span>拼音星球</span></Link>
      {onMap ? <Link className="top-icon top-icon-close" to="/" aria-label="关闭"><span aria-hidden="true"><X weight="bold" /></span></Link> : null}
    </header>}
    <main><Outlet /></main>
  </div>
}
