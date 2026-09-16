import { useState } from 'react'
import { MatchLab } from './MatchLab'
import { SequenceLab } from './SequenceLab'
import { LabelLab } from './LabelLab'
import { ScienceGlyph } from './icons'
import { errorFacts, isCorrect, scienceTitles, type ScienceAnswer, type ScienceExample } from './types'

export type { ScienceAnswer, ScienceExample, ScienceKind } from './types'
export { errorFacts, isCorrect, scienceKinds, scienceTitles } from './types'
export { ScienceGlyph, PlantDiagram } from './icons'

function Feedback({ title, children, tone = 'success' }: { title: string; children?: string; tone?: 'success' | 'again' }) {
  return <div className={`demo-feedback ${tone}`} role="status"><b>{title}</b><span>{children}</span></div>
}

function ChoiceLab({
  example,
  readOnly,
  initialId,
  onSelect,
  showFeedback,
}: {
  example: ScienceExample
  readOnly?: boolean
  initialId?: string
  onSelect?: (id: string) => void
  showFeedback?: boolean
}) {
  const [picked, setPicked] = useState(initialId ?? '')
  const options = example.options ?? []
  const choose = (id: string) => {
    if (readOnly || picked) return
    setPicked(id)
    onSelect?.(id)
  }
  const right = picked === example.answerId
  return <>
    <div className="demo-options">
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
          aria-label={`选择：${option.label}`}
          disabled={readOnly || Boolean(picked)}
          className={`${picked === option.id ? 'picked' : ''} ${picked && option.id === example.answerId ? 'right' : ''}`}
          onClick={() => choose(option.id)}
        >{option.icon ? <ScienceGlyph name={option.icon} /> : null}{option.label}</button>
      ))}
    </div>
    {showFeedback && picked ? <Feedback title={right ? '找对了！' : '再观察一下'} tone={right ? 'success' : 'again'}>{right ? example.explanation : example.tip}</Feedback> : null}
  </>
}

export function SciencePlayer({
  example,
  readOnly = false,
  initialAnswer,
  onAnswer,
  onSolved,
  onMark,
  resolveAssetUrl = (url) => url,
  showPrompt = true,
  showFeedback = true,
}: {
  example: ScienceExample
  readOnly?: boolean
  initialAnswer?: ScienceAnswer
  onAnswer?: (answer: ScienceAnswer) => void
  onSolved?: () => void
  onMark?: (correct: boolean) => void
  resolveAssetUrl?: (url: string) => string
  showPrompt?: boolean
  showFeedback?: boolean
}) {
  const mark = (correct: boolean) => onMark?.(correct)
  const visual = example.icon || example.visual
  return (
    <div className={`science-player type-${example.kind}${readOnly ? ' is-readonly' : ''}`}>
      {showPrompt ? <h1 className="canvas-prompt">{example.prompt}</h1> : null}
      {example.kind === 'choice' && visual ? <div className="scene scene-observe" data-testid="question-visual" aria-hidden="true">{example.icon ? <ScienceGlyph name={example.icon} /> : <span>{example.visual}</span>}</div> : null}
      {example.kind === 'choice' ? (
        <ChoiceLab example={example} readOnly={readOnly} initialId={initialAnswer?.selectedId} showFeedback={showFeedback} onSelect={(id) => {
          const answer = { kind: 'choice' as const, selectedId: id }
          onAnswer?.(answer)
          mark(isCorrect(example, answer))
          if (id === example.answerId) onSolved?.()
        }} />
      ) : null}
      {example.kind === 'match' ? (
        <MatchLab example={example} readOnly={readOnly} initialPairs={initialAnswer?.pairs} resolveAssetUrl={resolveAssetUrl} showFeedback={showFeedback} onChange={(pairs) => onAnswer?.({ kind: 'match', pairs })} onMark={mark} onSolved={onSolved} />
      ) : null}
      {example.kind === 'sequence' ? (
        <SequenceLab example={example} readOnly={readOnly} initialOrder={initialAnswer?.sequence} showFeedback={showFeedback} onChange={(sequence) => onAnswer?.({ kind: 'sequence', sequence })} onMark={mark} onSolved={onSolved} />
      ) : null}
      {example.kind === 'label' ? (
        <LabelLab example={example} readOnly={readOnly} initialLabels={initialAnswer?.labels} showFeedback={showFeedback} onChange={(labels) => onAnswer?.({ kind: 'label', labels })} onMark={mark} onSolved={onSolved} />
      ) : null}
    </div>
  )
}

export function ScienceFacts({ example, answer }: { example: ScienceExample; answer?: ScienceAnswer }) {
  const facts = errorFacts(example, answer)
  if (!facts.length && !answer) return <p>未保存当时的具体作答。</p>
  return <ul className="science-facts">{facts.map((fact) => <li key={fact}>{fact}</li>)}</ul>
}

export { scienceTitles as scienceTypeTitles }
