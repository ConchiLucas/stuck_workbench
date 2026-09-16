import type { ReactNode } from 'react'
import { CaretLeft, CaretRight, X } from '@phosphor-icons/react'
import { Link } from 'react-router-dom'

function TopIcon({
  to,
  label,
  variant,
  children,
}: {
  to?: string
  label: string
  variant: 'close' | 'prev' | 'next'
  children: ReactNode
}) {
  const className = `top-icon top-icon-${variant}`
  const icon = <span aria-hidden="true">{children}</span>
  if (!to) return <span className={className} aria-label={label} aria-disabled="true">{icon}</span>
  return <Link className={className} to={to} aria-label={label}>{icon}</Link>
}

export function PracticeStage({
  label,
  done,
  total,
  prevHref,
  nextHref,
  className = '',
  children,
}: {
  label: string
  done: number
  total: number
  prevHref?: string
  nextHref?: string
  className?: string
  children: ReactNode
}) {
  const progress = total ? Math.min(100, Math.max(0, (done / total) * 100)) : 0
  return (
    <section className={`practice-page page-enter ${className}`.trim()}>
      <div className="practice-top">
        <TopIcon to="/" label="退出练习" variant="close"><X weight="bold" /></TopIcon>
        <div className="progress-track" role="progressbar" aria-label={`${label}，${done} / ${total}`} aria-valuemin={0} aria-valuemax={total} aria-valuenow={done}>
          <i style={{ width: `${progress}%` }} />
          <span className="progress-label" aria-hidden="true">{done} / {total}</span>
        </div>
        <div className="practice-nav">
          <TopIcon to={prevHref} label="上一题" variant="prev"><CaretLeft weight="bold" /></TopIcon>
          <TopIcon to={nextHref} label="下一题" variant="next"><CaretRight weight="bold" /></TopIcon>
        </div>
      </div>
      {children}
    </section>
  )
}
