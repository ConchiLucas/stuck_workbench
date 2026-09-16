import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { FourLineGrid } from './FourLineGrid'
import { PlaySoundButton } from './PlaySoundButton'
import { optionCharCount, practiceStem } from './practiceStem'
import { SpeakerIcon } from './SpeakerIcon'
import type { PinyinQuestionView } from './types'
import './pinyinQuestion.css'

export { practiceStem, optionCharCount } from './practiceStem'
export type { PinyinQuestionType, PinyinOption, PinyinVisual, PinyinQuestionView } from './types'

export type PinyinQuestionProps = {
  question: PinyinQuestionView
  selectedOptionId?: string
  correct?: boolean
  disabled?: boolean
  readOnly?: boolean
  allowSyntheticSpeech?: boolean
  resolveUrl?: (url?: string) => string | undefined
  onPick?: (id: string) => void
  footer?: ReactNode
}

function playMedia(url: string | undefined, text: string | undefined, allowSynthetic: boolean, onReady: () => void, onEnd: () => void, onError: () => void) {
  let stopped = false
  let audio: HTMLAudioElement | undefined
  let utterance: SpeechSynthesisUtterance | undefined
  let failed = false
  const fail = () => {
    if (stopped || failed) return
    failed = true
    if (allowSynthetic && text && 'speechSynthesis' in window) {
      utterance = new SpeechSynthesisUtterance(text)
      utterance.lang = 'zh-CN'
      utterance.rate = 0.72
      utterance.onstart = () => { if (!stopped) onReady() }
      utterance.onend = () => { if (!stopped) onEnd() }
      utterance.onerror = () => { if (!stopped) onError() }
      window.speechSynthesis.speak(utterance)
    } else onError()
  }
  if (url) {
    audio = new Audio(url)
    audio.onended = () => { if (!stopped) onEnd() }
    audio.onerror = fail
    void audio.play().then(() => { if (!stopped && !failed) onReady() }).catch(fail)
  } else fail()
  return () => {
    stopped = true
    if (audio) { audio.pause(); audio.onended = null; audio.onerror = null }
    if (utterance && 'speechSynthesis' in window) window.speechSynthesis.cancel()
  }
}

export function PinyinQuestion(props: PinyinQuestionProps) {
  return <PinyinQuestionSession key={props.question.id} {...props} />
}

