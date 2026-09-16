import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../api/client'
import type { Matrix, MatrixPoint, MatrixSkill } from '../../../api/types'
import { useChildStore } from '../../../store/childStore'
import { barWidths } from '../../../lib/pinyinMastery'
import './english.css'

const ENGLISH_TYPES = [
  { code: 'listen', name: '听音选词', short: '听' },
  { code: 'picture', name: '看图选词', short: '图' },
  { code: 'build', name: '组句子', short: '组' },
  { code: 'type', name: '写单词', short: '写' },
  { code: 'read', name: '读一读', short: '读' },
] as const

type EnglishCounts = { mastered: number; answered: number; unattempted: number; total: number }
type EnglishPoint = MatrixPoint & { module_code: string }

function englishState(point: { skills?: MatrixSkill[] | null }) {
  const skills = ENGLISH_TYPES.map((type) => {
    const raw = point.skills?.find((item) => item.code === type.code)
    const lit = raw?.status === 'mastered' || raw?.status === 'review_due'
    const attempts = raw?.attempts ?? 0
    const detail = raw?.status === 'review_due' ? '待复习' : lit ? '已掌握' : raw?.status === 'shaky' ? '待巩固' : attempts ? '练习中' : '未练'
    return { ...type, status: raw?.status ?? 'not_started', attempts, lit, detail }
  })
  const complete = skills.every((skill) => skill.lit)
  const started = skills.some((skill) => skill.attempts > 0 || skill.status !== 'not_started')
  return { skills, complete, started, label: complete ? '五项均已掌握' : started ? '尚未完全掌握' : '未练' }
}

function summaryOf(points: EnglishPoint[]) {
  const skills = points.flatMap((point) => englishState(point).skills)
  return Object.fromEntries(ENGLISH_TYPES.map(({ code }) => {
    const applicable = skills.filter((skill) => skill.code === code)
    return [code, {
      mastered: applicable.filter((skill) => skill.lit).length,
      answered: applicable.filter((skill) => !skill.lit && skill.attempts > 0).length,
      unattempted: applicable.filter((skill) => !skill.lit && skill.attempts === 0).length,
      total: applicable.length,
    }]
  })) as Record<string, EnglishCounts>
}

function EnglishBar({ counts, label }: { counts: EnglishCounts; label: string }) {
  const widths = barWidths(counts)
  const tip = `${label}：已掌握 ${counts.mastered} · 已作答未掌握 ${counts.answered} · 未作答 ${counts.unattempted}（共 ${counts.total}）`
  return <div className="english-bar" role="img" title={tip} aria-label={tip}><span className="is-mastered" style={{ width: `${widths.mastered}%` }} /><span className="is-answered" style={{ width: `${widths.answered}%` }} /></div>
}

function EnglishCard({ point }: { point: EnglishPoint }) {
  const open = useChildStore((s) => s.openKp)
  const selected = useChildStore((s) => s.kpDrawerId === point.id)
  const state = englishState(point)
  const tip = [point.title, state.label, ...state.skills.map((skill) => `${skill.name}：${skill.detail}`)].join(' · ')
  return <button type="button" className={`english-card ${state.complete ? 'is-complete' : state.started ? 'is-partial' : ''}`} title={tip} aria-label={`${tip}；查看详情`} data-selected={selected} onClick={(event) => { event.currentTarget.focus(); open(point.id) }}>
    <span className="english-glyph">{point.title}</span>
    {state.complete && <span className="english-check" aria-label="完全掌握">✓</span>}
    <span className="english-marks">{state.skills.map((skill) => <span key={skill.code} className={skill.lit ? 'lit' : 'unlit'} aria-label={`${skill.name}：${skill.detail}`}>{skill.short}</span>)}</span>
  </button>
}

export function EnglishMastery() {
  const childId = useChildStore((s) => s.childId)
  const close = useChildStore((s) => s.closeKp)
  useEffect(() => () => close(), [childId, close])
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['matrix', childId, 'english'],
    queryFn: () => api.get<Matrix>(`/children/${childId}/mastery/matrix?subject=english`),
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
  const summary = useMemo(() => summaryOf(groups.flatMap((group) => group.points)), [groups])
  return <div className="english-page">
    <header className="english-heading"><Link to="/">进度总览 / 英语</Link><h1>英语掌握</h1><p>听音选词、看图选词、组句子、写单词、读一读五项均掌握后完全点亮。演示账本不能当作真实学习结果。</p></header>
    {isLoading ? <div className="dash-card english-empty" role="status">正在读取英语掌握情况…</div> : !data ? <div className="dash-card english-empty" role="alert">英语数据暂时无法读取。<button type="button" onClick={() => void refetch()}>重新加载</button></div> : <>
      {isError && <p role="alert" className="english-notice">刷新暂时失败，当前显示上次读取的数据。<button type="button" onClick={() => void refetch()}>重试</button></p>}
      <section className="dash-card english-summary" aria-label="英语五题型掌握进度">
        <div className="english-type-bars">{ENGLISH_TYPES.map((type) => <div key={type.code}><span>{type.name}</span><EnglishBar counts={summary[type.code]} label={type.name} /></div>)}</div>
        <div className="english-legend"><span><i className="is-mastered" />已掌握</span><span><i className="is-answered" />已作答未掌握</span><span><i className="is-unattempted" />未作答</span></div>
      </section>
      <p className="english-rule">按日、周、月的练习和新掌握看总览与任务历史；这里按单词看五项能力。打开卡片可查看当时保存的原题。试听不会记入作答。</p>
      <div className="english-toolbar"><label className="english-search"><span aria-hidden="true">⌕</span><input aria-label="查找单词" placeholder="查找单词，如 apple" value={search} onChange={(e) => setSearch(e.target.value)} />{search && <button type="button" aria-label="清空搜索" onClick={() => setSearch('')}>×</button>}</label></div>
      <section className="english-map" aria-labelledby="english-map-title">
        <header className="english-map-heading"><h2 id="english-map-title">英语掌握地图</h2><p>亮起已掌握，暗色未掌握 · 不是识字绿格子，打开单词看当时题目</p></header>
        {!groups.length ? <p className="english-empty">暂无英语内容。</p> : !groups.some((group) => group.visiblePoints.length) ? <p className="english-empty">没有匹配的单词。</p> : groups.map((group) => (
          <section key={group.code} className="dash-card english-group" aria-label={group.name}>
            <header><h3>{group.name}</h3></header>
            <div className="english-grid">{group.visiblePoints.map((point) => <EnglishCard key={point.id} point={point} />)}</div>
            {!group.visiblePoints.length && <p className="english-detail-note">{search ? '这个单元没有匹配项。' : '这个单元暂无内容。'}</p>}
          </section>
        ))}
      </section>
    </>}
  </div>
}
