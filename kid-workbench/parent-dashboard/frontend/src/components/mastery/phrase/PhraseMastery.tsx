import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../api/client'
import type { Matrix } from '../../../api/types'
import { useChildStore } from '../../../store/childStore'
import { barWidths } from '../../../lib/pinyinMastery'
import { PHRASE_TYPES, phraseState, phraseSummary, type PhraseCounts, type PhrasePoint } from '../../../lib/phraseMastery'
import './phrase.css'

function PhraseBar({ counts, label }: { counts: PhraseCounts; label: string }) {
  const widths = barWidths(counts)
  const tip = `${label}：已掌握 ${counts.mastered} · 已作答未掌握 ${counts.answered} · 未作答 ${counts.unattempted}（共 ${counts.total}）`
  return <div className="phrase-bar" role="img" title={tip} aria-label={tip}><span className="is-mastered" style={{ width: `${widths.mastered}%` }} /><span className="is-answered" style={{ width: `${widths.answered}%` }} /></div>
}

function PhraseCard({ point }: { point: PhrasePoint }) {
  const open = useChildStore((s) => s.openKp)
  const selected = useChildStore((s) => s.kpDrawerId === point.id)
  const state = phraseState(point)
  const tip = [point.title, state.label, ...state.skills.map((skill) => `${skill.name}：${skill.detail}`)].join(' · ')
  return <button type="button" className={`phrase-card ${state.complete ? 'is-complete' : state.started ? 'is-partial' : ''}`} title={tip} aria-label={`${tip}；查看详情`} data-selected={selected} onClick={(event) => { event.currentTarget.focus(); open(point.id) }}>
    <span className="phrase-glyph">{point.title}</span>
    {state.complete && <span className="phrase-check" aria-label="完全掌握">✓</span>}
    <span className="phrase-marks">{state.skills.map((skill) => <span key={skill.code} className={skill.lit ? 'lit' : 'unlit'} aria-label={`${skill.name}：${skill.detail}`}>{skill.short}</span>)}</span>
  </button>
}

export function PhraseMastery() {
  const childId = useChildStore((s) => s.childId)
  const close = useChildStore((s) => s.closeKp)
  useEffect(() => () => close(), [childId, close])
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['matrix', childId, 'phrase'],
    queryFn: () => api.get<Matrix>(`/children/${childId}/mastery/matrix?subject=phrase`),
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
  const summary = useMemo(() => phraseSummary(groups.flatMap((group) => group.points)), [groups])
  return <div className="phrase-page">
    <header className="phrase-heading"><Link to="/">进度总览 / 英语短句</Link><h1>英语短句掌握</h1><p>听一听、选句子、什么时候说三项均掌握后完全点亮。问与答有记录时单独显示。演示账本不能当作真实学习结果。</p></header>
    {isLoading ? <div className="dash-card phrase-empty" role="status">正在读取短句掌握情况…</div> : !data ? <div className="dash-card phrase-empty" role="alert">短句数据暂时无法读取。<button type="button" onClick={() => void refetch()}>重新加载</button></div> : <>
      {isError && <p role="alert" className="phrase-notice">刷新暂时失败，当前显示上次读取的数据。<button type="button" onClick={() => void refetch()}>重试</button></p>}
      <section className="dash-card phrase-summary" aria-label="短句题型掌握进度">
        <div className="phrase-type-bars">{PHRASE_TYPES.map((type) => <div key={type.code}><span>{type.name}</span><PhraseBar counts={summary[type.code]} label={type.name} /></div>)}</div>
        <div className="phrase-legend"><span><i className="is-mastered" />已掌握</span><span><i className="is-answered" />已作答未掌握</span><span><i className="is-unattempted" />未作答</span></div>
      </section>
      <p className="phrase-rule">按日、周、月的练习和新掌握看总览与任务历史；这里按短句看能力。打开卡片可查看当时保存的原题。试听不会记入作答。</p>
      <div className="phrase-toolbar"><label className="phrase-search"><span aria-hidden="true">⌕</span><input aria-label="查找短句" placeholder="查找短句，如 Good morning" value={search} onChange={(e) => setSearch(e.target.value)} />{search && <button type="button" aria-label="清空搜索" onClick={() => setSearch('')}>×</button>}</label></div>
      <section className="phrase-map" aria-labelledby="phrase-map-title">
        <header className="phrase-map-heading"><h2 id="phrase-map-title">短句掌握地图</h2><p>亮起已掌握，暗色未掌握 · 打开短句看当时题目</p></header>
        {!groups.length ? <p className="phrase-empty">暂无短句内容。</p> : !groups.some((group) => group.visiblePoints.length) ? <p className="phrase-empty">没有匹配的短句。</p> : groups.map((group) => (
          <section key={group.code} className="dash-card phrase-group" aria-label={group.name}>
            <header><h3>{group.name}</h3></header>
            <div className="phrase-grid">{group.visiblePoints.map((point) => <PhraseCard key={point.id} point={point} />)}</div>
            {!group.visiblePoints.length && <p className="phrase-detail-note">{search ? '这个单元没有匹配项。' : '这个单元暂无内容。'}</p>}
          </section>
        ))}
      </section>
    </>}
  </div>
}
