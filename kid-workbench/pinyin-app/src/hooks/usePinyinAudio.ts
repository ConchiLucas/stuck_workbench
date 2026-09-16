import { useEffect, useState } from 'react'
import { audioController } from '../audio/controller'

export function usePinyinAudio() {
  const [state, setState] = useState<'idle' | 'playing' | 'ended' | 'error'>('idle')
  useEffect(() => () => audioController.stop(), [])
  return { state, play: (src: string) => audioController.play(src, setState), stop: () => audioController.stop() }
}
