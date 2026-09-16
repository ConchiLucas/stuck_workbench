import { appPath } from '../../appPath'
import { ChevronLeft, ChevronRight, X } from 'lucide-react'
import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { generatePinyinQuizSet } from '../../api/pinyin'
import type { PinyinGeneratedQuiz } from '../../api/pinyinTypes'
import {
  optionLabel,
  pickKey,
  practiceStem,
  questionPrompt,
  questionTypes,
  typeTitle,
  type DemoType,
} from './kidDemoBank'
import './kid-pinyin-preview.css'

const ADVANCE_MS = 450

type View =
  | { kind: 'gallery' }
  | { kind: 'loading'; type: DemoType }
  | { kind: 'error'; type: DemoType; message: string }
  | { kind: 'practice'; type: DemoType; n: number; questions: PinyinGeneratedQuiz[] }
  | { kind: 'result'; type: DemoType; questions: PinyinGeneratedQuiz[] }

export function KidPinyinQuizSheet({ onClose }: { onClose: () => void }) {
  const [view, setView] = useState<View>({ kind: 'gallery' })
  const [picks, setPicks] = useState<Record<string, number>>({})
  const loadGen = useRef(0)

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      if (view.kind === 'gallery') onClose()
      else setView({ kind: 'gallery' })
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, view.kind])

  function clearType(type: DemoType) {
    setPicks((current) => {
      const next = { ...current }
      for (const name of Object.keys(next)) {
        if (name.startsWith(`${type}:`)) delete next[name]
      }
      return next
    })
  }

  function openType(type: DemoType) {
    clearType(type)
    const gen = ++loadGen.current
    setView({ kind: 'loading', type })
    void generatePinyinQuizSet(type).then(
      (questions) => {
        if (loadGen.current !== gen) return
        setView({ kind: 'practice', type, n: 1, questions })
      },
      (error: unknown) => {
        if (loadGen.current !== gen) return
        setView({
          kind: 'error',
          type,
          message: error instanceof Error ? error.message : '出题失败',
        })
      },
    )
  }

  function setPick(type: DemoType, n: number, option: number) {
    setPicks((current) => ({ ...current, [pickKey(type, n)]: option }))
  }

  return (
    <div className="fullscreen-sheet kid-pinyin-sheet" role="dialog" aria-modal="true" aria-label="拼音题型">
      <div className="kid-pinyin-preview">
        {view.kind === 'gallery' ? (
          <Gallery onClose={onClose} onOpen={openType} />
        ) : view.kind === 'loading' || view.kind === 'error' ? (
          <Status
            type={view.type}
            message={view.kind === 'loading' ? '出题中…' : view.message}
            retry={view.kind === 'error'}
            onExit={() => {
              loadGen.current += 1
              setView({ kind: 'gallery' })
            }}
            onRetry={view.kind === 'error' ? () => openType(view.type) : undefined}
          />
        ) : view.kind === 'result' ? (
          <Result
            type={view.type}
            questions={view.questions}
            picks={picks}
            onExit={() => setView({ kind: 'gallery' })}
            onHome={() => setView({ kind: 'gallery' })}
            onRetry={() => openType(view.type)}
          />
        ) : (
          <Practice
            type={view.type}
            n={view.n}
            questions={view.questions}
            picks={picks}
            onExit={() => setView({ kind: 'gallery' })}
            onPick={setPick}
            onGo={(n) => setView({ kind: 'practice', type: view.type, n, questions: view.questions })}
            onResult={() => setView({ kind: 'result', type: view.type, questions: view.questions })}
          />
        )}
      </div>
    </div>
  )
}

function Gallery({ onClose, onOpen }: { onClose: () => void; onOpen: (type: DemoType) => void }) {
  return (
    <section className="type-gallery-page" aria-label="题型">
      <TopIcon label="关闭" variant="close" onClick={onClose} className="gallery-close" />
      <div className="type-gallery">
        {questionTypes.map((type) => (
          <button
            key={type.key}
            type="button"
            className={`type-card type-card-${type.key}`}
            aria-label={type.title}
            onClick={() => onOpen(type.key)}
          >
            <img src={`/pinyin-cards/${type.key}.png`} alt="" />
            <strong>{type.title}</strong>
          </button>
        ))}
      </div>
    </section>
  )
}

