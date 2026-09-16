import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { literacyApi } from '../api/literacy'
import { StatusBadge } from '../components/StatusBadge'
import { useChildStore } from '../store/childStore'

function ModuleGroup({ code, name, progress }: { code: string; name: string; progress: Map<number, string> }) {
  const items = useQuery({ queryKey: ['items', code], queryFn: () => literacyApi.items(code) })
  return <section className="map-group">
    <h2>{name}</h2>
    <div className="letter-grid">{items.data?.map((item) => <Link className="letter-card" to={`/learn/${item.kpId}`} key={item.kpId}>
      <strong>{item.character}</strong>
      <StatusBadge status={progress.get(item.kpId) ?? 'not_started'} />
    </Link>)}</div>
  </section>
}

export function MapPage() {
  const childId = useChildStore((s) => s.childId)
  const modules = useQuery({ queryKey: ['modules'], queryFn: literacyApi.modules })
  const progress = useQuery({ queryKey: ['progress', childId], queryFn: () => literacyApi.progress(childId) })
  const progressMap = new Map(progress.data?.map((item) => [item.kpId, item.status]))
  return <div className="map-page page-enter">
    <h1>识字图</h1>
    {modules.isLoading && <div className="skeleton-block">汉字正在排队……</div>}
    {modules.isError && <button className="retry-card" onClick={() => modules.refetch()}>识字图没打开，点这里再试一次</button>}
    {modules.data?.map((module) => <ModuleGroup key={module.code} {...module} progress={progressMap} />)}
  </div>
}
