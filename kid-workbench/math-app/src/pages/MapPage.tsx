import { Link } from 'react-router-dom'
import { useMathModules, useMathProgress } from '../api/math'
import type { MasteryStatus } from '../api/types'
import { useChildStore } from '../store/childStore'

const statusLabel: Record<MasteryStatus, string> = { not_started: '未开始', learning: '学习中', shaky: '再练练', review_due: '该复习', mastered: '已掌握' }
const statusMark: Record<MasteryStatus, string> = { not_started: '○', learning: '◐', shaky: '△', review_due: '↻', mastered: '✓' }

export function MapPage() {
  const childId = useChildStore((state) => state.childId)
  const modules = useMathModules().data
  const progress = useMathProgress(childId).data
  return <section className="page map-page">
    <header className="page-header"><Link className="back-link" to="/">← 返回</Link><div><p className="eyebrow">LEARNING MAP</p><h1>算数地图</h1><p>一步一步，把每个小本领练扎实。</p></div></header>
    <div className="map-list">{modules?.map((module, moduleIndex) => {
      const moduleProgress = progress?.modules.find((value) => value.code === module.code)
      return <article className={`map-module ${module.code}`} key={module.code}>
        <div className="map-module-title"><span>{String(moduleIndex + 1).padStart(2, '0')}</span><div><h2>{module.name}</h2><p>{module.itemCount} 个知识点</p></div><Link to={`/module/${module.code}`}>进入模块 →</Link></div>
        <div className="stage-list">{module.stages.map((stage) => {
          const stageProgress = moduleProgress?.stages.find((value) => value.code === stage.code)
          return <section key={stage.code}><h3>{stage.name}</h3><div className="knowledge-list">{stageProgress?.items.map((item) => <div className={`knowledge-pill ${item.status}`} key={item.kpId}>
            <b aria-hidden>{statusMark[item.status]}</b><span>{item.title}</span><small>{statusLabel[item.status]}</small>
          </div>)}</div></section>
        })}</div>
      </article>
    })}</div>
  </section>
}
