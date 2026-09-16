import { Link, Outlet, useLocation } from 'react-router-dom'
import { BrandIcon } from './Icons'

export function Shell() {
  const location = useLocation()
  const immersive = location.pathname.startsWith('/practice')
  const home = location.pathname === '/'
  return <div className={immersive ? 'app-shell immersive' : 'app-shell'}>
    {immersive || home ? null : <header className="topbar">
      <Link className="brand" to="/" aria-label="田字格"><span className="brand-mark"><BrandIcon /></span><span className="brand-name">田字格</span></Link>
    </header>}
    <main><Outlet /></main>
  </div>
}
