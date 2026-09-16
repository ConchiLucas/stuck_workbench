import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { fetchSubject } from '../api/client'
import { getChildId } from '../store/childStore'
import { healthLabel } from '../lib/labels'
import { CountStrip } from './OverviewPage'

export function SubjectPage() {
  const { code = '' } = useParams()
  const childId = getChildId()
  const q = useQuery({
    queryKey: ['diagnosis', 'subject', childId, code],
    queryFn: () => fetchSubject(childId, code),
    enabled: code.length > 0,
  })

  if (q.isPending) {
    return <p className="state-line">正在读取学科诊断……</p>
  }
  if (q.isError) {
    return <p className="state-line error">{q.error.message}</p>
  }

  const data = q.data
  const gaps = data.skill_gaps ?? []
  const modules = data.modules ?? []

  return (
    <article>
      <p className="crumb">
        <Link to="/">诊断总览</Link>
        <span> / {data.name}</span>
      </p>
      <header className="page-head">
        <p className="eyebrow">Subject chart</p>
        <h1>学科诊断 · {data.name}</h1>
        <span className={`stamp stamp-lg health-${data.health}`}>{healthLabel(data.health)}</span>
      </header>
      <blockquote className="headline">{data.headline}</blockquote>
      <CountStrip counts={data.counts} total={data.total} />

      <section className="panel" aria-labelledby="gaps">
        <h2 id="gaps">技能缺口</h2>
        {gaps.length === 0 ? (
          <p className="empty">这个学科目前没有「一种会、一种不会」的明显缺口。</p>
        ) : (
          <ul className="gap-list">
            {gaps.map((g) => (
              <li key={`${g.kp_id}-${g.weak_skill}`}>
                <Link to={`/knowledge-points/${g.kp_id}`}>
                  <span className="urgent-title">{g.title}</span>
                  <span className="urgent-meta">
                    {g.strong_label}已掌握 · {g.weak_label}仍弱
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="panel" aria-labelledby="modules">
        <h2 id="modules">模块</h2>
        <div className="module-grid">
          {modules.map((m) => (
            <div key={m.code} className="module-card">
              <h3>{m.name}</h3>
              <p className="mono">{m.code}</p>
              <CountStrip counts={m.counts} total={m.total} />
            </div>
          ))}
        </div>
      </section>
    </article>
  )
}
