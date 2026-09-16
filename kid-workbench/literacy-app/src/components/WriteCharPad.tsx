import { useEffect, useRef } from 'react'
import HanziWriter from 'hanzi-writer'

export function WriteCharPad({
  character,
  disabled,
  hintToken = 0,
  onComplete,
  onMistake,
  onLoadError,
}: {
  character: string
  disabled?: boolean
  hintToken?: number
  onComplete: (totalMistakes: number) => void
  onMistake?: () => void
  onLoadError?: () => void
}) {
  const host = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLDivElement>(null)
  const writerRef = useRef<HanziWriter | null>(null)
  const done = useRef(false)
  const hintStarted = useRef(0)
  const onCompleteRef = useRef(onComplete)
  const onMistakeRef = useRef(onMistake)
  const onLoadErrorRef = useRef(onLoadError)
  onCompleteRef.current = onComplete
  onMistakeRef.current = onMistake
  onLoadErrorRef.current = onLoadError

  const startQuiz = (writer: HanziWriter) => {
    done.current = false
    void writer.quiz({
      leniency: 1.25,
      showHintAfterMisses: 3,
      highlightOnComplete: true,
      onMistake: () => onMistakeRef.current?.(),
      onComplete: (summary) => {
        if (done.current) return
        done.current = true
        onCompleteRef.current(summary.totalMistakes)
      },
    })
  }

  useEffect(() => {
    const el = host.current
    const box = frame.current
    if (!el || !character) return
    done.current = false
    el.replaceChildren()
    const size = Math.max(220, Math.round(box?.clientWidth || 320))
    let cancelled = false
    const writer = HanziWriter.create(el, character, {
      width: size,
      height: size,
      padding: 18,
      strokeColor: '#2c2218',
      outlineColor: '#d9b7a0',
      drawingColor: '#b3392b',
      highlightColor: '#9f2e22',
      highlightCompleteColor: '#2f5d40',
      showOutline: true,
      showCharacter: false,
      strokeAnimationSpeed: 1,
      strokeHighlightSpeed: 2,
      drawingWidth: 22,
      onLoadCharDataError: () => {
        if (!cancelled) onLoadErrorRef.current?.()
      },
    })
    writerRef.current = writer
    startQuiz(writer)
    return () => {
      cancelled = true
      writer.cancelQuiz()
      el.replaceChildren()
      writerRef.current = null
    }
  }, [character])

  useEffect(() => {
    if (!hintToken) return
    const writer = writerRef.current
    if (!writer || hintStarted.current === hintToken) return
    hintStarted.current = hintToken
    writer.cancelQuiz()
    void writer.animateCharacter().then(() => {
      if (writerRef.current !== writer) return
      startQuiz(writer)
    })
  }, [hintToken])

  return (
    <div className="write-wrap">
      <div className="tianzige stage" ref={frame}>
        <div ref={host} />
        {disabled && <div className="write-cover" />}
      </div>
    </div>
  )
}
