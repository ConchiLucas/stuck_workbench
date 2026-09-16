import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { scienceApi } from '../api/science'
import { useChildStore } from '../store/childStore'

export function ResultPage() {
  const planId = Number(useParams().planId)
  const childId = useChildStore((s) => s.childId)
  const detail = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => scienceApi.plan(childId, planId) })
  const plan = detail.data?.plan
  return <section className="result-page page-enter"><div className="result-burst">✓</div><p className="eyebrow">今天的科普完成了</p><h1>做得真棒！</h1>
    <div className="stars" aria-label={`${plan?.stars ?? 0} 颗星`}>{[1, 2, 3].map((star) => <span className={star <= (plan?.stars ?? 0) ? 'lit' : ''} key={star}>★</span>)}</div>
    <p className="result-count">答完 {plan?.doneCount ?? '—'} 题，答对 {plan?.correctCount ?? '—'} 题</p>
    <div className="result-actions"><Link className="primary-button" to="/">回到首页</Link><Link className="secondary-button" to="/explore">继续探索</Link></div>
  </section>
}
