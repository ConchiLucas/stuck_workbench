import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { literacyApi } from '../api/literacy'
import { ExitIcon, StarIcon } from '../components/Icons'
import { useChildStore } from '../store/childStore'

export function ResultPage() {
  const planId = Number(useParams().planId)
  const childId = useChildStore((s) => s.childId)
  const detail = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => literacyApi.plan(childId, planId) })
  const plan = detail.data?.plan
  return <section className="result-page page-enter">
    <Link className="exit-button result-exit" to="/"><ExitIcon />退出</Link>
    <h1>练习完成</h1>
    <div className="stars" aria-label={`${plan?.stars ?? 0} 颗星`}>{[1, 2, 3].map((star) => <span className={star <= (plan?.stars ?? 0) ? 'lit' : ''} key={star}><StarIcon lit={star <= (plan?.stars ?? 0)} /></span>)}</div>
    <p className="result-count">{plan?.correctCount ?? '—'} / {plan?.doneCount ?? '—'} 题答对</p>
    <div className="result-actions"><Link className="primary-button" to="/">回首页</Link></div>
  </section>
}
