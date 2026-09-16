import { useEffect, useRef, useState, type ReactNode } from 'react'
import type { PlayerQuestion } from './LiteracyPlayer'
import './glyphSense.css'

export type GlyphSenseProps = {
  question: PlayerQuestion
  mediaResolver: (ref: unknown) => string | undefined
  selectedOptionId?: string
  correct?: boolean
  disabled?: boolean
  readOnly?: boolean
  onPick?: (id: string) => void
  feedback?: ReactNode
  footer?: ReactNode
}

// All hosts share this view; judging and saving remain host responsibilities.
export function GlyphSenseQuestion(props: GlyphSenseProps) {
  return <GlyphSenseSession key={props.question.id} {...props} />
}

function GlyphSenseSession({ question, mediaResolver, selectedOptionId, correct, disabled,
  readOnly, onPick, feedback, footer }: GlyphSenseProps) {
  const [failed, setFailed] = useState<string[]>([])
  const [retry, setRetry] = useState(0)
  const [audioError, setAudioError] = useState('')
  const audio = useRef<HTMLAudioElement | null>(null)
  const playback = useRef(0)
  const resolve = (ref: unknown) => ref ? mediaResolver(ref) : undefined
  const stem = resolve(question.stem.image)
  const options = (question.options ?? []).map(option => ({ ...option,
    imageUrl: resolve(option.image), audioUrl: resolve(option.audio) }))
  const complete = !!stem && !failed.includes(stem) && options.length > 0 &&
    options.every(option => !!option.imageUrl && !failed.includes(option.imageUrl))
  const locked = disabled || readOnly || !complete
  function stopAudio() {
    playback.current++
    audio.current?.pause()
    audio.current = null
  }
  useEffect(() => stopAudio, [])
  async function play(url: string) {
    stopAudio()
    setAudioError('')
    const request = playback.current
    const next = new Audio(url)
    audio.current = next
    try { await next.play() } catch {
      if (request === playback.current) setAudioError('读音无法播放，请再试一次')
    }
  }
  function image(url: string | undefined, alt: string) {
    if (!url || failed.includes(url)) return <span className="gs-missing">图片素材暂不可用</span>
    return <img key={`${url}:${retry}`} src={url} alt={alt}
      onError={() => setFailed(previous => previous.includes(url) ? previous : [...previous, url])} />
  }
  const result = feedback ?? (correct === undefined ? '' : readOnly
    ? correct ? '当时答对' : '当时答错'
    : correct ? '答对啦' : '再试一次')
  return <section className="glyph-sense-question" aria-label="看字选义题面" data-readonly={!!readOnly}>
    <div className="gs-layout">
      <div className="gs-stem"><div className="gs-tianzige">{image(stem, question.stem.text ?? '题目字图')}</div></div>
      <div className="gs-answers">
        <div className={`gs-feedback${correct === true ? ' gs-success' : correct === false ? ' gs-miss' : ''}`} aria-live="polite">{result}{footer}</div>
        <div className="gs-options">{options.map((option, index) => {
          const selected = selectedOptionId === option.id
          const label = option.text || `选项 ${index + 1}`
          const pick = () => { if (!locked) onPick?.(option.id) }
          return <div key={option.id} role="button" aria-label={label} data-option-id={option.id}
            aria-pressed={selected} aria-disabled={!!locked} tabIndex={locked ? -1 : 0}
            className={`gs-option${selected && correct === true ? ' gs-correct' : selected && correct === false ? ' gs-wrong' : ''}`}
            onClick={pick} onKeyDown={event => {
              if (event.target !== event.currentTarget) return
              if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); pick() }
            }}>
            {image(option.imageUrl, '')}
            <button type="button" className="gs-play" aria-label={`播放「${label}」`}
              title={option.audioUrl ? undefined : '读音素材暂不可用'} disabled={!option.audioUrl} aria-disabled={!option.audioUrl}
              onClick={event => { event.stopPropagation(); if (option.audioUrl) void play(option.audioUrl) }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><path d="M11 5 6 9H3v6h3l5 4V5Z"/><path d="M15 8a6 6 0 0 1 0 8m3-11a10 10 0 0 1 0 14"/></svg>
            </button>
          </div>
        })}</div>
      </div>
    </div>
    {!!failed.length && <div className="gs-error" role="alert">图片加载失败 <button onClick={() => { setFailed([]); setRetry(value => value + 1) }}>重新加载图片</button></div>}
    {audioError && <p className="gs-error" role="alert">{audioError}</p>}
  </section>
}
