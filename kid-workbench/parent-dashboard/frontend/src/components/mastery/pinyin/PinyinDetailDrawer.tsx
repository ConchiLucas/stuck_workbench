import { PinyinHistoryQuestion } from '../../question/PinyinHistoryQuestion'
import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import type { KpDetail } from '../../../api/types'
import { formatPinyinDate, pinyinState, PINYIN_TYPES } from '../../../lib/pinyinMastery'
import { accuracyLabel } from '../../../lib/literacyMastery'
import './pinyin.css'

export function PinyinDetailDrawer({ data, isError, close, retry }: {
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
      if (!panel.current?.contains(document.activeElement)) { e.preventDefault(); first?.focus() }
      else if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last?.focus() }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first?.focus() }
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.body.style.overflow = oldOverflow
      document.removeEventListener('keydown', onKey)
      if (previous?.isConnected) previous.focus()
    }
  }, [])
  const kind = data?.kind ?? (data?.module_code === 'syllables' ? 'syllable' : 'letter')
  const state = pinyinState({ ...data, kind })
  const history = [...(data?.history ?? [])].sort((a, b) => b.at.localeCompare(a.at))
  return createPortal(<div className="pinyin-modal">
    <div className="pinyin-scrim" onClick={close} aria-hidden="true" />
    <aside ref={panel} className="pinyin-drawer" role="dialog" aria-modal="true" aria-labelledby="pinyin-detail-title">
      <header><span>拼音{data && ` · ${data.module_name}`}</span><button onClick={close} aria-label="关闭拼音详情">×</button></header>
      {!data ? <div className="pinyin-empty" role={isError ? 'alert' : 'status'}><h2 id="pinyin-detail-title">{isError ? '拼音详情暂时无法读取' : '正在读取拼音详情…'}</h2>{isError && <button onClick={retry}>重新加载</button>}</div> : <>
        {isError && <p role="alert" className="pinyin-notice">刷新失败，当前显示上次读取的详情。<button onClick={retry}>重试</button></p>}
        <div className="pinyin-detail-hero"><h2 id="pinyin-detail-title">{data.syllable || data.title}</h2><div><span className={`pinyin-status ${state.complete ? 'lit' : ''}`}>{state.label}</span><p>{kind === 'syllable' ? [data.initial, data.final].filter(Boolean).join(' + ') : '听、找、认三项均掌握后完全点亮'}</p></div></div>
        <section><h3>题型掌握</h3><div className="pinyin-detail-skills">{state.skills.map(skill => <div key={skill.code} className="pinyin-detail-skill">
          <div><span className={`pinyin-detail-mark ${skill.lit ? 'lit' : 'unlit'}`}>{skill.short}</span><strong>{skill.name}</strong><span className={`pinyin-status ${skill.lit ? 'lit' : ''}`}>{skill.detail}</span></div>
          <p>作答 {skill.attempts} 次<span>正确率 {accuracyLabel(skill.accuracy, skill.attempts)}</span></p>
          {skill.status === 'review_due' && <small>已掌握，待复习</small>}
        </div>)}</div><p className="pinyin-detail-note">根据各题型作答结果自动更新掌握情况。</p>
        {data.mastered_at && <p className="pinyin-milestone">首次完全掌握 <time dateTime={data.mastered_at}>{formatPinyinDate(data.mastered_at)}</time></p>}</section>
        <section><h3>作答记录</h3>{!history.length ? <p className="pinyin-detail-note">还没有作答记录。</p> : <ul className="pinyin-history">{history.map((item, index) => <li key={`${item.at}-${index}`}>
          <time dateTime={item.at}>{formatPinyinDate(item.at, true)}</time>
          <span>{item.source === 'parent_mark' ? '历史手动标记' : PINYIN_TYPES.find(skill => skill.code === item.skill_code)?.name ?? '题型未记录'}</span>
          <strong className={item.is_correct ? 'answer-correct' : 'answer-wrong'}>{item.source === 'parent_mark' ? '已标记' : item.is_correct ? '答对' : '答错'}</strong>
          {item.source !== 'parent_mark' ? <PinyinHistoryQuestion review={item.pinyin_review} correct={item.is_correct} /> : null}
        </li>)}</ul>}</section>
      </>}
    </aside>
  </div>, document.body)
}
