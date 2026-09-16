import { Link, Outlet, useLocation } from 'react-router-dom'

export function Shell() {
  const location = useLocation()
  const answering = location.pathname.startsWith('/question-types/')
  const gallery = location.pathname === '/'
  return <div className={answering || gallery ? 'app-shell is-immersive' : 'app-shell'}>
    {answering || gallery ? null : <header className="topbar">
      <Link className="brand" to="/" aria-label="回到首页"><span className="brand-mark">✦</span><span>小小发现局</span></Link>
      <Link className="quiet-button" to="/">回到首页</Link>
    </header>}
    <main><Outlet /></main>
  </div>
}
