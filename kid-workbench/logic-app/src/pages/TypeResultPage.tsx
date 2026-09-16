import { Link, Navigate, useParams } from 'react-router-dom'
import { isCorrect } from '@kid-workbench/logic-player'
import { fallbackExamples, quizToExample } from '../content/quizAdapter'
import {
  isPracticeType,
  practiceHref,
  typeTitles,
} from '../content/typePracticeBanks'
import { useLiveQuizStore } from '../store/liveQuizStore'
import { useTypePracticeStore } from '../store/typePracticeStore'

export function TypeResultPage() {
  const { type } = useParams()
  const picks = useTypePracticeStore((s) => s.picks)
  const sequences = useTypePracticeStore((s) => s.sequences)
  const clearType = useTypePracticeStore((s) => s.clearType)
  const liveQuestions = useLiveQuizStore((s) => (isPracticeType(type) ? s.questionsByType[type] : undefined))
  const fallback = useLiveQuizStore((s) => isPracticeType(type) && Boolean(s.fallback[type]))
  const invalidate = useLiveQuizStore((s) => s.invalidate)
  if (!isPracticeType(type)) return <Navigate to="/" replace />
  if (!fallback && !liveQuestions?.length) return <Navigate to={practiceHref(type, 1)} replace />
  const bank = fallback || !liveQuestions?.length ? fallbackExamples(type) : liveQuestions.map(quizToExample)
  const rows = bank.map((question, index) => {
    const n = index + 1
    const key = `${type}:${n}`
    const picked = picks[key]
    const sequence = sequences[key] ?? []
    if (question.kind === 'order') {
      const skipped = sequence.length === 0
      const correct = isCorrect(question, { kind: 'order', sequence })
      return {
        n,
        prompt: question.prompt,
        pickedLabel: skipped ? '未选' : sequence.map((id) => question.objects.find((o) => o.id === id)?.caption ?? id).join(' → '),
        skipped,
        correct,
      }
    }
    const skipped = picked === undefined
    const label = question.objects.find((o) => o.id === picked)?.caption ?? picked
    return {
      n,
      prompt: question.prompt,
      pickedLabel: skipped ? '未选' : (label || '未选'),
      skipped,
      correct: !skipped && isCorrect(question, { kind: question.kind, selectedId: picked }),
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
