import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ListenIcon } from './Icons'
import { FreeDrawPad } from './FreeDrawPad'
import { literacyApi } from '../api/literacy'
import { looksLikeCharacter, type InkStroke } from '../lib/handwriting'
import { playHanzi } from '../lib/speech'
import { useLiteracyAudio } from '../hooks/useLiteracyAudio'

export function ListenWritePad({
  character,
  kpId,
  disabled,
  passed,
  showHomeLink,
  onPass,
}: {
  character: string
  kpId?: number
  disabled?: boolean
  passed?: boolean
  showHomeLink?: boolean
  onPass: () => void
}) {
  const audio = useLiteracyAudio()
  const [round, setRound] = useState(0)
  const [strokes, setStrokes] = useState<InkStroke[]>([])
  const [status, setStatus] = useState<'ready' | 'checking' | 'ok' | 'miss'>(passed ? 'ok' : 'ready')

  const play = () => {
    if (kpId != null) audio.play(literacyApi.speechUrl(kpId))
    else playHanzi(character)
  }

  useEffect(() => {
    if (passed) return
    play()
  }, [character, kpId])

  const restart = () => {
    if (disabled || status === 'ok') return
    setStatus('ready')
    setStrokes([])
    setRound((value) => value + 1)
  }

  const check = async () => {
    if (!strokes.length || disabled || status === 'ok') return
    setStatus('checking')
    const result = await looksLikeCharacter(strokes, character)
    if (result.ok) {
      setStatus('ok')
      onPass()
      return
    }
    setStatus('miss')
  }

  const done = status === 'ok' || passed
  const copy = done
    ? '答对啦'
    : status === 'miss'
      ? '不太像，再听一次再写'
      : status === 'checking'
        ? '正在看你写的字……'
        : '先听读音，再写到格子里'

  return <>
    <div className="stage-body">
      <div className="stage-visual">
        <FreeDrawPad key={round} resetToken={round} disabled={disabled || done || status === 'checking'} onChange={setStrokes} />
        <button className="listen-inline" type="button" aria-label="播放读音" onClick={play}><ListenIcon size={22} /></button>
      </div>
    </div>
    <section className="choice-row" aria-label="当前题目">
      <p className={`feedback-bar demo-feedback show${done ? ' is-correct' : ''}${status === 'miss' ? ' is-wrong' : ''}`}>
        <span>{copy}</span>
        {done && showHomeLink ? <Link to="/">回首页</Link> : null}
      </p>
      <div className="write-actions">
        <button className="ghost-button" type="button" onClick={play}>再听一次</button>
        <button className="primary-button" type="button" onClick={() => void check()} disabled={!strokes.length || disabled || done || status === 'checking'}>写完了</button>
        <button className="ghost-button" type="button" onClick={restart} disabled={disabled || done}>再写一次</button>
      </div>
    </section>
  </>
}