function Status({
  type,
  message,
  retry,
  onExit,
  onRetry,
}: {
  type: DemoType
  message: string
  retry: boolean
  onExit: () => void
  onRetry?: () => void
}) {
  return (
    <section className="practice-page demo-practice">
      <div className="practice-top">
        <TopIcon label="退出练习" variant="close" onClick={onExit} />
        <div
          className="progress-track"
          role="progressbar"
          aria-label={`${typeTitle(type)}，0 / 4`}
          aria-valuemin={0}
          aria-valuemax={4}
          aria-valuenow={0}
        >
          <i style={{ width: '0%' }} />
          <span className="progress-label" aria-hidden="true">0 / 4</span>
        </div>
        <div className="practice-nav">
          <TopIcon label="上一题" variant="prev" />
          <TopIcon label="下一题" variant="next" />
        </div>
      </div>
      <div className="quiz-status">
        <p>{message}</p>
        {retry ? (
          <button type="button" className="primary-button" onClick={onRetry}>再试一次</button>
        ) : null}
      </div>
    </section>
  )
}

function Practice({
  type,
  n,
  questions,
  picks,
  onExit,
  onPick,
  onGo,
  onResult,
}: {
  type: DemoType
  n: number
  questions: PinyinGeneratedQuiz[]
  picks: Record<string, number>
  onExit: () => void
  onPick: (type: DemoType, n: number, option: number) => void
  onGo: (n: number) => void
  onResult: () => void
}) {
  const total = questions.length
  const current = n
  const question = questions[current - 1]
  const [playingOption, setPlayingOption] = useState<number | null>(null)
  const [heard, setHeard] = useState<Set<number>>(() => new Set())
  const pickGen = useRef(0)
  const advanceTimer = useRef(0)
  const picked = picks[pickKey(type, current)]
  const soundOnly = question.type === 'shape' || question.type === 'blend'
  const done = questions.filter((_, index) => picks[pickKey(type, index + 1)] !== undefined).length
  const canPrev = current > 1
  const canNext = current < total

  useEffect(() => {
    setPlayingOption(null)
    setHeard(new Set())
    return () => window.clearTimeout(advanceTimer.current)
  }, [type, current])

  function playOption(index: number) {
    const option = question.options[index]
    setPlayingOption(index)
    setHeard((now) => {
      if (now.has(index)) return now
      const next = new Set(now)
      next.add(index)
      return next
    })
    playPronunciation(option?.speechUrl, option?.speechText || option?.label, () => {
      setPlayingOption((currentPlaying) => (currentPlaying === index ? null : currentPlaying))
    })
  }

  function choose(index: number) {
    if (soundOnly && !heard.has(index)) return
    const gen = ++pickGen.current
    onPick(type, current, index)
    window.clearTimeout(advanceTimer.current)
    advanceTimer.current = window.setTimeout(() => {
      if (pickGen.current !== gen) return
      if (current < total) onGo(current + 1)
      else onResult()
    }, ADVANCE_MS)
  }

  return (
    <section className="practice-page demo-practice">
      <div className="practice-top">
        <TopIcon label="退出练习" variant="close" onClick={onExit} />
        <div
          className="progress-track"
          role="progressbar"
          aria-label={`${typeTitle(type)}，${done} / ${total}`}
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={done}
        >
          <i style={{ width: `${total ? Math.min(100, (done / total) * 100) : 0}%` }} />
          <span className="progress-label" aria-hidden="true">{done} / {total}</span>
        </div>
        <div className="practice-nav">
          <TopIcon label="上一题" variant="prev" onClick={canPrev ? () => onGo(current - 1) : undefined} />
          <TopIcon
            label="下一题"
            variant="next"
            onClick={canNext ? () => onGo(current + 1) : onResult}
          />
        </div>
      </div>
      <section className="question-workspace" aria-label="当前题目">
        <h1 className="question-prompt">{practiceStem(type, question.stem)}</h1>
        <div className="question-body">
          <div className={`sound-panel demo-visual demo-visual-${question.type}`}>
            <DemoVisual question={question} />
          </div>
          <div className="answer-area">
            <div className="option-grid">
              {soundOnly
                ? question.options.map((option, index) => {
                    const ready = heard.has(index)
                    return (
                      <div
                        key={`${question.instanceId}-${option.id}`}
                        className={`option-button audio-option${playingOption === index ? ' is-playing' : ''}${picked === index ? ' is-picked' : ''}${ready ? ' is-ready' : ''}`}
                      >
                        <button type="button" className="audio-play" aria-label={`播放读音 ${index + 1}`} onClick={() => playOption(index)}>
                          <span className="audio-option-icon" aria-hidden="true"><SpeakerIcon /></span>
                          <span>读音 {index + 1}</span>
                        </button>
                        <button type="button" className="audio-pick" aria-label={`选择读音 ${index + 1}`} disabled={!ready} onClick={() => choose(index)}>选这个</button>
                      </div>
                    )
                  })
                : question.options.map((option, index) => (
                    <button
                      key={`${question.instanceId}-${option.id}`}
                      type="button"
                      className={`option-button${picked === index ? ' is-picked' : ''}`}
                      data-chars={Math.max(1, Array.from(option.label ?? '').length)}
                      style={{ '--option-chars': Math.max(1, Array.from(option.label ?? '').length) } as CSSProperties}
                      onClick={() => choose(index)}
                    >
                      <strong>{option.label}</strong>
                    </button>
                  ))}
            </div>
          </div>
        </div>
      </section>
    </section>
  )
}

