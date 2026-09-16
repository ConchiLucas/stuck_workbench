import { Outlet, useLocation, useSearchParams } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { KpDetailDrawer } from '../mastery/KpDetailDrawer'
import { useChildStore } from '../../store/childStore'
import { useEffect } from 'react'

export function Shell() {
  const { pathname } = useLocation()
  const [params] = useSearchParams()
  const setChild = useChildStore((s) => s.setChild)
  useEffect(() => {
    const n = Number(params.get('child'))
    if (Number.isFinite(n) && n > 0) setChild(n)
  }, [params, setChild])
  const home = pathname === '/'
  const literacy = pathname === '/subjects/literacy'
  const math = pathname === '/subjects/math'
  const pinyin = pathname === '/subjects/pinyin'
  const english = pathname === '/subjects/english'
  const phrase = pathname === '/subjects/phrase'

  return (
    <div className={`flex min-h-screen bg-[#0f0814] text-[#f7eef5] ${literacy ? 'literacy-shell' : ''} ${pinyin ? 'pinyin-shell' : ''} ${math ? 'math-shell' : ''} ${english ? 'english-shell' : ''} ${phrase ? 'phrase-shell' : ''}`}>
      <div
        className="pointer-events-none fixed inset-0 overflow-hidden"
        style={{
          background:
            'radial-gradient(1100px 500px at 85% -5%, rgba(255, 78, 136, 0.18), transparent 60%), radial-gradient(800px 600px at -10% 15%, rgba(147, 40, 110, 0.22), transparent 55%), radial-gradient(700px 500px at 50% 100%, rgba(90, 24, 110, 0.15), transparent 60%)',
        }}
      />
      <Sidebar />
      <div className="relative flex min-w-0 flex-1 flex-col">
        {!home && !literacy && !pinyin && !math && !english && !phrase && <TopBar />}
        <main className="p-8 pt-6"><Outlet /></main>
      </div>

      <KpDetailDrawer />
    </div>
  )
}
