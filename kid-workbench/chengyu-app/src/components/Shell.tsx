import { Outlet, useLocation } from 'react-router-dom'

export function Shell() {
  const location = useLocation()
  const immersive = location.pathname === '/' || location.pathname.startsWith('/practice')
  return <div className={immersive ? 'app-shell immersive' : 'app-shell'}>
    <main><Outlet /></main>
  </div>
}
