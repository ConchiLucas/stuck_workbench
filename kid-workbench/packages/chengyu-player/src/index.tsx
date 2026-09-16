import { useEffect, useRef, useState } from 'react'

export type ChengyuKind = 'meaning' | 'pick' | 'pinyin' | 'example'
export type ChengyuChoice = { id: string; label: string }
export type ChengyuBlank = { full: string; blanked: string; target: string; start: number; length: number }
export type ChengyuExample = {
  kind: ChengyuKind
  stem?: string
  prompt?: string
  speech?: string
  speechUrl?: string
  options?: ChengyuChoice[]
  answerId?: string
  chengyu?: string
  pinyin?: string
  meaning?: string
  example?: string
  blank?: ChengyuBlank
}
export type ChengyuAnswer = { kind: 'choice'; value: string; correct: boolean | null }

export const chengyuKinds: ChengyuKind[] = ['meaning', 'pick', 'pinyin', 'example']
export const chengyuTitles: Record<ChengyuKind, string> = {
  meaning: '听释义',
  pick: '选成语',
  pinyin: '看拼音',
  example: '看句子',
}

export function ListenButton({ onClick, disabled }: { onClick: () => void; disabled?: boolean }) {
  const [playing, setPlaying] = useState(false)
  const timer = useRef(0)
  useEffect(() => () => window.clearTimeout(timer.current), [])
  function play() {
    if (disabled) return
    onClick()
    setPlaying(true)
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(() => setPlaying(false), 1600)
  }
  return (
    <button type="button" className={`play-sound${playing ? ' is-playing' : ''}`} aria-label="播放读音" aria-pressed={playing} disabled={disabled} onClick={play}>
      <span className="play-sound-rings" aria-hidden="true"><i /><i /><i /></span>
      <span className="play-sound-face" data-icon="play" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M4.5 5.653c0-1.427 1.529-2.33 2.779-1.643l11.54 6.347c1.295.712 1.295 2.573 0 3.286L7.28 19.99c-1.25.687-2.779-.217-2.779-1.643V5.653Z" />
        </svg>
      </span>
    </button>
  )
}

function OptionLabel({ text }: { text: string }) {
  const lines = text.split(/(?<=[，。；;])/).filter(Boolean)
  if (lines.length <= 1) return <strong>{text}</strong>
  return (
    <strong>
      {lines.map((line, index) => (
        <span key={index}>
          {index > 0 ? <br /> : null}
          {line}
        </span>
      ))}
    </strong>
  )
}

function PromptPanel({
  kind,
  stem,
  prompt,
  showPlay,
  onPlay,
  missing,
}: {
  kind: ChengyuKind
  stem?: string
  prompt: string
  showPlay: boolean
  onPlay: () => void
  missing?: string
}) {
  if (kind === 'meaning') {
    return (
      <div className="sound-panel is-meaning">
        {showPlay ? <ListenButton onClick={onPlay} /> : null}
        {missing ? <p className="media-missing">{missing}</p> : null}
      </div>
    )
  }
  if (kind === 'pick') {
    return (
      <div className="sound-panel is-pick">
        {stem ? <p className="chengyu-stem">{stem}</p> : null}
        <p className="chengyu-prompt is-meaning">{prompt}</p>
      </div>
    )
  }
  if (kind === 'pinyin') {
    return (
      <div className="sound-panel is-pinyin">
        <p className="chengyu-prompt is-pinyin">{prompt}</p>
      </div>
    )
  }
  return (
    <div className="sound-panel is-example">
      <p className="chengyu-prompt is-example">{prompt}</p>
    </div>
  )
}

export function ChengyuPlayer({
  example,
  readOnly = false,
  locked = false,
  selectedId,
  initialAnswer,
  onSelect,
  resolveAssetUrl = (url) => url,
}: {
  example: ChengyuExample
  readOnly?: boolean
  locked?: boolean
  selectedId?: string
  initialAnswer?: ChengyuAnswer
  onSelect?: (id: string) => void
  resolveAssetUrl?: (url: string) => string
}) {
  const picked = selectedId ?? initialAnswer?.value ?? ''
  const options = example.options ?? []
  const prompt = example.prompt || (example.kind === 'pick' ? example.meaning : '') || (example.kind === 'pinyin' ? example.pinyin : '') || example.blank?.blanked || ''
  const showPlay = example.kind === 'meaning'
  const [playError, setPlayError] = useState('')
  const missing = showPlay && !example.speechUrl ? '成语读音暂不可用' : playError
  const disabled = readOnly || locked || Boolean(picked)
  async function play() {
    if (!example.speechUrl) {
      setPlayError('成语读音暂不可用')
      return
    }
    try {
      const url = resolveAssetUrl(example.speechUrl)
      const res = await fetch(url)
      if (!res.ok) {
        setPlayError('成语读音暂不可用')
        return
      }
      const blob = await res.blob()
      const objectURL = URL.createObjectURL(blob)
      const audio = new Audio(objectURL)
      audio.onended = () => URL.revokeObjectURL(objectURL)
      await audio.play()
      setPlayError('')
    } catch {
      setPlayError('成语读音暂不可用')
    }
  }
  return (
    <div className={`chengyu-player type-${example.kind}`}>
      <section className="question-workspace" aria-label="当前题目">
        <div className="question-body">
          <PromptPanel kind={example.kind} stem={example.stem} prompt={prompt} showPlay={showPlay} onPlay={() => { void play() }} missing={missing} />
          <div className="answer-area">
            <div className="option-grid">
              {options.map((option) => (
                <button
                  key={option.id || option.label}
                  type="button"
                  disabled={disabled}
                  className={`option-button${picked === option.id ? ' is-picked' : ''}${readOnly && initialAnswer && option.id === example.answerId ? ' is-right' : ''}${readOnly && initialAnswer && picked === option.id && initialAnswer.correct === false ? ' is-wrong' : ''}`}
                  aria-label={option.label}
                  onClick={() => { if (!disabled) onSelect?.(option.id) }}
                >
                  <OptionLabel text={option.label} />
                </button>
              ))}
            </div>
          </div>
        </div>
        {readOnly && initialAnswer ? (
          <p className="history-caption">{initialAnswer.correct ? '当时答对' : initialAnswer.correct === false ? '当时答错' : '当时作答'}</p>
        ) : <p className="history-caption" aria-hidden="true" />}
      </section>
    </div>
  )
}
