import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { scienceApi } from '../api/science'
import { StatusBadge } from '../components/StatusBadge'
import { useChildStore } from '../store/childStore'

function ModuleGroup({ code, name, progress }: { code: string; name: string; progress: Map<number, string> }) {
  const items = useQuery({ queryKey: ['items', code], queryFn: () => scienceApi.items(code) })
  return <section className="map-group"><div className="section-heading"><h2>{name}</h2><span>{items.data?.length ?? 0} 个</span></div>
    <div className="letter-grid">{items.data?.map((item) => <Link className="letter-card" to={`/concept/${item.kpId}`} key={item.kpId}>
      <strong>{item.title}</strong><StatusBadge status={progress.get(item.kpId) ?? 'not_started'} />
    </Link>)}</div>
  </section>
}

export function MapPage() {
  const childId = useChildStore((s) => s.childId)
  const modules = useQuery({ queryKey: ['modules'], queryFn: scienceApi.modules })
  const progress = useQuery({ queryKey: ['progress', childId], queryFn: () => scienceApi.progress(childId) })
  const progressMap = new Map(progress.data?.map((item) => [item.kpId, item.status]))
  return <div className="map-page page-enter"><div className="page-title"><p className="eyebrow">选择一片未知区域</p><h1>科普地图</h1></div>
    {modules.isLoading && <div className="skeleton-block">科普正在排队……</div>}
    {modules.isError && <button className="retry-card" onClick={() => modules.refetch()}>科普图没打开，点这里再试一次</button>}
    {modules.data?.length === 0 && <p className="empty-card">还没有科普内容</p>}
    {modules.data?.map((module) => <ModuleGroup key={module.code} {...module} progress={progressMap} />)}
  </div>
}
