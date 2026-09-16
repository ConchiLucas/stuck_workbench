import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import type { KpDetail } from '../../../api/types'
import { PhraseHistoryQuestion } from '../../question/PhraseHistoryQuestion'
import { formatPinyinDate } from '../../../lib/pinyinMastery'
import { accuracyLabel } from '../../../lib/literacyMastery'
import { PHRASE_TYPES, phraseState } from '../../../lib/phraseMastery'
import './phrase.css'

export function PhraseDetailDrawer({ data, isError, close, retry }: {
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
      const items = Array.from(panel.current?.querySelectorAll<HTMLElement>('button:not([disabled]), a[href], input, [tabindex="0"]') ?? [])
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
  const state = phraseState(data ?? {})
  const history = [...(data?.history ?? [])].sort((a, b) => b.at.localeCompare(a.at))
  return createPortal(<div className="phrase-modal">
    <div className="phrase-scrim" onClick={close} aria-hidden="true" />
    <aside ref={panel} className="phrase-drawer" role="dialog" aria-modal="true" aria-labelledby="phrase-detail-title">
      <header><span>英语短句{data && ` · ${data.module_name}`}</span><button onClick={close} aria-label="关闭短句详情">×</button></header>
      {!data ? <div className="phrase-empty" role={isError ? 'alert' : 'status'}><h2 id="phrase-detail-title">{isError ? '短句详情暂时无法读取' : '正在读取短句详情…'}</h2>{isError && <button onClick={retry}>重新加载</button>}</div> : <>
        {isError && <p role="alert" className="phrase-notice">刷新失败，当前显示上次读取的详情。<button onClick={retry}>重试</button></p>}
        <div className="phrase-detail-hero"><h2 id="phrase-detail-title">{data.title}</h2><div><span className={`phrase-status ${state.complete ? 'lit' : ''}`}>{state.label}</span><p>听、选、场景三项均掌握后完全点亮。问与答有记录时单独列出。演示账本不能当作真实学习结果。</p></div></div>
        <section><h3>题型掌握</h3><div className="phrase-detail-skills">{state.skills.map(skill => <div key={skill.code} className="phrase-detail-skill">
          <div><span className={`phrase-detail-mark ${skill.lit ? 'lit' : 'unlit'}`}>{skill.short}</span><strong>{skill.name}</strong><span className={`phrase-status ${skill.lit ? 'lit' : ''}`}>{skill.detail}</span></div>
          <p>作答 {skill.attempts} 次<span>正确率 {accuracyLabel(skill.accuracy, skill.attempts)}</span></p>
          {skill.status === 'review_due' && <small>已掌握，待复习</small>}
        </div>)}</div><p className="phrase-detail-note">根据正式作答自动更新掌握情况，示例试做不计入。</p>
        </section>
        <section><h3>作答记录</h3>{!history.length ? <p className="phrase-detail-note">还没有作答记录。</p> : <ul className="phrase-history">{history.map((item, index) => <li key={`${item.at}-${index}`}>
          <time dateTime={item.at}>{formatPinyinDate(item.at, true)}</time>
          <span>{item.source === 'parent_mark' ? '历史手动标记' : PHRASE_TYPES.find(skill => skill.code === item.skill_code)?.name ?? '题型未记录'}</span>
          <strong className={item.is_correct ? 'answer-correct' : 'answer-wrong'}>{item.source === 'parent_mark' ? '已标记' : item.is_correct ? '答对' : '答错'}</strong>
          {item.source !== 'parent_mark' ? <PhraseHistoryQuestion review={item.phrase_review} correct={item.is_correct} /> : null}
        </li>)}</ul>}</section>
      </>}
    </aside>
  </div>, document.body)
}
