import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ChengyuPage } from './ChengyuPage'

const yixin = {
  kpId: 1,
  title: '一心一意',
  pinyin: 'yì xīn yì yì',
  meaning: '集中精神，做事专心',
  example: '做作业要一心一意。',
  wrong: ['心思不专一', '慢慢来'],
  difficulty: 1,
  moduleCode: 'daily',
  moduleName: '日常成语',
  moduleOrder: 1,
  kpOrder: 1,
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><ChengyuPage /></QueryClientProvider>)
}

describe('ChengyuPage', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('lists chengyu with pinyin, meaning and example, without sync or view toggles', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 1,
      groups: [{ moduleCode: 'daily', moduleName: '日常成语', moduleOrder: 1, items: [yixin] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()

    expect(await screen.findByRole('heading', { name: '成语素材' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19182')
    expect(screen.queryByRole('button', { name: '从题库同步' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '按组' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '表格' })).not.toBeInTheDocument()
    expect((await screen.findAllByText('日常成语')).length).toBeGreaterThan(0)
    expect(screen.getByText('一心一意')).toBeInTheDocument()
    expect(screen.getByText('yì xīn yì yì')).toBeInTheDocument()
    expect(screen.getByText('集中精神，做事专心')).toBeInTheDocument()
    expect(screen.getByText('做作业要一心一意。')).toBeInTheDocument()
  })

  it('filters chengyu by meaning or example', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 2,
      groups: [{
        moduleCode: 'daily',
        moduleName: '日常成语',
        moduleOrder: 1,
        items: [
          yixin,
          {
            ...yixin,
            kpId: 2,
            title: '狐假虎威',
            pinyin: 'hú jiǎ hǔ wēi',
            meaning: '仗着别人的势力欺负人',
            example: '他借爸爸的名义吓唬人，真是狐假虎威。',
            moduleCode: 'animal',
            moduleName: '动物成语',
            moduleOrder: 2,
          },
        ],
      }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()
    expect(await screen.findByText('一心一意')).toBeInTheDocument()
    expect(screen.getByText('狐假虎威')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('搜索成语 / 拼音 / 释义'), { target: { value: '专心' } })
    expect(screen.queryByText('狐假虎威')).not.toBeInTheDocument()
    expect(screen.getByText('一心一意')).toBeInTheDocument()
  })

  it('opens read-only detail and never shows generate or publish controls', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 1,
      groups: [{ moduleCode: 'daily', moduleName: '日常成语', moduleOrder: 1, items: [{ ...yixin, hasChengyuSpeech: true, chengyuSpeechUrl: '/api/v1/chengyu/items/1/speech.mp3' }] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    HTMLDialogElement.prototype.showModal = function showModal() { this.setAttribute('open', '') }
    HTMLDialogElement.prototype.close = function close() { this.removeAttribute('open') }

    renderPage()
    expect(await screen.findByRole('button', { name: '查看详情' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /生成|重做|导入|启用|停用|发布|删除|保存/ })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '查看详情' }))
    expect(await screen.findByRole('dialog', { name: '成语详情' })).toBeInTheDocument()
    expect(screen.getByText('听释义')).toBeInTheDocument()
    expect(screen.getByText('选成语')).toBeInTheDocument()
    expect(screen.getByText('看拼音')).toBeInTheDocument()
    expect(screen.getByText('看句子')).toBeInTheDocument()
    expect(screen.getByText('做作业要____。')).toBeInTheDocument()
  })

  it('plays existing idiom speech and does not fall back to speechSynthesis', async () => {
    const play = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('Audio', class {
      onended: (() => void) | null = null
      play = play
    })
    vi.stubGlobal('speechSynthesis', { cancel: vi.fn(), speak: vi.fn(), speaking: false })
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/speech.mp3')) {
        return new Response(new Uint8Array([73, 68, 51, 1]), { status: 200, headers: { 'Content-Type': 'audio/mpeg' } })
      }
      return new Response(JSON.stringify({
        view: 'groups',
        total: 1,
        groups: [{ moduleCode: 'daily', moduleName: '日常成语', moduleOrder: 1, items: [{ ...yixin, hasChengyuSpeech: true, chengyuSpeechUrl: '/api/v1/chengyu/items/1/speech.mp3' }] }],
      }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    })

    renderPage()
    fireEvent.click(await screen.findByRole('button', { name: '试听成语读音' }))
    await waitFor(() => expect(play).toHaveBeenCalled())
    expect(vi.mocked(window.speechSynthesis.speak)).not.toHaveBeenCalled()
  })
})
