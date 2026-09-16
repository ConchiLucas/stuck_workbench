import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../api/client'
import type { Matrix } from '../../../api/types'
import { useChildStore } from '../../../store/childStore'
import { MATH_TYPES, mathGroups, mathState, mathSummary, type MathCounts, type MathPoint } from '../../../lib/mathMastery'
import { barWidths, formatPinyinDate } from '../../../lib/pinyinMastery'
import { ShapeGlyph } from '../../question/ShapeGlyph'
import './math.css'

const SHAPES: Record<string, string> = { 圆形: 'circle', 正方形: 'square', 长方形: 'rect', 三角形: 'triangle', 椭圆形: 'oval', 梯形: 'trapezoid', 菱形: 'rhombus', 五角星: 'star' }
function MathBar({ counts, label }: { counts: MathCounts; label: string }) {
  const widths = barWidths(counts)
  const tip = `${label}：已掌握 ${counts.mastered} · 已作答未掌握 ${counts.answered} · 未作答 ${counts.unattempted}（共 ${counts.total}）`
  return <div className="math-bar" role="img" title={tip} aria-label={tip}><span className="is-mastered" style={{ width: `${widths.mastered}%` }} /><span className="is-answered" style={{ width: `${widths.answered}%` }} /></div>
}
function MathCard({ point }: { point: MathPoint }) {
  const open = useChildStore(s => s.openKp)
  const selected = useChildStore(s => s.kpDrawerId === point.id)
  const state = mathState(point)
  const tip = [point.title, state.label, ...state.skills.map(s => `${s.name}：${s.detail}`), point.last_at ? `最近作答 ${formatPinyinDate(point.last_at, true)}` : ''].filter(Boolean).join(' · ')
  return <button className={`math-card ${state.complete ? 'is-complete' : state.mastered ? 'is-partial' : ''}`} title={tip} aria-label={`${tip}；查看详情`} data-selected={selected} onClick={event => { event.currentTarget.focus(); open(point.id) }}>
    {point.module_code === 'shape' && <ShapeGlyph shape={SHAPES[point.title]} size={38} color="currentColor" />}
    <span className="math-glyph">{point.title}</span>
    {state.complete && <span className="math-check" aria-label="完全掌握">✓</span>}
    <span className="math-marks">{state.skills.map(s => <span key={s.code} className={s.lit ? 'lit' : 'unlit'} aria-label={`${s.name}：${s.lit ? '已掌握' : '未掌握'}`}>{s.short}</span>)}</span>
  </button>
}
export function MathMastery() {
  const childId = useChildStore(s => s.childId)
  const close = useChildStore(s => s.closeKp)
  useEffect(() => () => close(), [childId, close])
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['matrix', childId, 'math'],
    queryFn: () => api.get<Matrix>(`/children/${childId}/mastery/matrix?subject=math`), refetchInterval: 30_000, refetchIntervalInBackground: false, refetchOnWindowFocus: 'always' })
  const [search, setSearch] = useState('')
  const groups = useMemo(() => mathGroups(data?.modules, search), [data?.modules, search])
  const summary = useMemo(() => mathSummary(groups.flatMap(g => g.points)), [groups])
  const scrollTo = (shape: boolean) => {
    const target = document.getElementById(shape ? 'math-shape' : 'math-map')
    target?.focus({ preventScroll: true })
    target?.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth', block: 'start' })
  }
  return <div className="math-page">
    <header className="math-heading"><Link to="/">进度总览 / 算术</Link><h1>算术掌握</h1><p>从正式作答，看见加减运算与图形认识的进步</p></header>
    {isLoading ? <div className="dash-card math-empty" role="status">正在读取算术掌握情况…</div> : !data ? <div className="dash-card math-empty" role="alert">算术数据暂时无法读取。<button onClick={() => void refetch()}>重新加载</button></div> : <>
      {isError && <p role="alert" className="math-notice">刷新暂时失败，当前显示上次读取的数据。<button onClick={() => void refetch()}>重试</button></p>}
      <section className="dash-card math-summary" aria-label="算术四题型掌握进度"><div className="math-type-bars">{MATH_TYPES.map(type => <button key={type.code} onClick={() => scrollTo(type.code === 'find' || type.code === 'name')} aria-label={`查看${type.name}掌握地图`}>
        <span>{type.name}</span><MathBar counts={summary[type.code]} label={type.name} />{!summary[type.code].total && <small>暂无内容</small>}
      </button>)}</div><div className="math-legend"><span><i className="is-mastered" />已掌握</span><span><i className="is-answered" />已作答未掌握</span><span><i className="is-unattempted" />未作答</span></div></section>
      <p className="math-rule">加减法按算式计算、情境应用掌握；图形按听音找图形、看图认名称掌握。适用题型全部掌握才完全点亮。内容页示例试做不计入进度。</p>
      <div className="math-toolbar"><label className="math-search"><span aria-hidden="true">⌕</span><input aria-label="查找算式或图形" placeholder="查找算式或图形，如 2+3、圆形" value={search} onChange={e => setSearch(e.target.value)} />{search && <button aria-label="清空搜索" onClick={() => setSearch('')}>×</button>}</label></div>
      <section id="math-map" className="math-map" tabIndex={-1} aria-labelledby="math-map-title"><header className="math-map-heading"><h2 id="math-map-title">算术掌握地图</h2><p>亮起已掌握，暗色未掌握 · 加法按和、减法按被减数分组，后续单元不重复前一单元的算式</p></header>
        {!groups.length ? <p className="math-empty">暂无算术内容。</p> : !groups.some(g => g.visiblePoints.length) ? <p className="math-empty">没有匹配的算式或图形。</p> : groups.map(group => {
          const groupSummary = mathSummary(group.points)
          const types = group.code === 'shape' ? MATH_TYPES.slice(2) : MATH_TYPES.slice(0, 2)
          return <section key={group.code} id={`math-${group.code}`} tabIndex={-1} className="dash-card math-group" aria-label={group.name}>
            <header><h3>{group.name}</h3></header><div className="math-group-bars">{types.map(type => <div key={type.code}><span>{type.name}</span><MathBar counts={groupSummary[type.code]} label={`${group.name} · ${type.name}`} /></div>)}</div>
            <div className="math-grid">{group.visiblePoints.map(point => <MathCard key={point.id} point={point} />)}</div>
            {!group.visiblePoints.length && <p className="math-detail-note">{search ? '这个单元没有匹配项。' : '这个单元暂无内容。'}</p>}
          </section>
        })}
      </section>
    </>}
  </div>
}
