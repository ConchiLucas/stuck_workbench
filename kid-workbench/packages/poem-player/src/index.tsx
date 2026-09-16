import { useEffect, useRef, useState } from 'react'
import { errorFacts, isCorrect, poemTitles, type PoemAnswer, type PoemExample, type PoemKind } from './types'

export type { PoemAnswer, PoemExample, PoemKind } from './types'
export { errorFacts, isCorrect, poemKinds, poemTitles, displayFillLine, exampleFromClue } from './types'

function PlayButton({ url, missing }: { url?: string; missing?: string; readOnly?: boolean }) {
  const [playing, setPlaying] = useState(false)
  const [failed, setFailed] = useState(false)
  const audio = useRef<HTMLAudioElement | null>(null)
  useEffect(() => () => { audio.current?.pause() }, [])
  if (!url) {
    return <p className="poem-audio-missing" role="status">{missing || '读音素材暂不可用'}</p>
  }
  if (failed) {
    return <p className="poem-audio-missing" role="status">读音素材暂不可用</p>
  }
  function play() {
    if (!url) return
    if (!audio.current) audio.current = new Audio(url)
    audio.current.src = url
    audio.current.onended = () => setPlaying(false)
    audio.current.onerror = () => { setFailed(true); setPlaying(false) }
    void audio.current.play().then(() => setPlaying(true)).catch(() => setFailed(true))
  }
  return (
    <button type="button" className={`play-btn${playing ? ' is-playing' : ''}`} aria-label="朗读" aria-pressed={playing} onClick={play}>
      <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor" aria-hidden="true"><path d="M8 5v14l11-7z" /></svg>
    </button>
  )
}

function ChoiceLab({
  example, readOnly, initialId, onSelect, showFeedback,
}: {
  example: PoemExample
  readOnly?: boolean
  initialId?: string
  onSelect?: (id: string) => void
  showFeedback?: boolean
}) {
  const [picked, setPicked] = useState(initialId ?? '')
  const choose = (id: string) => {
    if (readOnly) return
    if (picked && picked === example.answerId) return
    setPicked(id)
    onSelect?.(id)
  }
  const right = picked === example.answerId
  return (
    <>
      <div className={`option-grid is-lines cols-${Math.min(example.options?.length ?? 0, 4)}`}>
        {(example.options ?? []).map((option) => {
          const selected = picked === option.id
          const isRight = picked && option.id === example.answerId
          const isWrong = picked === option.id && option.id !== example.answerId
          return (
            <button
              key={option.id}
              type="button"
              aria-label={option.label}
              aria-pressed={selected}
              disabled={readOnly || (Boolean(picked) && picked === example.answerId)}
              className={`option-card${selected ? ' is-picked' : ''}${isRight ? ' is-right' : ''}${isWrong ? ' is-wrong' : ''}`}
              onClick={() => choose(option.id)}
            >
              <strong>{option.label}</strong>
            </button>
          )
        })}
      </div>
      {showFeedback && picked ? <p className="kid-feedback" role="status">{right ? '答对了' : '再试一次'}</p> : null}
    </>
  )
}

function ReciteLab({
  example, readOnly, initial, onChange, showFeedback,
}: {
  example: PoemExample
  readOnly?: boolean
  initial?: string[]
  onChange?: (sequence: string[]) => void
  showFeedback?: boolean
}) {
  const slots = example.correctSequence ?? example.sequenceItems?.map((n) => n.id) ?? []
  const [sequence, setSequence] = useState<string[]>(initial ?? [])
  const done = sequence.length === slots.length && slots.length > 0
  const right = isCorrect(example, { kind: 'recite', sequence })
  const labelOf = (id: string) => example.sequenceItems?.find((n) => n.id === id)?.label || id
  function tap(id: string) {
    if (readOnly || done || sequence.includes(id)) return
    const next = [...sequence, id]
    setSequence(next)
    onChange?.(next)
  }
  return (
    <>
      <ol className="order-slots" data-count={slots.length}>
        {slots.map((id, index) => {
          const placed = sequence[index]
          const slotWrong = done && placed && placed !== (example.correctSequence ?? [])[index]
          return (
            <li key={id} aria-label={`第 ${index + 1} 句`} className={slotWrong ? 'is-wrong' : done && placed ? 'is-right' : ''}>
              <b>{index + 1}</b>
              <span>{placed ? labelOf(placed) : '？'}</span>
            </li>
          )
        })}
      </ol>
      <div className="option-grid is-lines cols-2">
        {(example.sequenceDisplayOrder ?? example.sequenceItems?.map((n) => n.id) ?? []).map((id) => {
          const selected = sequence.includes(id)
          return (
            <button
              key={id}
              type="button"
              aria-label={labelOf(id)}
              aria-pressed={selected}
              disabled={readOnly || selected || done}
              className={`option-card${selected ? ' is-picked' : ''}${done && right && selected ? ' is-right' : ''}`}
              onClick={() => tap(id)}
            >
              <strong>{labelOf(id)}</strong>
            </button>
          )
        })}
      </div>
      <p className="tap-count" aria-live="polite">已排 {sequence.length} / {slots.length} 句</p>
      {showFeedback && done ? <p className="kid-feedback" role="status">{right ? '答对了' : '再试一次'}</p> : null}
    </>
  )
}

export function PoemPlayer({
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
  example: PoemExample
  readOnly?: boolean
  initialAnswer?: PoemAnswer
  onAnswer?: (answer: PoemAnswer) => void
  onSolved?: () => void
  onMark?: (correct: boolean) => void
  resolveAssetUrl?: (url: string) => string
  showPrompt?: boolean
  showFeedback?: boolean
}) {
  const kind = example.kind
  const speech = example.speechUrl ? resolveAssetUrl(example.speechUrl) : ''
  return (
    <section className={`poem-player clue-board kind-${kind}${readOnly ? ' is-readonly' : ''}`}>
      <aside className="poem-scroll">
        <p className="clue-line">{example.line}</p>
      </aside>
      <div className="clue-main">
        <div className="clue-prompt">
          <PlayButton url={speech || undefined} missing={example.audioMissingReason} readOnly={readOnly} />
          <div>
            <p className="clue-kicker">{poemTitles[kind as PoemKind] ?? kind}</p>
            {showPrompt ? <h1>{example.prompt}</h1> : null}
          </div>
        </div>
        {kind === 'recite' ? (
          <ReciteLab
            example={example}
            readOnly={readOnly}
            initial={initialAnswer?.sequence}
            showFeedback={showFeedback}
            onChange={(sequence) => {
              const answer = { kind: 'recite' as const, sequence }
              onAnswer?.(answer)
              const ok = isCorrect(example, answer)
              onMark?.(ok)
              if (ok) onSolved?.()
            }}
          />
        ) : (
          <ChoiceLab
            example={example}
            readOnly={readOnly}
            initialId={initialAnswer?.selectedId}
            showFeedback={showFeedback}
            onSelect={(id) => {
              const answer = { kind, selectedId: id }
              onAnswer?.(answer)
              const ok = id === example.answerId
              onMark?.(ok)
              if (ok) onSolved?.()
            }}
          />
        )}
      </div>
    </section>
  )
}

export function PoemFacts({ example, answer }: { example: PoemExample; answer?: PoemAnswer }) {
  const facts = errorFacts(example, answer)
  if (!facts.length && !answer) return <p>未保存当时的具体作答。</p>
  return <ul className="poem-facts">{facts.map((fact) => <li key={fact}>{fact}</li>)}</ul>
}
