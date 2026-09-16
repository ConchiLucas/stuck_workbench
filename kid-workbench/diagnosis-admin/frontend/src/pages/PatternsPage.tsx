import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { fetchErrorPatterns } from '../api/client'
import { getChildId } from '../store/childStore'
import { patternKindLabel } from '../lib/labels'

export function PatternsPage() {
  const childId = getChildId()
  const q = useQuery({
    queryKey: ['diagnosis', 'patterns', childId],
    queryFn: () => fetchErrorPatterns(childId),
  })

  if (q.isPending) {
    return <p className="state-line">正在归纳错因……</p>
  }
  if (q.isError) {
    return <p className="state-line error">{q.error.message}</p>
  }

  const patterns = q.data.patterns ?? []

  return (
    <article>
      <header className="page-head">
        <p className="eyebrow">Error patterns</p>
        <h1>错因</h1>
      </header>
      <p className="lede">从技能缺口、计划错选和超时错题里归纳，不另写一套掌握度。</p>
      {patterns.length === 0 ? (
        <p className="empty">目前没有可归纳的错因。</p>
      ) : (
        <ul className="pattern-list">
          {patterns.map((p, i) => (
            <li key={`${p.kind}-${p.kp_id ?? i}-${i}`} className={`pattern kind-${p.kind}`}>
              {p.kp_id ? (
                <Link to={`/knowledge-points/${p.kp_id}`}>
                  <span className="stamp">{patternKindLabel(p.kind)}</span>
                  <strong>{p.title}</strong>
                  <span>{p.detail}</span>
                </Link>
              ) : (
                <div>
                  <span className="stamp">{patternKindLabel(p.kind)}</span>
                  <strong>{p.title}</strong>
                  <span>{p.detail}</span>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </article>
  )
}
