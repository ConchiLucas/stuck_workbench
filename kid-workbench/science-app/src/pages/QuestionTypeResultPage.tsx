import { Link, Navigate, useParams } from 'react-router-dom'
import { CloseIcon } from '../components/Icons'
import { findQuestionType, questionsFor } from '../data/questionTypes'
import { quizToPractice } from '../data/quizAdapter'
import { useDemoAnswerStore } from '../store/demoAnswerStore'
import { useLiveQuizStore } from '../store/liveQuizStore'

export function QuestionTypeResultPage() {
  const { slug } = useParams()
  const type = findQuestionType(slug)
  const live = slug === 'choice'
  const liveQuestions = useLiveQuizStore((s) => s.questions)
  const fallback = useLiveQuizStore((s) => s.fallback)
  const invalidate = useLiveQuizStore((s) => s.invalidate)
  const localBank = questionsFor(slug)
  const waiting = live && !fallback && liveQuestions.length === 0
  const bank = waiting ? [] : live && liveQuestions.length && !fallback ? liveQuestions.map(quizToPractice) : localBank
  const solved = useDemoAnswerStore((s) => s.solved)
  const clearType = useDemoAnswerStore((s) => s.clearType)
  if (!type || !slug) return <Navigate to="/" replace />
  if (waiting) return <Navigate to={`/question-types/${slug}`} replace />
  if (bank.length === 0) return <Navigate to="/" replace />

  const rows = bank.map((question, index) => {
    const key = `${slug}:${question.id}`
    const marked = solved[key]
    const skipped = marked === undefined
    return {
      n: index + 1,
      prompt: question.prompt,
      skipped,
      correct: marked === true,
    }
  })
  const correctCount = rows.filter((row) => row.correct).length

  return (
    <section className="result-page result-list-page page-enter" aria-label="答题结果">
      <Link className="result-exit" to="/" aria-label="退出"><CloseIcon /></Link>
      <p className="eyebrow">{type.title}</p>
      <h1>答题结果</h1>
      <p className="result-count">答对 {correctCount} / {bank.length} 题</p>
      <ol className="result-list">
        {rows.map((row) => (
          <li key={row.n} className={row.skipped ? 'is-skipped' : row.correct ? 'is-correct' : 'is-wrong'}>
            <span className="result-seq">第 {row.n} 题</span>
            <strong>{row.prompt}</strong>
            <b>{row.skipped ? '未作答' : row.correct ? '答对' : '答错'}</b>
          </li>
        ))}
      </ol>
      <div className="result-actions">
        <Link className="primary-button" to={`/question-types/${slug}`} onClick={() => {
          clearType(slug)
          if (live) invalidate()
        }}>再练一次</Link>
        <Link className="secondary-button" to="/">回到首页</Link>
      </div>
    </section>
  )
}
