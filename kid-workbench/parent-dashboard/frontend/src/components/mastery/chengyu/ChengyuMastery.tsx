import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../api/client'
import type { Matrix } from '../../../api/types'
import { useChildStore } from '../../../store/childStore'
import { barWidths } from '../../../lib/pinyinMastery'
import { CHENGYU_TYPES, chengyuState, chengyuSummary, type ChengyuCounts, type ChengyuPoint } from '../../../lib/chengyuMastery'
import './chengyu.css'

function ChengyuBar({ counts, label }: { counts: ChengyuCounts; label: string }) {
  const widths = barWidths(counts)
  const tip = `${label}：已掌握 ${counts.mastered} · 已作答未掌握 ${counts.answered} · 未作答 ${counts.unattempted}（共 ${counts.total}）`
  return <div className="chengyu-bar" role="img" title={tip} aria-label={tip}><span className="is-mastered" style={{ width: `${widths.mastered}%` }} /><span className="is-answered" style={{ width: `${widths.answered}%` }} /></div>
}

function ChengyuCard({ point }: { point: ChengyuPoint }) {
  const open = useChildStore((s) => s.openKp)
  const selected = useChildStore((s) => s.kpDrawerId === point.id)
  const state = chengyuState(point)
  const tip = [point.title, state.label, ...state.skills.map((skill) => `${skill.name}：${skill.detail}`)].join(' · ')
  return <button type="button" className={`chengyu-card ${state.complete ? 'is-complete' : state.started ? 'is-partial' : ''}`} title={tip} aria-label={`${tip}；查看详情`} data-selected={selected} onClick={(event) => { event.currentTarget.focus(); open(point.id) }}>
    <span className="chengyu-glyph">{point.title}</span>
    {state.complete && <span className="chengyu-check" aria-label="完全掌握">✓</span>}
    <span className="chengyu-marks">{state.skills.map((skill) => <span key={skill.code} className={skill.lit ? 'lit' : 'unlit'} aria-label={`${skill.name}：${skill.detail}`}>{skill.short}</span>)}</span>
  </button>
}

export function ChengyuMastery() {
  const childId = useChildStore((s) => s.childId)
  const close = useChildStore((s) => s.closeKp)
  useEffect(() => () => close(), [childId, close])
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['matrix', childId, 'chengyu'],
    queryFn: () => api.get<Matrix>(`/children/${childId}/mastery/matrix?subject=chengyu`),
    refetchInterval: 30_000, refetchIntervalInBackground: false, refetchOnWindowFocus: 'always',
  })
  const [search, setSearch] = useState('')
  const groups = useMemo(() => {
    const query = search.trim().toLowerCase()
    return (data?.modules ?? []).map((module) => {
      const points = (module.points ?? []).map((point) => ({ ...point, module_code: module.code }))
      return { ...module, points, visiblePoints: points.filter((point) => !query || point.title.toLowerCase().includes(query)) }
    })
  }, [data?.modules, search])
  const summary = useMemo(() => chengyuSummary(groups.flatMap((group) => group.points)), [groups])
  return <div className="chengyu-page">
    <header className="chengyu-heading"><Link to="/">进度总览 / 成语</Link><h1>成语掌握</h1><p>听释义、选成语、看拼音、看句子四项均掌握后完全点亮。答对一道不等于整个成语已掌握。演示账本不能当作真实学习结果。</p></header>
    {isLoading ? <div className="dash-card chengyu-empty" role="status">正在读取成语掌握情况…</div> : !data ? <div className="dash-card chengyu-empty" role="alert">成语数据暂时无法读取。<button type="button" onClick={() => void refetch()}>重新加载</button></div> : <>
      {isError && <p role="alert" className="chengyu-notice">刷新暂时失败，当前显示上次读取的数据。<button type="button" onClick={() => void refetch()}>重试</button></p>}
      <section className="dash-card chengyu-summary" aria-label="成语题型掌握进度">
        <div className="chengyu-type-bars">{CHENGYU_TYPES.map((type) => <div key={type.code}><span>{type.name}</span><ChengyuBar counts={summary[type.code]} label={type.name} /></div>)}</div>
        <div className="chengyu-legend"><span><i className="is-mastered" />已掌握</span><span><i className="is-answered" />已作答未掌握</span><span><i className="is-unattempted" />未作答</span></div>
      </section>
      <p className="chengyu-rule">按日、周、月的练习和新掌握看总览与任务历史；这里按成语看能力。打开卡片可查看当时保存的原题。试听不会记入作答。</p>
      <div className="chengyu-toolbar"><label className="chengyu-search"><span aria-hidden="true">⌕</span><input aria-label="查找成语" placeholder="查找成语，如 一心一意" value={search} onChange={(e) => setSearch(e.target.value)} />{search && <button type="button" aria-label="清空搜索" onClick={() => setSearch('')}>×</button>}</label></div>
      <section className="chengyu-map" aria-labelledby="chengyu-map-title">
        <header className="chengyu-map-heading"><h2 id="chengyu-map-title">成语掌握地图</h2><p>亮起已掌握，暗色未掌握 · 打开成语看当时题目</p></header>
        {!groups.length ? <p className="chengyu-empty">暂无成语内容。</p> : !groups.some((group) => group.visiblePoints.length) ? <p className="chengyu-empty">没有匹配的成语。</p> : groups.map((group) => (
          <section key={group.code} className="dash-card chengyu-group" aria-label={group.name}>
            <header><h3>{group.name}</h3></header>
            <div className="chengyu-grid">{group.visiblePoints.map((point) => <ChengyuCard key={point.id} point={point} />)}</div>
            {!group.visiblePoints.length && <p className="chengyu-detail-note">{search ? '这个单元没有匹配项。' : '这个单元暂无内容。'}</p>}
          </section>
        ))}
      </section>
    </>}
  </div>
}
