import { useState } from 'react'
import { LogicGlyph } from './glyphs'
import { errorFacts, isCorrect, logicTitles, objectById, type LogicAnswer, type LogicExample, type LogicKind, type LogicObject } from './types'

export type { LogicAnswer, LogicExample, LogicKind, LogicObject } from './types'
export { errorFacts, isCorrect, logicKinds, logicTitles } from './types'
export { LogicGlyph } from './glyphs'

function Face({ object, url, resolve }: { object: LogicObject; url?: string; resolve: (url: string) => string }) {
  if (url) return <img className="logic-glyph" src={resolve(url)} alt="" />
  return <LogicGlyph object={object} />
}

function Tile({
  object, url, selected, wrong, disabled, weight, onClick, resolve,
}: {
  object: LogicObject
  url?: string
  selected?: boolean
  wrong?: boolean
  disabled?: boolean
  weight?: number
  onClick: () => void
  resolve: (url: string) => string
}) {
  const caption = object.caption
  const showCaption = Boolean(caption && caption !== object.glyph)
  return (
    <button
      type="button"
      className={`option-button${selected ? ' is-picked' : ''}${wrong ? ' is-wrong' : ''}${showCaption ? ' has-caption' : ''}`}
      data-weight={weight ?? object.scale ?? 2}
      aria-label={showCaption ? `${object.glyph} ${caption}` : caption || object.glyph}
      disabled={disabled}
      onClick={onClick}
    >
      <span className="option-glyph" aria-hidden="true">
        <Face object={object} url={url} resolve={resolve} />
      </span>
      {showCaption ? <span className="option-caption">{caption}</span> : null}
    </button>
  )
}

export function LogicPlayer({
  example,
  readOnly = false,
  initialAnswer,
  onAnswer,
  onSolved,
  resolveAssetUrl = (url) => url,
  showPrompt = true,
  showFeedback = true,
  eyebrow,
}: {
  example: LogicExample
  readOnly?: boolean
  initialAnswer?: LogicAnswer
  onAnswer?: (answer: LogicAnswer) => void
  onSolved?: () => void
  resolveAssetUrl?: (url: string) => string
  showPrompt?: boolean
  showFeedback?: boolean
  eyebrow?: string
}) {
  const [picked, setPicked] = useState(initialAnswer?.selectedId ?? '')
  const [sequence, setSequence] = useState<string[]>(initialAnswer?.sequence ?? [])
  const [rejected, setRejected] = useState(initialAnswer?.rejected ?? [])
  const [wrongId, setWrongId] = useState('')
  const isOrder = example.kind === 'order'
  const expected = example.correctSequence ?? []
  const optionIds = isOrder ? (example.displayOrder ?? expected) : (example.options ?? [])
  const locked = readOnly || (isOrder ? sequence.length === expected.length : Boolean(picked))
  const current = isOrder
    ? { kind: example.kind, sequence, rejected }
    : { kind: example.kind, selectedId: picked }
  const right = isCorrect(example, current)
  const feedback = isOrder
    ? (sequence.length === expected.length ? (right ? '答对了' : '再试一次') : wrongId ? '再试一次' : '')
    : picked ? (right ? '答对了' : '再试一次') : ''
  const showSeq = (example.kind === 'pattern' || example.kind === 'shape_reason') && (example.sequence?.length ?? 0) > 0
  const gridClass = [
    'option-grid',
    example.kind === 'diff' ? 'is-diff' : '',
    example.kind === 'compare' ? 'is-compare' : '',
  ].filter(Boolean).join(' ')

  function choose(id: string) {
    if (locked) return
    setPicked(id)
    const answer: LogicAnswer = { kind: example.kind, selectedId: id }
    onAnswer?.(answer)
    if (id === example.answerId) onSolved?.()
  }

  function tapOrder(id: string) {
    if (locked || sequence.includes(id)) return
    const nextIndex = sequence.length
    if (id === expected[nextIndex]) {
      const next = [...sequence, id]
      setSequence(next)
      setWrongId('')
      const answer: LogicAnswer = { kind: 'order', sequence: next, rejected }
      onAnswer?.(answer)
      if (next.length === expected.length) onSolved?.()
      return
    }
    const nextRejected = [...rejected, { id, atIndex: nextIndex }]
    setRejected(nextRejected)
    setWrongId(id)
    onAnswer?.({ kind: 'order', sequence, rejected: nextRejected })
  }

  return (
    <div className={`logic-player type-${example.kind}${readOnly ? ' is-readonly' : ''}`}>
      <section className="question-workspace is-split" aria-label="当前题目">
        <div className="question-stem">
          {eyebrow || logicTitles[example.kind as LogicKind] ? <p className="eyebrow">{eyebrow ?? logicTitles[example.kind]}</p> : null}
          {showPrompt ? <h1>{example.prompt}</h1> : null}
          {example.kind === 'classify' ? <p className="stem-hint">点出不一样的那个</p> : null}
          {example.kind === 'diff' ? <p className="stem-hint">找出不一样的那个</p> : null}
          {showSeq ? (
            <p className="seq-row" aria-hidden="true">
              {(example.sequence ?? []).map((id, index) => {
                const object = objectById(example.objects, id)
                return object ? <span key={`${id}-${index}`}><Face object={object} url={example.imageUrls?.[id]} resolve={resolveAssetUrl} /></span> : null
              })}
              <span className="seq-hole">？</span>
            </p>
          ) : null}
          {isOrder ? (
            <ol className="order-slots">
              {expected.map((id, index) => {
                const filled = objectById(example.objects, sequence[index] ?? '')
                return (
                  <li key={`${id}-${index}`} aria-label={`第 ${index + 1} 位`}>
                    <b>{index + 1}</b>
                    <span>{filled ? filled.caption || filled.glyph : '？'}</span>
                  </li>
                )
              })}
            </ol>
          ) : null}
          {showFeedback && feedback ? <p className="kid-feedback" role="status">{feedback}</p> : null}
        </div>
        <div className="choice-row">
          <div className={gridClass}>
            {optionIds.map((id) => {
              const object = objectById(example.objects, id)
              if (!object) return null
              const selected = isOrder ? sequence.includes(id) : picked === id
              return (
                <Tile
                  key={id}
                  object={object}
                  url={example.imageUrls?.[id]}
                  selected={selected}
                  wrong={isOrder && wrongId === id}
                  disabled={locked || (isOrder && sequence.includes(id))}
                  weight={example.kind === 'compare' ? (object.scale || 2) : undefined}
                  onClick={() => (isOrder ? tapOrder(id) : choose(id))}
                  resolve={resolveAssetUrl}
                />
              )
            })}
          </div>
        </div>
      </section>
    </div>
  )
}

export function LogicFacts({ example, answer }: { example: LogicExample; answer?: LogicAnswer }) {
  const facts = errorFacts(example, answer)
  if (!facts.length) return null
  return <ul className="logic-facts">{facts.map((fact) => <li key={fact}>{fact}</li>)}</ul>
}
