import { Link } from 'react-router-dom'
import { additionEquationType } from '../content/questionTypePrototype'
import { useQuestionTypePracticeStore } from '../store/questionTypePracticeStore'

export function QuestionTypeResultPage() {
  const completed = useQuestionTypePracticeStore((state) => state.completed)
  const score = useQuestionTypePracticeStore((state) => state.score)
  const start = useQuestionTypePracticeStore((state) => state.start)

  if (!completed) return <section className="page prototype-result-page"><article className="result-sheet prototype-empty-result"><p className="eyebrow">QUESTION TYPE 01</p><h1>先完成一次练习</h1><p>完成 5 道模拟题后，这里会显示本次结果。</p><Link className="primary-button" to="/types/addition-equation">查看题型详情</Link></article></section>

  return <section className="page prototype-result-page"><article className="result-sheet">
    <p className="eyebrow">PRACTICE COMPLETE</p><h1>这一组完成啦</h1><p className="result-message">{additionEquationType.mode}</p>
    <div className="prototype-score"><strong>{score}</strong><span>/ {additionEquationType.questions.length}</span></div>
    <h2>答对 {score} / {additionEquationType.questions.length} 题</h2><p>这是前端模拟结果，还没有写入学习进度。</p>
    <div className="result-actions"><Link className="primary-button" to="/types/addition-equation/practice" onClick={start}>再练一次</Link><Link className="secondary-button" to="/types/addition-equation">回到题型详情</Link></div>
  </article></section>
}
