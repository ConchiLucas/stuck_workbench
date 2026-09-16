import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { mathApi } from '../api/math'
import type { Home, PlanDetail } from '../api/types'
import { useChildStore } from '../store/childStore'

export function ResultPage() {
  const childId = useChildStore((state) => state.childId)
  const planId = Number(useParams().planId)
  const [detail, setDetail] = useState<PlanDetail>()
  const [home, setHome] = useState<Home>()
  useEffect(() => { Promise.all([mathApi.plan(childId, planId), mathApi.home(childId)]).then(([plan, summary]) => { setDetail(plan); setHome(summary) }) }, [childId, planId])
  if (!detail || !home) return <section className="page result-page"><p className="loading">正在数一数你的星星…</p></section>
  const { plan } = detail
  const reward = 1 + plan.stars
  return <section className="page result-page">
    <div className="result-sheet">
      <p className="eyebrow">TODAY'S WORK</p><h1>练习完成</h1><p className="result-message">认真完成，比做得快更重要。</p>
      <div className="stars" aria-label={`获得 ${plan.stars} 颗星`}>{[0, 1, 2].map((star) => <span key={star} className={star < plan.stars ? 'earned' : ''}>★</span>)}</div>
      <div className="result-stats"><div><strong>{plan.correctCount} / {plan.targetCount}</strong><span>答对题数</span></div><div><strong>{Math.floor(plan.durationSec / 60)}:{String(plan.durationSec % 60).padStart(2, '0')}</strong><span>练习时间</span></div><div><strong>+{reward} 🌼</strong><span>本次奖励</span></div></div>
      <div className="flower-total">花朵总数 <strong>🌼 {home.child.flowers} 朵</strong></div>
      <div className="result-actions"><Link className="primary-button" to="/">回到首页</Link><Link className="secondary-button" to="/map">看看学习地图</Link></div>
    </div>
  </section>
}
