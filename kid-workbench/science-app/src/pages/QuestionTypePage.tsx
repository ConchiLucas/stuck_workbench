import { useCallback, useEffect, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { SciencePlayer } from '@kid-workbench/science-player'
import { CloseIcon, NextIcon, PrevIcon } from '../components/Icons'
import { findQuestionType, questionPath, questionsFor, toScienceExample, type PracticeQuestion, type QuestionType } from '../data/questionTypes'
import { quizToPractice } from '../data/quizAdapter'
import { useDemoAnswerStore } from '../store/demoAnswerStore'
import { useLiveQuizStore } from '../store/liveQuizStore'
import { appPath } from '../appPath'
import '@kid-workbench/science-player/player.css'

export function QuestionTypePage() {
  const { slug, questionId } = useParams()
  const type = findQuestionType(slug)
  const live = slug === 'choice'
  const ensure = useLiveQuizStore((s) => s.ensure)
  const invalidate = useLiveQuizStore((s) => s.invalidate)
  const useFallback = useLiveQuizStore((s) => s.useFallback)
  const liveQuestions = useLiveQuizStore((s) => s.questions)
  const loading = useLiveQuizStore((s) => s.loading)
  const error = useLiveQuizStore((s) => s.error)
  const fallback = useLiveQuizStore((s) => s.fallback)
  useEffect(() => {
    if (live) void ensure()
  }, [live, ensure])
  const localBank = questionsFor(slug)
  const waiting = live && !fallback && liveQuestions.length === 0
  const bank = waiting ? [] : live && liveQuestions.length && !fallback ? liveQuestions.map(quizToPractice) : localBank
  if (!type) return <Navigate to="/" replace />
  if (waiting) {
    return (
      <article className={`question-type-page type-${type.slug} page-enter`}>
        <section className="fullscreen-question-stage" aria-label="答题">
          <div className="question-canvas" data-testid="question-canvas">
            <nav className="canvas-nav" aria-label="答题进度">
              <Link to="/" aria-label="退出"><CloseIcon /></Link>
              <div className="practice-progress" role="progressbar" aria-valuemin={1} aria-valuemax={4} aria-valuenow={1} aria-label="第 1 题 / 4 题">
                <div className="progress-track"><i style={{ width: '0%' }}><span className="progress-count" aria-hidden="true">0 / 4</span></i></div>
              </div>
              <div className="question-stepper">
                <button type="button" disabled aria-label="上一题"><PrevIcon /></button>
                <button type="button" disabled aria-label="下一题"><NextIcon /></button>
              </div>
            </nav>
            <section className="question-hero" aria-label="当前题目">
              <div className="quiz-status">
                <p>{loading || !error ? '出题中…' : error}</p>
                {error && !loading ? (
                  <div className="quiz-status-actions">
                    <button type="button" className="primary-button" onClick={() => { invalidate(); void ensure() }}>再试一次</button>
                    <button type="button" className="secondary-button" onClick={() => useFallback()}>用示例题</button>
                  </div>
                ) : null}
              </div>
            </section>
          </div>
        </section>
      </article>
    )
  }
  if (bank.length === 0) return <Navigate to="/" replace />
  const index = questionId ? bank.findIndex((item) => item.id === questionId) : 0
  if (questionId && index < 0) return <Navigate to={`/question-types/${slug}`} replace />
  const question = bank[index] ?? bank[0]
  return <QuestionTypeExperience key={question.id} type={type} question={question} index={index < 0 ? 0 : index} bank={bank} />
}

function QuestionTypeExperience({ type, question, index, bank }: { type: QuestionType; question: PracticeQuestion; index: number; bank: PracticeQuestion[] }) {
  const prev = bank[index - 1]
  const next = bank[index + 1]
  const [solved, setSolved] = useState(false)
  const markSolved = useCallback(() => setSolved(true), [])
  const mark = useDemoAnswerStore((s) => s.mark)
  const onMark = useCallback((correct: boolean) => {
    mark(`${type.slug}:${question.id}`, correct)
  }, [mark, question.id, type.slug])
  const seq = index + 1
  const total = bank.length
  const resultHref = `/question-types/${type.slug}/result`
  const example = toScienceExample(question)

  return <article className={`question-type-page type-${type.slug} page-enter`}>
    <section className="fullscreen-question-stage" aria-label="答题">
      <div className="question-canvas" data-testid="question-canvas">
        <nav className="canvas-nav" aria-label="答题进度">
          <Link to="/" aria-label="退出"><CloseIcon /></Link>
          <div className="practice-progress" role="progressbar" aria-valuemin={1} aria-valuemax={total} aria-valuenow={seq} aria-label={`第 ${seq} 题 / ${total} 题`}>
            <div className="progress-track"><i style={{ width: `${(seq / total) * 100}%` }}><span className="progress-count" aria-hidden="true">{seq} / {total}</span></i></div>
          </div>
          <div className="question-stepper">
            {prev
              ? <Link to={questionPath(type.slug, prev, bank)} aria-label="上一题"><PrevIcon /></Link>
              : <button type="button" disabled aria-label="上一题"><PrevIcon /></button>}
            {next
              ? <Link to={questionPath(type.slug, next, bank)} aria-label="下一题"><NextIcon /></Link>
              : <Link to={resultHref} aria-label="下一题"><NextIcon /></Link>}
          </div>
        </nav>
        <div className="question-hero">
          <h1 className="canvas-prompt">{question.prompt}</h1>
        </div>
        <div className="canvas-question">
          <SciencePlayer example={example} showPrompt={false} onMark={onMark} onSolved={markSolved} resolveAssetUrl={(url) => appPath(url)} />
          {solved && !next && <Link className="next-question" to={resultHref}>查看结果</Link>}
        </div>
      </div>
    </section>
  </article>
}
