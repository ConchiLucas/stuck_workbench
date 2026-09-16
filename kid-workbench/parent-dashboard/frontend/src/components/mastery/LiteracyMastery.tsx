import { useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useMatrix } from '../../api/dashboard'
import type { MatrixPoint } from '../../api/types'
import { useChildStore } from '../../store/childStore'
import { LITERACY_TYPES, literacyState, literacySummary, type LiteracyView } from '../../lib/literacyMastery'
import './literacy.css'

export function MasteryBar({ mastered, answered, total, label }: {
  mastered: number; answered: number; total: number; label: string
}) {
  const remaining = Math.max(0, total - mastered - answered)
  const tip = `${label}：已掌握 ${mastered} · 已作答未掌握 ${answered} · 未作答 ${remaining}（共 ${total}）`
  return <div className="literacy-bar" role="img" aria-label={tip} title={tip}>
    <span className="is-mastered" style={{ width: `${total ? mastered / total * 100 : 0}%` }} />
    <span className="is-answered" style={{ width: `${total ? answered / total * 100 : 0}%` }} />
  </div>
}

export function LiteracyMastery() {
  const childId = useChildStore(s => s.childId)
  const { data, isLoading, isError, refetch } = useMatrix(childId, 'literacy')
  const [params, setParams] = useSearchParams()
  const rawView = params.get('type')
  const view: LiteracyView = LITERACY_TYPES.some(t => t.code === rawView) ? rawView as LiteracyView : 'all'
  const setView = (next: LiteracyView) => setParams(previous => {
    const updated = new URLSearchParams(previous)
    if (next === 'all') updated.delete('type')
    else updated.set('type', next)
    return updated
  }, { replace: true })
  const [search, setSearch] = useState('')
  const points = useMemo(() => (data?.modules ?? []).flatMap(m => m.points ?? []), [data])
  const summary = useMemo(() => literacySummary(points), [points])
  const modules = (data?.modules ?? []).map(m => ({ ...m,
    visiblePoints: (m.points ?? []).filter(p => !search.trim() || p.title.includes(search.trim())),
  })).filter(m => m.visiblePoints.length > 0)
  const ratio = summary.overall.total ? summary.overall.mastered / summary.overall.total : 0
  const overallTip = `完全掌握 ${summary.overall.mastered} / ${summary.overall.total} 个字`

  return <div className="literacy-page">
    <header className="literacy-heading">
      <Link to="/">进度总览 / 识字</Link>
      <h1>识字掌握</h1>
      <p>看字选义、看义选字、手写全部掌握，才算完全掌握这个字。</p>
    </header>
    {isLoading ? <div className="dash-card literacy-empty" role="status">正在读取识字掌握情况…</div>
      : isError ? <div className="dash-card literacy-empty" role="alert">识字数据暂时无法读取。<button onClick={() => void refetch()}>重新加载</button></div>
      : <>
        <section className="dash-card literacy-summary" aria-label="识字整体与题型掌握">
          <div className="literacy-overall" tabIndex={0} title={overallTip} aria-label={overallTip}>
            <svg viewBox="0 0 80 80" aria-hidden="true"><circle cx="40" cy="40" r="30" className="ring-track" />{ratio > 0 && <circle cx="40" cy="40" r="30" className="ring-fill" pathLength="100" strokeDasharray={`${ratio * 100} 100`} transform="rotate(-90 40 40)" />}</svg>
            <span>整体掌握</span>
          </div>
          <div className="literacy-type-summary">
            <div className="literacy-type-bars">{LITERACY_TYPES.map(type => <button key={type.code} onClick={() => setView(type.code)} aria-label={`按${type.name}查看掌握情况`}>
              <span>{type.name}</span>
              <MasteryBar {...summary.types[type.code]} label={type.name} />
            </button>)}</div>
            <div className="literacy-legend" aria-label="题型进度图例"><span><i className="is-mastered" />已掌握</span><span><i className="is-answered" />已作答未掌握</span><span><i className="is-unattempted" />未作答</span></div>
          </div>
        </section>
        <div className="literacy-toolbar">
          <div className="literacy-tabs" role="group" aria-label="选择掌握视图">{[{ code: 'all', name: '整体掌握' }, ...LITERACY_TYPES].map(type => <button key={type.code} aria-pressed={view === type.code} onClick={() => setView(type.code as LiteracyView)}>{type.name}</button>)}</div>
          <label className="literacy-search"><span aria-hidden="true">⌕</span><input aria-label="查找汉字" placeholder="查找汉字" value={search} onChange={e => setSearch(e.target.value)} />{search && <button aria-label="清空搜索" onClick={() => setSearch('')}>×</button>}</label>
        </div>
        <div className="literacy-map-caption">
          <span>{view === 'all' ? '全部题型掌握的字显示 ✓' : `当前按「${LITERACY_TYPES.find(t => t.code === view)?.name}」查看`}</span>
          <span>义 · 字 · 写：<b>亮起已掌握</b>，暗色未掌握 · 悬浮查看详情</span>
        </div>
        {!points.length ? <div className="dash-card literacy-empty">暂无识字内容。</div>
          : !modules.length ? <div className="dash-card literacy-empty">没有找到「{search}」。<button onClick={() => setSearch('')}>显示全部汉字</button></div>
          : modules.map(module => {
            const modulePoints = module.points ?? []
            const stats = literacySummary(modulePoints)
            const m = view === 'all' ? stats.overall.mastered : stats.types[view].mastered
            const moduleTip = `${module.name} · ${view === 'all' ? '完全掌握' : LITERACY_TYPES.find(t => t.code === view)?.name} ${m} / ${modulePoints.length}`
            return <section key={module.code} className="dash-card literacy-unit" aria-label={module.name}>
              <header><h2>{module.name}</h2><div className="literacy-unit-progress" role="img" title={moduleTip} aria-label={moduleTip}><span style={{ width: `${modulePoints.length ? m / modulePoints.length * 100 : 0}%` }} /></div></header>
              <div className="literacy-grid">{module.visiblePoints.map(point => <LiteracyCard key={point.id} point={point} view={view} />)}</div>
            </section>
          })}
      </>}
  </div>
}

