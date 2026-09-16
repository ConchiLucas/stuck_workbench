import { Link, Navigate, useParams } from 'react-router-dom'
import { quizToPractice } from '../content/quizAdapter'
import {
  isPracticeType,
  practiceHref,
  typePracticeBanks,
  typeTitles,
} from '../content/typePracticeBanks'
import { useLiveQuizStore } from '../store/liveQuizStore'
import { useTypePracticeStore } from '../store/typePracticeStore'

export function TypeResultPage() {
  const { type } = useParams()
  const picks = useTypePracticeStore((s) => s.picks)
  const clearType = useTypePracticeStore((s) => s.clearType)
  const liveQuestions = useLiveQuizStore((s) => (isPracticeType(type) ? s.questionsByType[type] : undefined))
  const fallback = useLiveQuizStore((s) => isPracticeType(type) && Boolean(s.fallback[type]))
  const invalidate = useLiveQuizStore((s) => s.invalidate)
  if (!isPracticeType(type)) return <Navigate to="/" replace />
  if (!fallback && !liveQuestions?.length) return <Navigate to={practiceHref(type, 1)} replace />
  const bank = fallback || !liveQuestions?.length ? typePracticeBanks[type] : liveQuestions.map(quizToPractice)
  const rows = bank.map((question, index) => {
    const n = index + 1
    const picked = picks[`${type}:${n}`]
    const skipped = picked === undefined
    return {
      n,
      prompt: question.prompt,
      pickedLabel: skipped ? '未选' : question.options[picked],
      answerLabel: question.options[question.answerIndex],
      skipped,
      correct: !skipped && picked === question.answerIndex,
    }
  })
  const correctCount = rows.filter((row) => row.correct).length

  return (
    <section className="type-result-page" aria-label="答题结果">
      <Link className="icon-button type-result-close" to="/" aria-label="退出练习">×</Link>
      <p className="eyebrow">{typeTitles[type]}</p>
      <h1>答题结果</h1>
      <p className="result-count">答对 {correctCount} / {bank.length} 题</p>
      <ol className="type-result-list">
        {rows.map((row) => (
          <li key={row.n} className={row.skipped ? 'is-skipped' : row.correct ? 'is-correct' : 'is-wrong'}>
            <span>第 {row.n} 题</span>
            <strong>{row.prompt}</strong>
            <span>你选了 {row.pickedLabel}</span>
            <span>正确答案 {row.answerLabel}</span>
            <b>{row.skipped ? '未作答' : row.correct ? '答对' : '答错'}</b>
          </li>
        ))}
      </ol>
      <div className="result-actions">
        <Link className="primary-button" to={practiceHref(type, 1)} onClick={() => {
          clearType(type)
          invalidate(type)
        }}>再练一次</Link>
        <Link className="secondary-button" to="/">回到首页</Link>
      </div>
    </section>
  )
}
