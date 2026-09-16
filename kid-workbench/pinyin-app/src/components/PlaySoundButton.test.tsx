import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'
import { PlaySoundButton } from './PlaySoundButton'

it('is an icon-only play control with a clay face and accessible name', () => {
  render(<PlaySoundButton playing={false} onPlay={() => {}} />)
  const button = screen.getByRole('button', { name: '播放读音' })
  expect(button).toHaveAttribute('aria-pressed', 'false')
  expect(button.querySelector('.play-sound-face')).toHaveAttribute('aria-hidden', 'true')
  expect(button.querySelector('.play-sound-rings')).toHaveAttribute('aria-hidden', 'true')
  expect(button.querySelector('[data-icon="play"]')).toBeTruthy()
  expect(button.querySelector('.play-sound-face svg')).toBeTruthy()
})

it('marks playing state and fires onPlay', async () => {
  const onPlay = vi.fn()
  const { rerender } = render(<PlaySoundButton playing={false} onPlay={onPlay} />)
  await userEvent.click(screen.getByRole('button', { name: '播放读音' }))
  expect(onPlay).toHaveBeenCalledTimes(1)

  rerender(<PlaySoundButton playing onPlay={onPlay} />)
  const button = screen.getByRole('button', { name: '播放读音' })
  expect(button).toHaveAttribute('aria-pressed', 'true')
  expect(button).toHaveClass('is-playing')
})
