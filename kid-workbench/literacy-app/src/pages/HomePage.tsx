import { appPath } from '../appPath'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { literacyApi } from '../api/literacy'
import { useChildStore } from '../store/childStore'

const questionTypes = [
  { key: 'glyph', title: '看字选义', copy: '看字，选意思', to: '/practice/type/glyph', image: appPath('/types/glyph.png') },
  { key: 'sense', title: '看义选字', copy: '看意思，选字', to: '/practice/type/sense', image: appPath('/types/sense.png') },
  { key: 'write', title: '写一写', copy: '在田字格里写完', to: '/practice/type/write', image: appPath('/types/write.png') },
  { key: 'map', title: '识字图', copy: '看看学过的字', to: '/map', image: appPath('/types/map.png') },
] as const

export function HomePage() {
  const childId = useChildStore((s) => s.childId)
  const home = useQuery({ queryKey: ['home', childId], queryFn: () => literacyApi.home(childId) })
  return <section className="home-page" aria-label="题型">
    <div className="type-grid">
      {questionTypes.map((type) => <Link
        className="type-card"
        to={type.to}
        key={type.key}
        aria-label={`${type.title}，${type.copy}`}
      >
        <img src={type.image} alt="" />
        <h2>{type.title}</h2>
      </Link>)}
    </div>
    {home.isError && <button className="retry-card" onClick={() => home.refetch()}>内容没加载出来，点这里再试一次</button>}
  </section>
}