function LiteracyCard({ point, view }: { point: MatrixPoint; view: LiteracyView }) {
  const open = useChildStore(s => s.openKp)
  const selected = useChildStore(s => s.kpDrawerId === point.id)
  const state = literacyState(point)
  const activeSkill = state.skills.find(s => s.code === view)
  const lit = activeSkill ? activeSkill.lit : state.complete
  const partial = !activeSkill && !state.complete && state.mastered > 0
  const summary = `${point.title} · ${activeSkill ? `${activeSkill.name}：${activeSkill.detail}` : state.label}`
  const detail = state.skills.map(s => `${s.name}：${s.detail}`).join('；')
  return <button className={`literacy-char ${lit ? 'is-complete' : partial ? 'is-partial' : ''}`} data-selected={selected} onClick={() => open(point.id)} title={`${summary}\n${detail}`} aria-label={`${summary}；${detail}；查看详情`}>
    <span className="literacy-glyph">{point.title}</span>
    {state.complete && <span className="literacy-complete-mark" aria-label="完全掌握">✓</span>}
    <span className="literacy-skill-marks" aria-label="题型掌握">{state.skills.map(s => <span key={s.code} className={s.lit ? 'lit' : 'unlit'} data-active={view === s.code} title={`${s.name}：${s.detail}`} aria-label={`${s.name}：${s.lit ? '已掌握' : '未掌握'}`}>{s.short}</span>)}</span>
    <span className="literacy-card-tip" aria-hidden="true"><strong>{point.title} · {state.label}</strong>{state.skills.map(s => <span key={s.code}>{s.name}<b>{s.detail}</b></span>)}</span>
  </button>
}
