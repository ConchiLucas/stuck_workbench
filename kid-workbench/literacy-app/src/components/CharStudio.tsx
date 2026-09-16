import { useState } from 'react'
import { ListenIcon } from './Icons'
import { FreeDrawPad } from './FreeDrawPad'
import { literacyApi } from '../api/literacy'
import { looksLikeCharacter, type InkStroke } from '../lib/handwriting'
import { useLiteracyAudio } from '../hooks/useLiteracyAudio'

function speak(character: string) {
  if (!('speechSynthesis' in window)) return
  window.speechSynthesis.cancel()
  const utterance = new SpeechSynthesisUtterance(character)
  utterance.lang = 'zh-CN'
  utterance.rate = 0.72
  window.speechSynthesis.speak(utterance)
}

export function CharStudio({ character, kpId }: { character: string; kpId?: number }) {
  const audio = useLiteracyAudio()
  const [round, setRound] = useState(0)
  const [strokes, setStrokes] = useState<InkStroke[]>([])
  const [status, setStatus] = useState<'ready' | 'checking' | 'ok' | 'miss'>('ready')

  const play = () => {
    if (kpId != null) audio.play(literacyApi.speechUrl(kpId))
    else speak(character)
  }

  const restart = () => {
    setStatus('ready')
    setStrokes([])
    setRound((value) => value + 1)
  }

  const check = async () => {
    if (!strokes.length) return
    setStatus('checking')
    const result = await looksLikeCharacter(strokes, character)
    setStatus(result.ok ? 'ok' : 'miss')
  }

  const copy = status === 'ok'
    ? `写对了，这是「${character}」`
    : status === 'miss'
      ? '不太像，再听一次再写'
      : status === 'checking'
        ? '正在看你写的字……'
        : '听一听，写出你听到的字'

  return <section className="char-studio page-enter">
    <div className="char-studio-play">
      <button className="listen-fab char-listen" type="button" aria-label="播放读音" onClick={play}><ListenIcon size={72} /></button>
      <p className={`char-studio-status${status === 'ok' ? ' is-ok' : ''}${status === 'miss' ? ' is-miss' : ''}`}>{copy}</p>
      <div className="char-studio-actions">
        <button className="ghost-button" type="button" onClick={play}>再听一次</button>
        <button className="primary-button" type="button" onClick={() => void check()} disabled={!strokes.length || status === 'checking' || status === 'ok'}>写完了</button>
        <button className="ghost-button" type="button" onClick={restart}>再写一次</button>
      </div>
    </div>
    <div className="char-studio-board">
      <FreeDrawPad key={round} resetToken={round} disabled={status === 'ok' || status === 'checking'} onChange={setStrokes} />
    </div>
  </section>
}
