import { describe, expect, it, vi } from 'vitest'
import { AudioController } from './controller'

it('stops the current track before starting the next one', async () => {
  const tracks: Array<{ pause: ReturnType<typeof vi.fn>; play: ReturnType<typeof vi.fn>; currentTime: number }> = []
  const controller = new AudioController((url) => {
    const track = { pause: vi.fn(), play: vi.fn().mockResolvedValue(undefined), currentTime: 0, url }
    tracks.push(track)
    return track
  })
  await controller.play('/first.mp3')
  await controller.play('/second.mp3')
  expect(tracks[0].pause).toHaveBeenCalledOnce()
  expect(tracks[0].currentTime).toBe(0)
  expect(tracks[1].play).toHaveBeenCalledOnce()
})
