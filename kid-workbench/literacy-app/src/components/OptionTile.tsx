import type { KeyboardEvent, MouseEvent } from 'react'
import { ListenIcon } from './Icons'

export function OptionTile({
  label,
  picture,
  imageSrc,
  showPlay,
  pressed,
  tone,
  onPick,
  onPlay,
}: {
  label: string
  picture?: string
  imageSrc?: string
  showPlay?: boolean
  pressed?: boolean
  tone?: 'correct' | 'wrong'
  onPick: () => void
  onPlay?: () => void
}) {
  const play = (event: MouseEvent) => {
    event.stopPropagation()
    onPlay?.()
  }
  const keyPick = (event: KeyboardEvent) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      onPick()
    }
  }
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={label}
      aria-pressed={pressed}
      className={`option-button${picture || imageSrc ? ' is-picture' : ' is-char'}${tone === 'correct' ? ' correct-answer' : ''}${tone === 'wrong' ? ' wrong-answer' : ''}`}
      onClick={onPick}
      onKeyDown={keyPick}
    >
      {imageSrc ? <img src={imageSrc} alt="" /> : picture ? <span className="option-picture" aria-hidden="true">{picture}</span> : <strong>{label}</strong>}
      {showPlay ? <button className="option-play" type="button" aria-label={`播放「${label}」`} onClick={play}><ListenIcon size={18} /></button> : null}
    </div>
  )
}
