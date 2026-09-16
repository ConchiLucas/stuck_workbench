import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PinyinPage } from './PinyinPage'

const items = ['b', 'p', 'm', 'f'].map((letter, index) => ({
  kpId: index + 1,
  letter,
  moduleCode: 'initials',
  moduleName: '声母',
  moduleOrder: 1,
  kpOrder: index + 1,
  soloText: letter,
  wordText: ['播', '坡', '摸', '佛'][index],
  wordExamples: [
    ['播', '八', '不', '白', '北'],
    ['坡', '皮', '平', '跑', '片'],
    ['摸', '马', '米', '木', '门'],
    ['佛', '风', '父', '分', '发'],
  ][index],
  soloSpeechUrl: `/audio/${letter}.mp3`,
  wordSpeechUrl: `/audio/${letter}-word.mp3`,
  glyphImageUrl: '',
}))

function renderPage() {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    if (String(input).includes('/speech/')) {
      return new Response(new Blob(['audio']), { status: 200, headers: { 'Content-Type': 'audio/mpeg' } })
    }
    return new Response(JSON.stringify({
      view: 'groups',
      total: items.length,
      groups: [{ moduleCode: 'initials', moduleName: '声母', moduleOrder: 1, items }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } })
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={client}><PinyinPage /></QueryClientProvider>)
  return fetchMock
}

describe('PinyinPage materials', () => {
  afterEach(() => { cleanup(); vi.restoreAllMocks() })
  it('keeps audio materials without quiz controls or requests', async () => {
    const fetchMock = renderPage()
    await screen.findByText('共 4 音')
    expect(screen.getByRole('button', { name: '读例字 八' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '题型' })).not.toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([url]) => String(url).includes('/quiz/'))).toBe(false)
  })
})

describe('PinyinPage example-word playback', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

	it('plays stored audio for the primary example and fetches TTS audio for the others', async () => {
    const user = userEvent.setup()
    const fetchMock = renderPage()
    await screen.findByText('共 4 音')

    class FakeAudio {
      onended: (() => void) | null = null
      onerror: (() => void) | null = null
      src = ''
      play() {
        this.onended?.()
        return Promise.resolve()
      }
    }
    vi.stubGlobal('Audio', FakeAudio)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:example')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})

    await user.click(screen.getByRole('button', { name: '读例字 八' }))
    expect(fetchMock.mock.calls.some((call) => String(call[0]).includes('/api/v1/pinyin/items/1/speech/word-1.mp3'))).toBe(true)

    await user.click(screen.getByRole('button', { name: '读例字 播' }))
    expect(fetchMock.mock.calls.some((call) => String(call[0]).includes('/api/v1/pinyin/items/1/speech/word.mp3'))).toBe(true)
  })
})
