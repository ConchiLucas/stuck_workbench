import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { pinyinApi } from '../api/pinyin'
import { StatusBadge } from '../components/StatusBadge'
import { useChildStore } from '../store/childStore'

function ModuleGroup({ code, name, progress }: { code: string; name: string; progress: Map<number, string> }) {
  const items = useQuery({ queryKey: ['items', code], queryFn: () => pinyinApi.items(code) })
  return <section className="map-group"><div className="section-heading"><h2>{name}</h2><span>{items.data?.length ?? 0} 个</span></div>
    <div className="letter-grid">{items.data?.map((item) => <Link className="letter-card" to={`/learn/${item.kpId}`} key={item.kpId}>
      <strong>{item.letter}</strong><StatusBadge status={progress.get(item.kpId) ?? 'not_started'} />
    </Link>)}</div>
  </section>
}

export function MapPage() {
  const childId = useChildStore((s) => s.childId)
  const modules = useQuery({ queryKey: ['modules'], queryFn: pinyinApi.modules })
  const progress = useQuery({ queryKey: ['progress', childId], queryFn: () => pinyinApi.progress(childId) })
  const progressMap = new Map(progress.data?.map((item) => [item.kpId, item.status]))
  return <div className="map-page page-enter"><div className="page-title"><p className="eyebrow">按顺序认识它们</p><h1>我的拼音图</h1></div>
    {modules.isLoading && <div className="skeleton-block">拼音正在排队……</div>}
    {modules.isError && <button className="retry-card" onClick={() => modules.refetch()}>拼音图没打开，点这里再试一次</button>}
    {modules.data?.length === 0 && <p className="empty-card">还没有拼音内容</p>}
    {modules.data?.map((module) => <ModuleGroup key={module.code} {...module} progress={progressMap} />)}
  </div>
}
