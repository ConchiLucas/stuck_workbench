import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import type { KpDetail } from '../../../api/types'
import { EnglishHistoryQuestion } from '../../question/EnglishHistoryQuestion'
import { formatPinyinDate } from '../../../lib/pinyinMastery'
import { accuracyLabel } from '../../../lib/literacyMastery'
import './english.css'

const ENGLISH_TYPES = [
  { code: 'listen', name: '听音选词', short: '听' },
  { code: 'picture', name: '看图选词', short: '图' },
  { code: 'build', name: '组句子', short: '组' },
  { code: 'type', name: '写单词', short: '写' },
  { code: 'read', name: '读一读', short: '读' },
]

function englishState(data: Partial<KpDetail>) {
  const skills = ENGLISH_TYPES.map((type) => {
    const skill = data.skills?.find((item) => item.code === type.code)
    const status = skill?.status ?? 'not_started'
    const attempts = skill?.attempts ?? 0
    const lit = status === 'mastered' || status === 'review_due'
    return { ...type, status, attempts, accuracy: skill?.accuracy ?? 0, lit, detail: status === 'mastered' ? '已掌握' : status === 'review_due' ? '待复习' : status === 'shaky' ? '待巩固' : attempts ? '练习中' : '未练' }
  })
  const complete = skills.every((skill) => skill.lit)
  return { skills, complete, label: complete ? '五项均已掌握' : '尚未完全掌握' }
}

export function EnglishDetailDrawer({ data, isError, close, retry }: {
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
  const state = englishState(data ?? {})
  const history = [...(data?.history ?? [])].sort((a, b) => b.at.localeCompare(a.at))
  return createPortal(<div className="english-modal">
    <div className="english-scrim" onClick={close} aria-hidden="true" />
    <aside ref={panel} className="english-drawer" role="dialog" aria-modal="true" aria-labelledby="english-detail-title">
      <header><span>英语{data && ` · ${data.module_name}`}</span><button onClick={close} aria-label="关闭英语详情">×</button></header>
      {!data ? <div className="english-empty" role={isError ? 'alert' : 'status'}><h2 id="english-detail-title">{isError ? '英语详情暂时无法读取' : '正在读取英语详情…'}</h2>{isError && <button onClick={retry}>重新加载</button>}</div> : <>
        {isError && <p role="alert" className="english-notice">刷新失败，当前显示上次读取的详情。<button onClick={retry}>重试</button></p>}
        <div className="english-detail-hero"><h2 id="english-detail-title">{data.title}</h2><div><span className={`english-status ${state.complete ? 'lit' : ''}`}>{state.label}</span><p>听、图、组、写、读五项均掌握后完全点亮。演示账本不能当作真实学习结果。</p></div></div>
        <section><h3>题型掌握</h3><div className="english-detail-skills">{state.skills.map(skill => <div key={skill.code} className="english-detail-skill">
          <div><span className={`english-detail-mark ${skill.lit ? 'lit' : 'unlit'}`}>{skill.short}</span><strong>{skill.name}</strong><span className={`english-status ${skill.lit ? 'lit' : ''}`}>{skill.detail}</span></div>
          <p>作答 {skill.attempts} 次<span>正确率 {accuracyLabel(skill.accuracy, skill.attempts)}</span></p>
          {skill.status === 'review_due' && <small>已掌握，待复习</small>}
        </div>)}</div><p className="english-detail-note">根据正式作答自动更新掌握情况，示例试做不计入。</p>
        </section>
        <section><h3>作答记录</h3>{!history.length ? <p className="english-detail-note">还没有作答记录。</p> : <ul className="english-history">{history.map((item, index) => <li key={`${item.at}-${index}`}>
          <time dateTime={item.at}>{formatPinyinDate(item.at, true)}</time>
          <span>{item.source === 'parent_mark' ? '历史手动标记' : ENGLISH_TYPES.find(skill => skill.code === item.skill_code)?.name ?? '题型未记录'}</span>
          <strong className={item.is_correct ? 'answer-correct' : 'answer-wrong'}>{item.source === 'parent_mark' ? '已标记' : item.is_correct ? '答对' : '答错'}</strong>
          {item.source !== 'parent_mark' ? <EnglishHistoryQuestion review={item.english_review} correct={item.is_correct} /> : null}
        </li>)}</ul>}</section>
      </>}
    </aside>
  </div>, document.body)
}