function Result({
  type,
  questions,
  picks,
  onExit,
  onHome,
  onRetry,
}: {
  type: DemoType
  questions: PinyinGeneratedQuiz[]
  picks: Record<string, number>
  onExit: () => void
  onHome: () => void
  onRetry: () => void
}) {
  const rows = questions.map((question, index) => {
    const seq = index + 1
    const picked = picks[pickKey(type, seq)]
    const skipped = picked === undefined
    const correct = !skipped && picked === question.answerIndex
    return {
      seq,
      prompt: questionPrompt(question),
      pickedLabel: skipped ? '未选' : optionLabel(question, picked),
      answerLabel: optionLabel(question, question.answerIndex),
      skipped,
      correct,
    }
  })
  const correctCount = rows.filter((row) => row.correct).length

  return (
    <section className="result-page result-list-page">
      <TopIcon label="退出练习" variant="close" onClick={onExit} className="result-exit" />
      <p className="eyebrow">{typeTitle(type)}</p>
      <h1>答题结果</h1>
      <p className="result-count">答对 {correctCount} / {questions.length} 题</p>
      <ol className="result-list">
        {rows.map((row) => (
          <li key={row.seq} className={row.skipped ? 'is-skipped' : row.correct ? 'is-correct' : 'is-wrong'}>
            <span className="result-seq">第 {row.seq} 题</span>
            <strong>{row.prompt}</strong>
            <span>你选了 {row.pickedLabel}</span>
            <span>正确答案 {row.answerLabel}</span>
            <b>{row.skipped ? '未作答' : row.correct ? '答对' : '答错'}</b>
          </li>
        ))}
      </ol>
      <div className="result-actions">
        <button type="button" className="primary-button" onClick={onRetry}>再练一次</button>
        <button type="button" className="secondary-button" onClick={onHome}>回到首页</button>
      </div>
    </section>
  )
}

function DemoVisual({ question }: { question: PinyinGeneratedQuiz }) {
  if (question.type === 'listen') {
    return <PlaySoundButton onPlay={() => playPronunciation(question.speechUrl, question.speechText)} />
  }
  if (question.type === 'inword' && question.visual.text) {
    return (
      <button type="button" className="demo-word" onClick={() => playPronunciation(question.speechUrl, question.speechText)} aria-label="播放读音">
        {Array.from(question.visual.text).map((char, index) => <span key={`${char}-${index}`}>{char}</span>)}
      </button>
    )
  }
  if (question.type === 'shape') {
    return <div className="four-lines"><span><strong>{question.visual.text}</strong></span></div>
  }
  if (question.type === 'blend') {
    return (
      <div className="demo-blend">
        <strong>{question.visual.initial}</strong><span>+</span>
        <strong>{question.visual.final}</strong><span>=</span><strong>?</strong>
      </div>
    )
  }
  return null
}

