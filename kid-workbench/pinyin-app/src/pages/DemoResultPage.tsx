import { X } from '@phosphor-icons/react'
import { useEffect } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { optionLabel, questionPrompt } from '../store/demoAnswerStore'
import { sessionKey, useDemoQuizStore } from '../store/demoQuizStore'
import { useChildStore } from '../store/childStore'
import { demoHref, isDemoType, typeTitle } from './demoBank'

export function DemoResultPage() {
  const { type } = useParams()
  const childId = useChildStore(s => s.childId)
  const ensure = useDemoQuizStore(s => s.ensure)
  const restart = useDemoQuizStore(s => s.restart)
  const session = useDemoQuizStore(s => isDemoType(type) ? s.sessions[sessionKey(childId, type)] : undefined)

  useEffect(() => {
    if (isDemoType(type)) void ensure(childId, type)
  }, [childId, type, ensure])

  if (!isDemoType(type)) return <Navigate to="/" replace />
  if (!session || session.status !== 'ready' || !session.verified) {
    return <section className="result-page result-list-page page-enter" role="status">
      <p>{session?.status === 'error' ? session.error : '正在核对答题结果…'}</p>
      {session?.status === 'error' ? <button type="button" className="primary-button" onClick={() => { void ensure(childId, type) }}>再试一次</button> : null}
    </section>
  }

  const rows = session.entries.map(({ question, result, pending, unavailable }, index) => ({
    n: index + 1,
    prompt: questionPrompt(question),
    pickedLabel: result ? optionLabel(question, result.selectedOptionId) : pending ? optionLabel(question, pending.optionId) : '未选',
    answerLabel: result ? optionLabel(question, result.answerOptionId) : undefined,
    pending: Boolean(pending && !unavailable),
    skipped: !result,
    correct: result?.correct === true,
  }))
  const correctCount = rows.filter(row => row.correct).length

  return (
    <section className="result-page result-list-page page-enter">
      <Link className="top-icon top-icon-close result-exit" to="/" aria-label="退出练习"><span aria-hidden="true"><X weight="bold" /></span></Link>
      <p className="eyebrow">{typeTitle(type)}</p>
      <h1>答题结果</h1>
      <p className="result-count">答对 {correctCount} / {rows.length} 题</p>
      <ol className="result-list">
        {rows.map(row => (
          <li key={row.n} className={row.skipped ? 'is-skipped' : row.correct ? 'is-correct' : 'is-wrong'}>
            <span className="result-seq">第 {row.n} 题</span><strong>{row.prompt}</strong>
            <span>你选了 {row.pickedLabel}</span>
            {row.answerLabel ? <span>正确答案 {row.answerLabel}</span> : null}
            <b>{row.pending ? '待确认' : row.skipped ? '未作答' : row.correct ? '答对' : '答错'}</b>
            {row.pending ? <Link to={demoHref(type, row.n)}>继续确认本题</Link> : null}
          </li>
        ))}
      </ol>
      <div className="result-actions">
        <Link className="primary-button" to={demoHref(type, 1)} onClick={() => restart(childId, type)}>再练一次</Link>
        <Link className="secondary-button" to="/">回到首页</Link>
      </div>
    </section>
  )
}
