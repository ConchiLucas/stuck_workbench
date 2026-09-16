type Listener = (state: 'idle' | 'playing' | 'ended' | 'error') => void

export class AudioController {
  private current?: HTMLAudioElement
  private listener?: Listener

  async play(src: string, listener?: Listener) {
    this.stop()
    const audio = new Audio(src)
    this.current = audio
    this.listener = listener
    audio.onended = () => this.emit('ended')
    audio.onerror = () => this.emit('error')
    this.emit('playing')
    try { await audio.play() } catch (error) { this.emit('error'); throw error }
  }

  stop() {
    if (this.current) {
      this.current.pause()
      this.current.onended = null
      this.current.onerror = null
    }
    this.current = undefined
    this.emit('idle')
  }

  private emit(state: Parameters<Listener>[0]) { this.listener?.(state) }
}

export const audioController = new AudioController()