function TopIcon({
  onClick,
  label,
  variant,
  className = '',
}: {
  onClick?: () => void
  label: string
  variant: 'close' | 'prev' | 'next'
  className?: string
}) {
  const icon = variant === 'close' ? <X strokeWidth={3} /> : variant === 'prev' ? <ChevronLeft strokeWidth={3} /> : <ChevronRight strokeWidth={3} />
  const classes = `top-icon top-icon-${variant}${className ? ` ${className}` : ''}`
  if (!onClick) {
    return <span className={classes} aria-label={label} aria-disabled="true"><span aria-hidden="true">{icon}</span></span>
  }
  return (
    <button type="button" className={classes} aria-label={label} onClick={onClick}>
      <span aria-hidden="true">{icon}</span>
    </button>
  )
}

function PlaySoundButton({ onPlay }: { onPlay: () => void }) {
  const [burst, setBurst] = useState(false)
  const burstTimer = useRef(0)
  useEffect(() => () => window.clearTimeout(burstTimer.current), [])
  return (
    <button
      type="button"
      className={`play-sound${burst ? ' is-playing' : ''}`}
      aria-label="播放读音"
      aria-pressed={burst}
      onClick={() => {
        onPlay()
        window.clearTimeout(burstTimer.current)
        setBurst(true)
        burstTimer.current = window.setTimeout(() => setBurst(false), 1600)
      }}
    >
      <span className="play-sound-rings" aria-hidden="true"><i /><i /><i /></span>
      <span className="play-sound-face" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="currentColor" data-icon="play" aria-hidden="true">
          <path d="M4.5 5.653c0-1.427 1.529-2.33 2.779-1.643l11.54 6.347c1.295.712 1.295 2.573 0 3.286L7.28 19.99c-1.25.687-2.779-.217-2.779-1.643V5.653Z" />
        </svg>
      </span>
    </button>
  )
}

function SpeakerIcon() {
  return (
    <svg className="speaker-icon" viewBox="0 0 256 256" data-icon="speaker" aria-hidden="true">
      <path className="horn" d="M64 84v88a4 4 0 0 1-4 4H32a16 16 0 0 1-16-16V96a16 16 0 0 1 16-16h28a4 4 0 0 1 4 4Zm93.15-58.15a8 8 0 0 0-10-.16l-65.57 51A4 4 0 0 0 80 79.84v96.32a4 4 0 0 0 1.55 3.15l65.57 51a8 8 0 0 0 9 .56 8.29 8.29 0 0 0 3.91-7.18V32.25a8.27 8.27 0 0 0-2.88-6.4Z" />
      <path className="wave wave-1" d="M188 100a36 36 0 0 1 0 56" />
      <path className="wave wave-2" d="M214 76a68 68 0 0 1 0 104" />
      <path className="wave wave-3" d="M240 52a100 100 0 0 1 0 152" />
    </svg>
  )
}

function playableURL(url?: string) {
  if (!url) return undefined
  try {
    const parsed = new URL(url, window.location.origin)
    if (parsed.pathname.startsWith('/api/')) return appPath(`${parsed.pathname}${parsed.search}`)
    return url
  } catch {
    return url
  }
}

function playPronunciation(url?: string, text?: string, onEnd?: () => void) {
  const done = () => onEnd?.()
  const src = playableURL(url)
  if (src) {
    const audio = new Audio(src)
    audio.onended = done
    audio.onerror = () => speak(text, done)
    void audio.play().catch(() => speak(text, done))
    return
  }
  speak(text, done)
}

function speak(text?: string, onEnd?: () => void) {
  if (!text || !('speechSynthesis' in window)) {
    onEnd?.()
    return
  }
  window.speechSynthesis.cancel()
  const utterance = new SpeechSynthesisUtterance(text)
  utterance.lang = 'zh-CN'
  utterance.rate = 0.72
  utterance.onend = () => onEnd?.()
  utterance.onerror = () => onEnd?.()
  window.speechSynthesis.speak(utterance)
}