function PinyinQuestionSession({
  question,
  selectedOptionId,
  correct,
  disabled,
  readOnly,
  allowSyntheticSpeech = false,
  resolveUrl,
  onPick,
  footer,
}: PinyinQuestionProps) {
  const [playingOption, setPlayingOption] = useState<number | null>(null)
  const [heard, setHeard] = useState<Set<number>>(() => new Set())
  const [audioError, setAudioError] = useState('')
  const playback = useRef(0)
  const stopAudio = useRef<(() => void) | undefined>(undefined)
  const soundOnly = question.type === 'shape' || question.type === 'blend'
  const locked = !!disabled || !!readOnly
  const resolve = (url?: string) => (resolveUrl ? resolveUrl(url) : url)
  const prompt = practiceStem(question.type, question.stem)
  const result = correct === undefined ? '' : readOnly
    ? correct ? '当时答对' : '当时答错'
    : correct ? '答对啦' : '再试一次'

  useEffect(() => () => { playback.current += 1; stopAudio.current?.() }, [])

  function playStem() {
    const url = resolve(question.speechUrl)
    if (!url && !allowSyntheticSpeech) {
      setAudioError('读音素材暂不可用')
      return
    }
    setAudioError('')
    stopAudio.current?.()
    stopAudio.current = playMedia(url, question.speechText, allowSyntheticSpeech, () => {}, () => {}, () => setAudioError('读音素材暂不可用'))
  }

  function playOption(index: number) {
    const option = question.options[index]
    const url = resolve(option?.speechUrl)
    const text = allowSyntheticSpeech ? (option?.speechText || option?.label) : undefined
    if (!url && !text) {
      setAudioError('读音素材暂不可用')
      return
    }
    const request = ++playback.current
    setAudioError('')
    setPlayingOption(index)
    stopAudio.current?.()
    stopAudio.current = playMedia(url, text, allowSyntheticSpeech, () => {
      if (playback.current === request) setHeard(now => new Set(now).add(index))
    }, () => {
      if (playback.current === request) setPlayingOption((current) => (current === index ? null : current))
    }, () => {
      if (playback.current !== request) return
      setPlayingOption(null)
      setHeard(now => { const next = new Set(now); next.delete(index); return next })
      setAudioError('读音素材暂不可用')
    })
  }

  function choose(id: string, index: number) {
    if (locked) return
    if (soundOnly && !heard.has(index)) return
    onPick?.(id)
  }

  return (
    <section className="pinyin-question" aria-label={`${prompt}题面`} data-readonly={readOnly ? 'true' : undefined}>
      <h1 className="question-prompt">{prompt}</h1>
      <div className="question-body">
        <div className={`sound-panel demo-visual demo-visual-${question.type}`}>
          {question.type === 'listen' ? <PlaySoundButton onPlay={playStem} /> : null}
          {question.type === 'inword' && question.visual.text ? (
            <button type="button" className="demo-word" onClick={playStem} aria-label="播放读音">
              {Array.from(question.visual.text).map((char, index) => <span key={`${char}-${index}`}>{char}</span>)}
            </button>
          ) : null}
          {question.type === 'shape' ? <FourLineGrid><strong>{question.visual.text}</strong></FourLineGrid> : null}
          {question.type === 'blend' ? (
            <div className="demo-blend">
              <strong>{question.visual.initial}</strong><span>+</span>
              <strong>{question.visual.final}</strong><span>=</span><strong>?</strong>
            </div>
          ) : null}
        </div>
        <div className="answer-area">
          <div className={`pq-feedback${correct === true ? ' is-success' : correct === false ? ' is-miss' : ''}`} aria-live="polite">{result}</div>
          <div className="option-grid">
            {soundOnly
              ? question.options.map((option, index) => {
                  const ready = readOnly || heard.has(index)
                  const selected = selectedOptionId === option.id
                  return (
                    <div
                      key={option.id}
                      className={`option-button audio-option${playingOption === index ? ' is-playing' : ''}${selected ? ' is-picked' : ''}${ready ? ' is-ready' : ''}${selected && correct === true ? ' is-correct' : ''}${selected && correct === false ? ' is-wrong' : ''}`}
                    >
                      <button type="button" className="audio-play" aria-label={`播放读音 ${index + 1}`} onClick={() => playOption(index)}>
                        <span className="audio-option-icon" aria-hidden="true"><SpeakerIcon /></span>
                        <span>读音 {index + 1}</span>
                      </button>
                      <button
                        type="button"
                        className="audio-pick"
                        aria-label={`选择读音 ${index + 1}`}
                        disabled={!ready || locked}
                        onClick={() => choose(option.id, index)}
                      >
                        选这个
                      </button>
                    </div>
                  )
                })
              : question.options.map((option, index) => {
                  const selected = selectedOptionId === option.id
                  const chars = optionCharCount(option.label)
                  return (
                    <button
                      key={option.id}
                      type="button"
                      className={`option-button${selected ? ' is-picked' : ''}${selected && correct === true ? ' is-correct' : ''}${selected && correct === false ? ' is-wrong' : ''}`}
                      data-chars={chars}
                      data-option-id={option.id}
                      style={{ '--option-chars': chars } as CSSProperties}
                      disabled={locked}
                      aria-pressed={selected}
                      onClick={() => choose(option.id, index)}
                    >
                      <strong>{option.label}</strong>
                    </button>
                  )
                })}
          </div>
          {footer}
        </div>
      </div>
      {audioError ? <p className="pq-error" role="alert">{audioError}</p> : null}
    </section>
  )
}
