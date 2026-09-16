type Track = { play: () => Promise<void>; pause: () => void; currentTime: number }

const silentWav = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEAESsAACJWAAACABAAZGF0YQAAAAA='

export class AudioController {
  private current?: Track
  private allowed = false
  constructor(private readonly create: (url: string) => Track = (url) => new Audio(url)) {}

  async play(url: string) {
    this.stop()
    this.current = this.create(url)
    await this.current.play()
  }

  async unlock(url = silentWav) {
    await this.play(url)
    this.allowed = true
  }

  isUnlocked() { return this.allowed }

  preload(urls: string[]) {
    urls.forEach((url) => {
      const audio = new Audio(url)
      audio.preload = 'auto'
      audio.load()
    })
  }

  stop() {
    if (!this.current) return
    this.current.pause()
    this.current.currentTime = 0
    this.current = undefined
  }
}

export const mathAudio = new AudioController()
