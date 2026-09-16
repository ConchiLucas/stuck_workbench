import { useEffect, useRef, useState } from 'react'

export function speak(text: string, lang = 'en-US') {
  if (!text || !('speechSynthesis' in window)) return
  window.speechSynthesis.cancel()
  const utterance = new SpeechSynthesisUtterance(text)
  utterance.lang = lang || 'en-US'
  utterance.rate = 0.86
  window.speechSynthesis.speak(utterance)
}

export function ListenButton({ onPlay }: { onPlay: () => void }) {
  const [playing, setPlaying] = useState(false)
  const timer = useRef(0)
  useEffect(() => () => window.clearTimeout(timer.current), [])
  function handlePlay() {
    onPlay()
    window.clearTimeout(timer.current)
    setPlaying(true)
    timer.current = window.setTimeout(() => setPlaying(false), 1600)
  }
  return (
    <button type="button" className={`play-sound${playing ? ' is-playing' : ''}`} aria-label="播放读音" aria-pressed={playing} onClick={handlePlay}>
      <span className="play-sound-rings" aria-hidden="true"><i /><i /><i /></span>
      <span className="play-sound-face" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="currentColor" data-icon="play" aria-hidden="true">
          <path d="M4.5 5.653c0-1.427 1.529-2.33 2.779-1.643l11.54 6.347c1.295.712 1.295 2.573 0 3.286L7.28 19.99c-1.25.687-2.779-.217-2.779-1.643V5.653Z" />
        </svg>
      </span>
    </button>
  )
}
