import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'
import { PinyinQuestion, type PinyinQuestionView } from '@kid-workbench/pinyin-player'

const listen: PinyinQuestionView = {
  id: 'listen-1',
  type: 'listen',
  stem: '听一听，选出你听到的拼音',
  speechUrl: '/api/v1/pinyin/items/1/speech/solo.mp3',
  visual: { kind: 'sound' },
  options: [
    { id: 'option-0', label: 'b' },
    { id: 'option-1', label: 'p' },
    { id: 'option-2', label: 'm' },
    { id: 'option-3', label: 'f' },
  ],
}

const inword: PinyinQuestionView = {
  id: 'inword-1',
  type: 'inword',
  stem: '听一听，这个字里藏着哪个拼音？',
  speechUrl: '/api/v1/pinyin/items/1/speech/word.mp3',
  visual: { kind: 'char', text: '播' },
  options: [
    { id: 'option-0', label: 'p' },
    { id: 'option-1', label: 'b' },
    { id: 'option-2', label: 'd' },
    { id: 'option-3', label: 't' },
  ],
}

const shape: PinyinQuestionView = {
  id: 'shape-1',
  type: 'shape',
  stem: '看一看，选出四线格里拼音的读音',
  visual: { kind: 'glyph', text: 'ɑ' },
  options: [
    { id: 'option-0', label: 'o', speechUrl: '/api/v1/pinyin/items/8/speech/solo.mp3' },
    { id: 'option-1', label: 'ɑ', speechUrl: '/api/v1/pinyin/items/7/speech/solo.mp3' },
    { id: 'option-2', label: 'e', speechUrl: '/api/v1/pinyin/items/9/speech/solo.mp3' },
    { id: 'option-3', label: 'i', speechUrl: '/api/v1/pinyin/items/10/speech/solo.mp3' },
  ],
}

const blend: PinyinQuestionView = {
  id: 'blend-1',
  type: 'blend',
  stem: '把声母和韵母拼在一起',
  visual: { kind: 'blend', initial: 'b', final: 'ā' },
  options: [
    { id: 'option-0', label: 'pā', speechUrl: '/api/v1/pinyin/items/40/speech/solo.mp3' },
    { id: 'option-1', label: 'bā', speechUrl: '/api/v1/pinyin/items/41/speech/solo.mp3' },
    { id: 'option-2', label: 'mā', speechUrl: '/api/v1/pinyin/items/42/speech/solo.mp3' },
    { id: 'option-3', label: 'fā', speechUrl: '/api/v1/pinyin/items/43/speech/solo.mp3' },
  ],
}

beforeEach(() => {
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue()
  vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => {})
})

it('keeps listen stem playback separate from choosing a letter', () => {
  const pick = vi.fn()
  render(<PinyinQuestion question={listen} allowSyntheticSpeech onPick={pick} />)
  expect(screen.getByRole('heading', { name: '听一听，选出你听到的拼音' })).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '播放读音' }))
  expect(pick).not.toHaveBeenCalled()
  fireEvent.click(screen.getByRole('button', { name: 'b' }))
  expect(pick).toHaveBeenCalledWith('option-0')
})

it('plays the inword character without choosing a letter', () => {
  const pick = vi.fn()
  render(<PinyinQuestion question={inword} allowSyntheticSpeech onPick={pick} />)
  expect(screen.getByRole('heading', { name: '听一听，这个字里藏着哪个拼音？' })).toBeInTheDocument()
  expect(screen.getByText('播')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '播放读音' }))
  expect(pick).not.toHaveBeenCalled()
  fireEvent.click(screen.getByRole('button', { name: 'p' }))
  expect(pick).toHaveBeenCalledWith('option-0')
})

it('requires shape options to be heard before they can be picked', async () => {
  const pick = vi.fn()
  render(<PinyinQuestion question={shape} allowSyntheticSpeech onPick={pick} />)
  expect(screen.getByRole('heading', { name: '看一看，选出四线格里拼音的读音' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '选择读音 1' })).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: '播放读音 1' }))
  expect(pick).not.toHaveBeenCalled()
  await waitFor(() => expect(screen.getByRole('button', { name: '选择读音 1' })).toBeEnabled())
  fireEvent.click(screen.getByRole('button', { name: '选择读音 1' }))
  expect(pick).toHaveBeenCalledWith('option-0')
  expect(screen.getByRole('button', { name: '选择读音 2' })).toBeDisabled()
})

it('requires blend options to be heard before they can be picked', async () => {
  const pick = vi.fn()
  render(<PinyinQuestion question={blend} allowSyntheticSpeech onPick={pick} />)
  expect(screen.getByText('b')).toBeInTheDocument()
  expect(screen.getByText('ā')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '选择读音 2' })).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: '播放读音 2' }))
  await waitFor(() => expect(screen.getByRole('button', { name: '选择读音 2' })).toBeEnabled())
  fireEvent.click(screen.getByRole('button', { name: '选择读音 2' }))
  expect(pick).toHaveBeenCalledWith('option-1')
})

it('replays a saved listen answer without letting history choose again', () => {
  const pick = vi.fn()
  render(<PinyinQuestion question={listen} selectedOptionId="option-1" correct={false} readOnly onPick={pick} />)
  const picked = screen.getByRole('button', { name: 'p' })
  expect(picked).toHaveAttribute('aria-pressed', 'true')
  expect(picked).toHaveClass('is-wrong')
  fireEvent.click(picked)
  expect(pick).not.toHaveBeenCalled()
  expect(screen.getByText('当时答错')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '播放读音' })).toBeEnabled()
})

it('does not use browser speech when history audio is missing', () => {
  const speak = vi.fn()
  vi.stubGlobal('speechSynthesis', { cancel: vi.fn(), speak })
  render(<PinyinQuestion question={{ ...listen, speechUrl: undefined }} readOnly />)
  fireEvent.click(screen.getByRole('button', { name: '播放读音' }))
  expect(screen.getByRole('alert')).toHaveTextContent('读音素材暂不可用')
  expect(speak).not.toHaveBeenCalled()
  vi.unstubAllGlobals()
})

it('keeps selection locked and reports rejected recording playback', async () => {
 vi.spyOn(HTMLMediaElement.prototype,'play').mockRejectedValue(new Error('offline'))
 render(<PinyinQuestion question={blend} />)
 fireEvent.click(screen.getByRole('button',{name:'播放读音 1'}))
 expect(await screen.findByRole('alert')).toHaveTextContent('读音素材暂不可用')
 expect(screen.getByRole('button',{name:'选择读音 1'})).toBeDisabled()
})

it('relocks an option if its recording errors after starting', async () => {
 const audios: HTMLAudioElement[] = []
 vi.spyOn(HTMLMediaElement.prototype,'play').mockImplementation(function(this: HTMLAudioElement){ audios.push(this); return Promise.resolve() })
 render(<PinyinQuestion question={shape} />)
 fireEvent.click(screen.getByRole('button',{name:'播放读音 1'}))
 await waitFor(()=>expect(screen.getByRole('button',{name:'选择读音 1'})).toBeEnabled())
 fireEvent.error(audios[0])
 expect(await screen.findByRole('alert')).toHaveTextContent('读音素材暂不可用')
 expect(screen.getByRole('button',{name:'选择读音 1'})).toBeDisabled()
})
