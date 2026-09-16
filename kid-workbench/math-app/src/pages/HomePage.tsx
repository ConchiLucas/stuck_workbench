import { QuestionTypeCard } from '../components/QuestionTypeCard'
import { questionTypePacks } from '../content/questionTypePrototype'

export function HomePage() {
  return <section className="type-gallery-page" aria-label="题型">
    <div className="question-type-grid">{questionTypePacks.map((card) => (
      <QuestionTypeCard card={{...card,href:`/types/${card.questionIds[0]}`}} key={card.id}/>
    ))}</div>
  </section>
}
