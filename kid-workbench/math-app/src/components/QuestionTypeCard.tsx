import { appPath } from '../appPath'
import { Link } from 'react-router-dom'
import type { QuestionTypeCard as QuestionTypeCardData } from '../content/questionTypePrototype'

interface Props {
  card: QuestionTypeCardData
  onOpen?: () => void
}

export function QuestionTypeCard({ card, onOpen }: Props) {
  const body = <>
    <img src={appPath(`/cards/${card.id}.svg`)} alt="" />
    <strong>{card.title}</strong>
  </>
  if (card.href) return <Link className="question-type-card" data-testid="question-type-card" to={card.href} aria-label={card.title} onClick={onOpen}>{body}</Link>
  return <article className="question-type-card" data-testid="question-type-card">{body}</article>
}
