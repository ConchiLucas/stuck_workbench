import { ExitIcon, NextIcon, PrevIcon } from './Icons'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

export function PracticeStage({
  current,
  total,
  children,
  onPrev,
  onNext,
}: {
  current: number
  total: number
  children: ReactNode
  onPrev?: () => void
  onNext?: () => void
}) {
  const progress = total ? (current / total) * 100 : 0
  return (
    <section className="practice-page page-enter">
      <div className="practice-top">
        <Link className="icon-button" to="/" aria-label="退出练习"><ExitIcon /></Link>
        <div className="progress-track" role="progressbar" aria-valuemin={0} aria-valuemax={total} aria-valuenow={current} aria-label={`第 ${current} 题，共 ${total} 题`}>
          <i style={{ width: `${Math.max(progress, 18)}%` }}><span>{current} / {total}</span></i>
        </div>
        <div className="practice-nav">
          <button className="icon-button" type="button" aria-label="上一题" disabled={current <= 1} onClick={onPrev}><PrevIcon /></button>
          <button className="icon-button nav-next" type="button" aria-label="下一题" disabled={current >= total} onClick={onNext}><NextIcon /></button>
        </div>
      </div>
      {children}
    </section>
  )
}
