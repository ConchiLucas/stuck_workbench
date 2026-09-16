import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { additionEquationType } from '../content/questionTypePrototype'
import { useQuestionTypePracticeStore } from '../store/questionTypePracticeStore'

const letters = ['A', 'B', 'C', 'D']

export function QuestionTypePracticePage() {
  const { index, answered, choose, next } = useQuestionTypePracticeStore()
  const [feedback, setFeedback] = useState('')
  const navigate = useNavigate()
  const question = additionEquationType.questions[index]
  const isLast = index === additionEquationType.questions.length - 1

  const select = (optionIndex: number) => {
    const result = choose(optionIndex)
    if (result === 'correct') setFeedback('答对了')
    if (result === 'retry') setFeedback('再想一想，你还有一次机会')
    if (result === 'revealed') setFeedback(`正确答案是 ${question.options[question.answerIndex]}`)
  }

  const advance = () => {
    if (!next()) return
    if (isLast) {
      navigate('/types/addition-equation/result')
      return
    }
    setFeedback('')
  }

  return <section className="page prototype-practice-page">
    <header className="practice-header"><Link className="icon-button" aria-label="退出练习" to="/types/addition-equation">×</Link><div className="prototype-progress" aria-label={`第 ${index + 1} 题，共 ${additionEquationType.questions.length} 题`}>{additionEquationType.questions.map((item, itemIndex) => <i className={itemIndex < index ? 'done' : itemIndex === index ? 'current' : ''} key={item.id} />)}</div><span className="practice-mode">看算式</span></header>
    <article className="question-sheet prototype-question-sheet">
      <div className="prototype-question-label"><span>第 {index + 1} / {additionEquationType.questions.length} 题</span><small>20 以内加法</small></div>
      <h1>{question.stem}</h1>
      <div className="equation prototype-equation" aria-hidden><span>{question.a}</span><i>+</i><span>{question.b}</span><b>= ?</b></div>
      <div className="option-grid prototype-options">{question.options.map((option, optionIndex) => <button key={option} className="option-button" disabled={answered} aria-label={`${letters[optionIndex]} ${option}`} onClick={() => select(optionIndex)}><span className="option-letter">{letters[optionIndex]}</span>{option}</button>)}</div>
      <p className={`feedback prototype-feedback ${feedback === '答对了' ? 'correct' : ''}`} aria-live="polite">{feedback}</p>
      {answered && <button className="primary-button prototype-next" onClick={advance}>{isLast ? '查看结果' : '下一题'} <span>→</span></button>}
    </article>
  </section>
}
