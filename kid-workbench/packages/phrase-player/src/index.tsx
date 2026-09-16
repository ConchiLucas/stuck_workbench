import { useEffect, useRef, useState } from 'react'

export type PhraseKind = 'listen_zh' | 'listen_en' | 'scene' | 'reply'
export type PhraseChoice = { id: string; label: string }
export type PhraseExample = {
  kind: PhraseKind
  stem?: string
  prompt?: string
  speech?: string
  speechUrl?: string
  options?: PhraseChoice[]
  answerId?: string
  scene?: string
  replyTo?: string
}
export type PhraseAnswer = { kind: 'choice'; value: string; correct: boolean | null }

export const phraseKinds: PhraseKind[] = ['listen_zh', 'listen_en', 'scene', 'reply']
export const phraseTitles: Record<PhraseKind, string> = {
  listen_zh: '听一听',
  listen_en: '选句子',
  scene: '什么时候说',
  reply: '问与答',
}

function speak(text: string) {
  if (!text || !('speechSynthesis' in window)) return
  window.speechSynthesis.cancel()
  const utterance = new SpeechSynthesisUtterance(text)
  utterance.lang = 'en-US'
  utterance.rate = 0.86
  window.speechSynthesis.speak(utterance)
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

function PromptPanel({
  kind,
  stem,
  prompt,
  showPlay,
  playDisabled,
  onPlay,
  missing,
}: {
  kind: PhraseKind
  stem?: string
  prompt: string
  showPlay: boolean
  playDisabled?: boolean
  onPlay: () => void
  missing?: string
}) {
  if (kind === 'scene') {
    return (
      <div className="prompt-panel is-scene">
        <div className="scene-stage">
          {stem ? <p className="phrase-stem">{stem}</p> : null}
          <p className="scene-caption">{prompt}</p>
        </div>
      </div>
    )
  }
  if (kind === 'reply') {
    return (
      <div className="prompt-panel is-reply">
        <div className="reply-stage">
          <div className="reply-bubble">
            {stem ? <p className="phrase-stem">{stem}</p> : null}
            <p className="phrase-prompt is-en">{prompt}</p>
            {showPlay ? <ListenButton onClick={onPlay} disabled={playDisabled} /> : null}
            {missing ? <p className="media-missing">{missing}</p> : null}
          </div>
        </div>
      </div>
    )
  }
  return (
    <div className="prompt-panel is-listen">
      <div className="listen-stage">
        {stem ? <p className="phrase-stem">{stem}</p> : null}
        {showPlay ? <ListenButton onClick={onPlay} disabled={playDisabled} /> : null}
        {missing ? <p className="media-missing">{missing}</p> : null}
      </div>
    </div>
  )
}

export function PhrasePlayer({
  example,
  readOnly = false,
  locked = false,
  selectedId,
  initialAnswer,
  onSelect,
  resolveAssetUrl = (url) => url,
  allowSyntheticSpeech = false,
}: {
  example: PhraseExample
  readOnly?: boolean
  locked?: boolean
  selectedId?: string
  initialAnswer?: PhraseAnswer
  onSelect?: (id: string) => void
  resolveAssetUrl?: (url: string) => string
  allowSyntheticSpeech?: boolean
}) {
  const picked = selectedId ?? initialAnswer?.value ?? ''
  const options = example.options ?? []
  const prompt = example.prompt || example.scene || example.replyTo || ''
  const showPlay = example.kind === 'listen_zh' || example.kind === 'listen_en' || example.kind === 'reply'
  const missing = showPlay && !example.speechUrl && !allowSyntheticSpeech ? '读音素材暂不可用' : ''
  const disabled = readOnly || locked || Boolean(picked)
  async function play() {
    if (example.speechUrl) {
      const url = resolveAssetUrl(example.speechUrl)
      const res = await fetch(url)
      if (!res.ok) return
      const blob = await res.blob()
      const objectURL = URL.createObjectURL(blob)
      const audio = new Audio(objectURL)
      audio.onended = () => URL.revokeObjectURL(objectURL)
      await audio.play()
      return
    }
    if (allowSyntheticSpeech && example.speech) speak(example.speech)
  }
  return (
    <div className={`phrase-player type-${example.kind}`}>
      <section className="question-workspace" aria-label="当前题目">
        <div className="question-body">
          <PromptPanel kind={example.kind} stem={example.stem} prompt={prompt} showPlay={showPlay} playDisabled={false} onPlay={() => { void play() }} missing={missing} />
          <div className="answer-area">
            <div className="option-grid">
              {options.map((option) => (
                <button
                  key={option.id || option.label}
                  type="button"
                  disabled={disabled}
                  className={`option-button${picked === option.id ? ' is-picked' : ''}`}
                  onClick={() => { if (!disabled) onSelect?.(option.id) }}
                >
                  <strong>{option.label}</strong>
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
