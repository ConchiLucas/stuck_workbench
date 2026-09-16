import { useEffect, useRef } from 'react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { LogicPlayer, isCorrect, type LogicAnswer } from '@kid-workbench/logic-player'
import { fallbackExamples, quizToExample } from '../content/quizAdapter'
import {
  isPracticeType,
  practiceHref,
  practiceResultHref,
  typeTitles,
} from '../content/typePracticeBanks'
import { useLiveQuizStore } from '../store/liveQuizStore'
import { useTypePracticeStore } from '../store/typePracticeStore'
import '@kid-workbench/logic-player/player.css'

const ADVANCE_MS = 450

export function TypePracticePage() {
  const { type, n } = useParams()
  const navigate = useNavigate()
  const setPick = useTypePracticeStore((s) => s.setPick)
  const setSequence = useTypePracticeStore((s) => s.setSequence)
  const current = n ? Number(n) : 1
  const picked = useTypePracticeStore((s) => (isPracticeType(type) ? s.pick(type, current) : undefined))
  const sequence = useTypePracticeStore((s) => (isPracticeType(type) ? s.sequence(type, current) : []))
  const rejected = useTypePracticeStore((s) => (isPracticeType(type) ? s.rejectedTaps(type, current) : []))
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
          <div className="progress-track" role="progressbar" aria-label={`${typeTitles[quizType]}，${current} / 4`} aria-valuemin={1} aria-valuemax={4} aria-valuenow={current}>
            <i style={{ width: `${(current / 4) * 100}%` }} />
            <span aria-hidden="true">{current} / 4</span>
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
  const bank = fallback || !liveQuestions?.length ? fallbackExamples(quizType) : liveQuestions.map(quizToExample)
  if (!Number.isInteger(current) || current < 1 || current > bank.length) {
    return <Navigate to={practiceHref(quizType, 1)} replace />
  }
  const question = bank[current - 1]
  const total = bank.length
  const prevHref = current > 1 ? practiceHref(quizType, current - 1) : undefined
  const nextHref = current < total ? practiceHref(quizType, current + 1) : practiceResultHref(quizType)
  const nextLabel = current < total ? '下一题' : '查看结果'
  const initial: LogicAnswer | undefined = question.kind === 'order'
    ? (sequence.length || rejected.length ? { kind: 'order', sequence, rejected } : undefined)
    : (picked ? { kind: question.kind, selectedId: picked } : undefined)

  function scheduleAdvance() {
    const gen = ++pickGen.current
    window.clearTimeout(advanceTimer.current)
    advanceTimer.current = window.setTimeout(() => {
      if (pickGen.current === gen) navigate(nextHref)
    }, ADVANCE_MS)
  }

  return (
    <section className="type-practice-page">
      <nav className="type-practice-top" aria-label="练习导航">
        <Link className="icon-button" to="/" aria-label="退出练习">×</Link>
        <div className="progress-track" role="progressbar" aria-label={`${typeTitles[quizType]}，${current} / ${total}`} aria-valuemin={1} aria-valuemax={total} aria-valuenow={current}>
          <i style={{ width: `${(current / total) * 100}%` }} />
          <span aria-hidden="true">{current} / {total}</span>
        </div>
        <div className="type-practice-nav">
          {prevHref ? <Link className="icon-button" to={prevHref} aria-label="上一题">‹</Link> : <span className="icon-button is-muted" aria-label="上一题" aria-disabled="true">‹</span>}
          <Link className="icon-button" to={nextHref} aria-label={nextLabel}>›</Link>
        </div>
      </nav>
      <LogicPlayer
        key={`${quizType}-${question.prompt}-${current}`}
        example={question}
        initialAnswer={initial}
        onAnswer={(answer) => {
          if (answer.kind === 'order') {
            setSequence(quizType, current, answer.sequence ?? [], answer.rejected)
            if (isCorrect(question, answer)) scheduleAdvance()
            return
          }
          if (answer.selectedId) {
            setPick(quizType, current, answer.selectedId)
            if (answer.selectedId === question.answerId) scheduleAdvance()
          }
        }}
      />
    </section>
  )
}
