import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { fetchOverview } from '../api/client'
import type { StatusCounts } from '../api/types'
import { getChildId } from '../store/childStore'
import { healthLabel, pct, statusLabel } from '../lib/labels'

export function OverviewPage() {
  const childId = getChildId()
  const q = useQuery({
    queryKey: ['diagnosis', 'overview', childId],
    queryFn: () => fetchOverview(childId),
  })

  if (q.isPending) {
    return <p className="state-line">正在读取病历……</p>
  }
  if (q.isError) {
    return <p className="state-line error">{q.error.message}</p>
  }

  const data = q.data
  const urgent = (data.urgent ?? []).filter((item) => item.subject_code !== 'game')
  const subjects = (data.subjects ?? []).filter((s) => s.code !== 'game')

  return (
    <article>
      <header className="page-head">
        <p className="eyebrow">Impression</p>
        <h1>诊断总览</h1>
      </header>
      <blockquote className="headline">{data.headline}</blockquote>

      <section className="panel" aria-labelledby="subject-health">
        <h2 id="subject-health">学科健康度</h2>
        <div className="subject-grid">
          {subjects.map((s) => (
            <Link key={s.code} to={`/subjects/${s.code}`} className={`health-card health-${s.health}`}>
              <div className="health-card-top">
                <span className="stamp">{healthLabel(s.health)}</span>
                <span className="mono">{s.code}</span>
              </div>
              <h3>{s.name}</h3>
              <p>{s.summary}</p>
              <CountStrip counts={s.counts} total={s.total} />
            </Link>
          ))}
        </div>
      </section>

      <section className="panel" aria-labelledby="urgent">
        <h2 id="urgent">最急薄弱点</h2>
        {urgent.length === 0 ? (
          <p className="empty">目前没有 shaky / 到期复习的知识点。</p>
        ) : (
          <ol className="urgent-list">
            {urgent.map((item) => (
              <li key={item.kp_id}>
                <Link to={`/knowledge-points/${item.kp_id}`}>
                  <span className="urgent-title">{item.title}</span>
                  <span className="urgent-meta">
                    {item.subject_name} · {item.module_name} · {statusLabel(item.status)} · {pct(item.accuracy)}
                  </span>
                  <span className="urgent-reason">{item.reason}</span>
                </Link>
              </li>
            ))}
          </ol>
        )}
      </section>
    </article>
  )
}

export function CountStrip({ counts, total }: { counts: StatusCounts; total: number }) {
  return (
    <ul className="count-strip">
      <li>学习 {counts.learning}</li>
      <li>巩固 {counts.shaky}</li>
      <li>复习 {counts.review_due}</li>
      <li>掌握 {counts.mastered}</li>
      <li>共 {total}</li>
    </ul>
  )
}
