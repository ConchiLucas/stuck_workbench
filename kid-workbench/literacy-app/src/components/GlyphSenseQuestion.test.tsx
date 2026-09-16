import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'
import { GlyphSenseQuestion, LiteracyPlayer } from '@kid-workbench/literacy-player'

const question = { id: 'glyph-1', questionType: 'glyph_sense', interaction: 'choice' as const,
  stem: { text: '山', image: '/frozen/glyph' },
  options: [
    { id: 'water', text: '水', image: '/frozen/water', audio: '/frozen/water.mp3' },
    { id: 'hill', text: '山', image: '/frozen/hill', audio: '/frozen/hill.mp3' },
  ] }
const resolve = (ref: unknown) => typeof ref === 'string' ? ref : undefined
beforeEach(() => {
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue()
  vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => {})
})

it('renders the same material images and stable option IDs without an extra stem audio button', () => {
  const pick = vi.fn()
  render(<GlyphSenseQuestion question={question} mediaResolver={resolve} onPick={pick} />)
  expect(screen.getByRole('img', { name: '山' })).toHaveAttribute('src', '/frozen/glyph')
  expect(screen.getAllByRole('button', { name: /^(水|山)$/ }).map(el => el.getAttribute('data-option-id'))).toEqual(['water', 'hill'])
  const audio = screen.getByRole('button', { name: '播放「水」' })
  fireEvent.keyDown(audio, { key: 'Enter' }); fireEvent.click(audio)
  expect(pick).not.toHaveBeenCalled()
  fireEvent.keyDown(screen.getByRole('button', { name: '水' }), { key: ' ' })
  expect(pick).toHaveBeenCalledWith('water')
  expect(screen.queryByRole('button', { name: '播放读音' })).not.toBeInTheDocument()
})

it('replays a saved wrong selection read-only while retaining audio and without choosing the right answer for the child', () => {
  const pick = vi.fn()
  render(<GlyphSenseQuestion question={question} mediaResolver={resolve} selectedOptionId="water" correct={false} readOnly onPick={pick} />)
  const wrong = screen.getByRole('button', { name: '水' })
  expect(wrong).toHaveAttribute('aria-pressed', 'true')
  expect(wrong).toHaveClass('gs-wrong')
  fireEvent.click(wrong); fireEvent.keyDown(wrong, { key: 'Enter' })
  expect(pick).not.toHaveBeenCalled()
  expect(screen.getByRole('button', { name: '播放「水」' })).toBeEnabled()
  expect(screen.getByRole('button', { name: '播放「水」' })).toHaveAttribute('aria-disabled', 'false')
  expect(screen.getByText('当时答错')).toBeInTheDocument()
  expect(screen.queryByText('再试一次')).not.toBeInTheDocument()
})

it('does not replace missing or broken meaning pictures with a character and prevents answering an incomplete question', () => {
  const pick = vi.fn()
  const { rerender } = render(<GlyphSenseQuestion question={{ ...question, options: [{ id: 'water', text: '水' }] }} mediaResolver={resolve} onPick={pick} />)
  expect(screen.getByText('图片素材暂不可用')).toBeInTheDocument()
  expect(screen.queryByText('水')).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '水' })); expect(pick).not.toHaveBeenCalled()
  rerender(<GlyphSenseQuestion question={question} mediaResolver={resolve} onPick={pick} />)
  fireEvent.error(document.querySelector('img[src="/frozen/water"]')!)
  fireEvent.click(screen.getByRole('button', { name: '山' })); expect(pick).not.toHaveBeenCalled()
  expect(screen.getByRole('button', { name: '重新加载图片' })).toBeInTheDocument()
})

it('routes the backend glyph preview through the same view and retains retry after a wrong answer', async () => {
  const submit = vi.fn().mockResolvedValueOnce({ correct: false, canRetry: true }).mockResolvedValueOnce({ correct: true })
  render(<LiteracyPlayer mode="preview" question={question} mediaResolver={resolve} onSubmit={submit} />)
  expect(screen.getByLabelText('看字选义题面')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '水' }))
  await screen.findAllByText('再试一次')
  fireEvent.click(screen.getByRole('button', { name: '山' }))
  await waitFor(() => expect(submit).toHaveBeenLastCalledWith({ kind: 'choice', selectedOptionId: 'hill' }))
  await screen.findByText('答对啦')
})

it('does not invite another submission when a formal answer has exhausted its retries', async () => {
  const submit = vi.fn().mockResolvedValue({ correct: false, canRetry: false })
  render(<LiteracyPlayer question={question} mediaResolver={resolve} onSubmit={submit} />)
  fireEvent.click(screen.getByRole('button', { name: '水' }))
  await screen.findByText('记住这个字')
  expect(screen.queryByText('再试一次')).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '山' }))
  expect(submit).toHaveBeenCalledTimes(1)
})
