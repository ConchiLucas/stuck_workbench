import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../api/client'
import type { Matrix } from '../../../api/types'
import { useChildStore } from '../../../store/childStore'
import { formatPinyinDate, pinyinGroups, pinyinSummary, type PinyinSkillCode, type PinyinView } from '../../../lib/pinyinMastery'
import { PinyinTypeSummary } from './PinyinTypeSummary'
import { PinyinLetterMap } from './PinyinLetterMap'
import { PinyinSyllableMap } from './PinyinSyllableMap'
import './pinyin.css'

export function PinyinMastery() {
  const childId = useChildStore(s => s.childId)
  const closeKp = useChildStore(s => s.closeKp)
  useEffect(() => () => closeKp(), [childId, closeKp])
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['matrix', childId, 'pinyin'],
    queryFn: () => api.get<Matrix>(`/children/${childId}/mastery/matrix?subject=pinyin`),
    refetchInterval: 30_000, refetchIntervalInBackground: false, refetchOnWindowFocus: 'always' })
  const [search, setSearch] = useState('')
  const [view, setView] = useState<PinyinView>('all')
  const groups = useMemo(() => pinyinGroups(data?.modules, search), [data?.modules, search])
  const summary = useMemo(() => pinyinSummary([...groups.letters, ...groups.syllables].flatMap(g => g.points)), [groups])
  const select = (code: PinyinSkillCode) => {
    if (code !== 'blend') setView(code)
    const target = document.getElementById(code === 'blend' ? 'pinyin-syllable-map' : 'pinyin-letter-map')
    target?.focus({ preventScroll: true })
    target?.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth', block: 'start' })
  }
  return <div className="pinyin-page">
    <header className="pinyin-heading"><Link to="/">进度总览 / 拼音</Link><h1>拼音掌握</h1><p>从答题结果，看见字母认读与音节拼读的积累</p></header>
    {isLoading ? <div className="dash-card pinyin-empty" role="status">正在读取拼音掌握情况…</div> : !data ? <div className="dash-card pinyin-empty" role="alert">拼音数据暂时无法读取。<button onClick={() => void refetch()}>重新加载</button></div> : <>
      {isError && <p className="pinyin-notice" role="alert">刷新暂时失败，当前显示上次读取的数据。<button onClick={() => void refetch()}>重试</button></p>}
      <PinyinTypeSummary summary={summary} select={select} catalogAvailable={data.catalog_available !== false} />
      <p className="pinyin-rule">字母按听、找、认三项掌握，音节按声韵拼读掌握。{data.rule_effective_at && <>规则自 {formatPinyinDate(data.rule_effective_at)} 生效，之前的日历记录保留原口径。</>}</p>
      <div className="pinyin-toolbar"><label className="pinyin-search"><span aria-hidden="true">⌕</span><input aria-label="查找字母或音节" placeholder="查找字母或音节，如 bǎ、ba、lü" value={search} onChange={e => setSearch(e.target.value)} />{search && <button aria-label="清空搜索" onClick={() => setSearch('')}>×</button>}</label><span>支持无调拼写，v / u: 可输入 ü</span></div>
      <PinyinLetterMap groups={groups.letters} view={view} setView={setView} />
      <PinyinSyllableMap groups={groups.syllables} available={data.catalog_available !== false} />
    </>}
  </div>
}
