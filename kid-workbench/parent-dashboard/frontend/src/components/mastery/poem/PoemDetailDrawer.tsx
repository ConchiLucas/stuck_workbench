import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import type { KpDetail } from '../../../api/types'
import { PoemHistoryQuestion } from '../../question/PoemHistoryQuestion'
import { formatPinyinDate } from '../../../lib/pinyinMastery'
import { accuracyLabel } from '../../../lib/literacyMastery'
import { POEM_TYPES, poemState } from '../../../lib/poemMastery'
import '../phrase/phrase.css'

export function PoemDetailDrawer({ data, isError, close, retry }: {
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
  const state = poemState(data ?? {})
  const history = [...(data?.history ?? [])].sort((a, b) => b.at.localeCompare(a.at))
  return createPortal(<div className="phrase-modal">
    <div className="phrase-scrim" onClick={close} aria-hidden="true" />
    <aside ref={panel} className="phrase-drawer" role="dialog" aria-modal="true" aria-labelledby="poem-detail-title">
      <header><span>古诗{data && ` · ${data.module_name}`}</span><button onClick={close} aria-label="关闭古诗详情">×</button></header>
      {!data ? <div className="phrase-empty" role={isError ? 'alert' : 'status'}><h2 id="poem-detail-title">{isError ? '古诗详情暂时无法读取' : '正在读取古诗详情…'}</h2>{isError && <button onClick={retry}>重新加载</button>}</div> : <>
        {isError && <p role="alert" className="phrase-notice">刷新失败，当前显示上次读取的详情。<button onClick={retry}>重试</button></p>}
        <div className="phrase-detail-hero"><h2 id="poem-detail-title">{data.title}</h2><div><span className={`phrase-status ${state.complete ? 'lit' : ''}`}>{state.label}</span><p>按当时保存的题目和作答查看。答对一道选诗名不等于整首诗已掌握。演示账本不能当作真实学习结果。</p></div></div>
        <section><h3>题型掌握</h3><div className="phrase-detail-skills">{state.skills.map(skill => <div key={skill.code} className="phrase-detail-skill">
          <div><span className={`phrase-detail-mark ${skill.lit ? 'lit' : 'unlit'}`}>{skill.short}</span><strong>{skill.name}</strong><span className={`phrase-status ${skill.lit ? 'lit' : ''}`}>{skill.detail}</span></div>
          <p>作答 {skill.attempts} 次<span>正确率 {accuracyLabel(skill.accuracy, skill.attempts)}</span></p>
          {skill.status === 'review_due' && <small>已掌握，待复习</small>}
        </div>)}</div><p className="phrase-detail-note">完全掌握需要选诗名、补字、选下一句、排顺序都达到已掌握。根据正式作答自动更新，示例试做不计入。</p>
        </section>
        <section><h3>作答记录</h3>{!history.length ? <p className="phrase-detail-note">还没有作答记录。</p> : <ul className="phrase-history">{history.map((item, index) => <li key={`${item.at}-${index}`}>
          <time dateTime={item.at}>{formatPinyinDate(item.at, true)}</time>
          <span>{item.source === 'parent_mark' ? '历史手动标记' : POEM_TYPES.find(skill => skill.code === item.skill_code)?.name ?? '题型未记录'}</span>
          <strong className={item.is_correct ? 'answer-correct' : 'answer-wrong'}>{item.source === 'parent_mark' ? '已标记' : item.is_correct ? '答对' : '答错'}</strong>
          {item.source !== 'parent_mark' ? <PoemHistoryQuestion review={item.poem_review} correct={item.is_correct} /> : null}
        </li>)}</ul>}</section>
      </>}
    </aside>
  </div>, document.body)
}
