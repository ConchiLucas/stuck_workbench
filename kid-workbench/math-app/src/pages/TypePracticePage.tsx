import { useEffect, useRef } from 'react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { quizToPractice, isRedundantVisual } from '../content/quizAdapter'
import {
  isPracticeType,
  practiceHref,
  practiceResultHref,
  typePracticeBanks,
  typeTitles,
} from '../content/typePracticeBanks'
import { useLiveQuizStore } from '../store/liveQuizStore'
import { useTypePracticeStore } from '../store/typePracticeStore'

const ADVANCE_MS = 450

export function TypePracticePage() {
  const { type, n } = useParams()
  const navigate = useNavigate()
  const setPick = useTypePracticeStore((s) => s.setPick)
  const picks = useTypePracticeStore((s) => s.picks)
  const current = n ? Number(n) : 1
  const picked = useTypePracticeStore((s) => (isPracticeType(type) ? s.pick(type, current) : undefined))
  const pickGen = useRef(0)
  const advanceTimer = useRef(0)
  const ensure = useLiveQuizStore((s) => s.ensure)
  const invalidate = useLiveQuizStore((s) => s.invalidate)
  const useFallback = useLiveQuizStore((s) => s.useFallback)
  const liveQuestions = useLiveQuizStore((s) => (isPracticeType(type) ? s.questionsByType[type] : undefined))
  const loading = useLiveQuizStore((s) => isPracticeType(type) && s.loadingType === type)
  const error = useLiveQuizStore((s) => s.error)
  const fallback = useLiveQuizStore((s) => isPracticeType(type) && Boolean(s.fallback[type]))

  useEffect(() => () => window.clearTimeout(advanceTimer.current), [])
  useEffect(() => {
    if (isPracticeType(type)) void ensure(type)
  }, [type, ensure])

  if (!isPracticeType(type)) return <Navigate to="/" replace />
  const quizType = type
  const waiting = !fallback && !liveQuestions?.length
  if (waiting) {
    return (
      <section className="type-practice-page">
        <nav className="type-practice-top" aria-label="练习导航">
          <Link className="icon-button" to="/" aria-label="退出练习">×</Link>
          <div className="progress-track" role="progressbar" aria-label={`${typeTitles[quizType]}，0 / 4`} aria-valuemin={0} aria-valuemax={4} aria-valuenow={0}>
            <i style={{ width: '0%' }} />
            <span className="progress-label" aria-hidden="true">0 / 4</span>
          </div>
          <div className="type-practice-nav">
            <span className="icon-button is-muted" aria-label="上一题" aria-disabled="true">‹</span>
            <span className="icon-button is-muted" aria-label="下一题" aria-disabled="true">›</span>
          </div>
        </nav>
        <section className="question-workspace" aria-label="当前题目">
          <div className="quiz-status">
            <p>{loading || !error ? '出题中…' : error}</p>
            {error && !loading ? (
              <div className="quiz-status-actions">
                <button type="button" className="primary-button" onClick={() => { invalidate(quizType); void ensure(quizType) }}>再试一次</button>
                <button type="button" className="secondary-button" onClick={() => useFallback(quizType)}>用示例题</button>
              </div>
            ) : null}
          </div>
        </section>
      </section>
    )
  }
  const bank = fallback || !liveQuestions?.length ? typePracticeBanks[quizType] : liveQuestions.map(quizToPractice)
  if (!Number.isInteger(current) || current < 1 || current > bank.length) {
    return <Navigate to={practiceHref(quizType, 1)} replace />
  }
  const question = bank[current - 1]
  const done = bank.filter((_, index) => picks[`${quizType}:${index + 1}`] !== undefined).length
  const prevHref = current > 1 ? practiceHref(quizType, current - 1) : undefined
  const nextHref = current < bank.length ? practiceHref(quizType, current + 1) : practiceResultHref(quizType)
  const two = question.options.length === 2

  function choose(index: number) {
    const gen = ++pickGen.current
    setPick(quizType, current, index)
    window.clearTimeout(advanceTimer.current)
    advanceTimer.current = window.setTimeout(() => {
      if (pickGen.current === gen) navigate(nextHref)
    }, ADVANCE_MS)
  }

  return (
    <section className="type-practice-page">
      <nav className="type-practice-top" aria-label="练习导航">
        <Link className="icon-button" to="/" aria-label="退出练习">×</Link>
        <div className="progress-track" role="progressbar" aria-label={`${typeTitles[quizType]}，${done} / ${bank.length}`} aria-valuemin={0} aria-valuemax={bank.length} aria-valuenow={done}>
          <i style={{ width: `${(done / bank.length) * 100}%` }} />
          <span className="progress-label" aria-hidden="true">{done} / {bank.length}</span>
        </div>
        <div className="type-practice-nav">
          {prevHref ? <Link className="icon-button" to={prevHref} aria-label="上一题">‹</Link> : <span className="icon-button is-muted" aria-label="上一题" aria-disabled="true">‹</span>}
          <Link className="icon-button" to={nextHref} aria-label="下一题">›</Link>
        </div>
      </nav>
      <section className="question-workspace" aria-label="当前题目">
        <p className="eyebrow">{typeTitles[quizType]}</p>
        <h1>{question.prompt}</h1>
        {isRedundantVisual(question.prompt, question.visual) ? null : <p className="type-practice-visual" aria-hidden="true">{question.visual}</p>}
        <div className={`option-grid${two ? ' is-two' : ''}`}>
          {question.options.map((option, index) => (
            <button
              key={`${question.id}-${option}-${index}`}
              className={`option-button${picked === index ? ' is-picked' : ''}`}
              onClick={() => choose(index)}
            >
              {option}
            </button>
          ))}
        </div>
      </section>
    </section>
  )
}
