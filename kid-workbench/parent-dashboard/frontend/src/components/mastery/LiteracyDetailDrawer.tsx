import { LiteracyHistoryQuestion } from '../question/LiteracyHistoryQuestion'
import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import type { KpDetail } from '../../api/types'
import { accuracyLabel, literacyState } from '../../lib/literacyMastery'
import './literacy.css'

export function LiteracyDetailDrawer({ data, isError, close, retry }: {
  data?: KpDetail; isError: boolean; close: () => void; retry: () => void
}) {
  const panel = useRef<HTMLElement>(null)
  const closeRef = useRef(close)
  useEffect(() => { closeRef.current = close }, [close])
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null
    const oldOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    panel.current?.querySelector<HTMLButtonElement>('button')?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.preventDefault(); closeRef.current(); return }
      if (e.key !== 'Tab') return
      const items = Array.from(panel.current?.querySelectorAll<HTMLElement>('button:not([disabled]), a[href], input, summary, [tabindex="0"]') ?? []).filter(item => {
        if (item.closest('[hidden]')) return false
        for (let parent = item.parentElement; parent && parent !== panel.current; parent = parent.parentElement) {
          if (parent.tagName === 'DETAILS' && !parent.hasAttribute('open') && !(item.tagName === 'SUMMARY' && item.parentElement === parent)) return false
        }
        return true
      })
      const first = items[0], last = items[items.length - 1]
      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last?.focus() }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first?.focus() }
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.body.style.overflow = oldOverflow
      document.removeEventListener('keydown', onKey)
      previous?.focus()
    }
  }, [])
  const state = literacyState(data ?? {})
  const history = [...(data?.history ?? [])].sort((a, b) => b.at.localeCompare(a.at))
  return createPortal(<div className="literacy-modal">
    <div className="literacy-scrim" onClick={close} aria-hidden="true" />
    <aside ref={panel} className="literacy-drawer" role="dialog" aria-modal="true" aria-labelledby="literacy-detail-title">
      <header><span>识字{data && ` · ${data.module_name}`}</span><button onClick={close} aria-label="关闭字详情">×</button></header>
      {!data ? <div className="literacy-empty" role={isError ? 'alert' : 'status'}><h2 id="literacy-detail-title">{isError ? '字详情暂时无法读取' : '正在读取字详情…'}</h2>{isError && <button onClick={retry}>重新加载</button>}</div> : <>
        <div className="literacy-detail-hero"><h2 id="literacy-detail-title">{data.title}</h2><div><span className={`literacy-status ${state.complete ? 'lit' : ''}`}>{state.label}</span><p>{state.complete ? '义、字、写三项均已掌握' : `已掌握 ${state.mastered} / 3 项题型`}</p></div></div>
        <section><h3>题型掌握</h3><div className="literacy-detail-skills">{state.skills.map(skill => <div key={skill.code} className="literacy-detail-skill">
          <div><span className={`literacy-detail-mark ${skill.lit ? 'lit' : 'unlit'}`}>{skill.short}</span><strong>{skill.name}</strong><span className={`literacy-status ${skill.lit ? 'lit' : ''}`}>{skill.detail}</span></div>
          <p>作答 {skill.attempts} 次<span>正确率 {accuracyLabel(skill.accuracy, skill.attempts)}</span></p>
          {skill.status === 'review_due' && <small>已掌握，待复习</small>}
        </div>)}</div><p className="literacy-detail-note">根据各题型作答结果自动更新掌握情况。</p></section>
        <section><h3>作答记录</h3>{!history.length ? <p className="literacy-detail-note">这个字还没有作答记录。</p> : <ul className="literacy-history">{history.map((item, index) => <li key={`${item.at}-${index}`}>
          <time dateTime={item.at}>{formatDate(item.at, true)}</time>
          <span>{item.source === 'parent_mark' ? '历史手动标记' : state.skills.find(skill => skill.code === item.skill_code)?.name ?? '题型未记录'}</span>
          <strong className={item.is_correct ? 'answer-correct' : 'answer-wrong'}>{item.is_correct ? '答对' : '答错'}</strong>
          {item.skill_code === 'glyph_sense' && <LiteracyHistoryQuestion review={item.literacy_review} correct={item.is_correct} />}
        </li>)}</ul>}</section>
      </>}
    </aside>
  </div>, document.body)
}

function formatDate(value: string, withTime = false) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '日期未记录'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    ...(withTime ? { hour: '2-digit', minute: '2-digit' } as const : {}),
  }).format(date)
}
