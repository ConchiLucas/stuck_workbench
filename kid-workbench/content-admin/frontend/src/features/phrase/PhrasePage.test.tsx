import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PhrasePage } from './PhrasePage'

const morning = {
  kpId: 1,
  title: 'Good morning.',
  zh: '早上好。',
  wrong: ['下午好。', '晚上好。'],
  scene: '早上见到老师',
  replyTo: '',
  difficulty: 1,
  moduleCode: 'greet',
  moduleName: '问候',
  moduleOrder: 1,
  kpOrder: 1,
  hasSpeech: true,
  speechAudioUrl: '/api/v1/phrase/items/1/speech.mp3',
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><PhrasePage /></QueryClientProvider>)
}

describe('PhrasePage', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('lists phrases with chinese, scene and reply, without sync or edit actions', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 1,
      groups: [{
        moduleCode: 'greet',
        moduleName: '问候',
        moduleOrder: 1,
        items: [{ ...morning, replyTo: 'How are you?' }],
      }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()

    expect(await screen.findByRole('heading', { name: '英语短句素材' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19172')
    expect(screen.queryByRole('button', { name: '从题库同步' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '生成' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '删除' })).not.toBeInTheDocument()
    expect(await screen.findByRole('option', { name: '问候' })).toBeInTheDocument()
    expect(screen.getByText('Good morning.')).toBeInTheDocument()
    expect(screen.getByText('早上好。')).toBeInTheDocument()
    expect(screen.getByText('早上见到老师')).toBeInTheDocument()
    expect(screen.getByText('应答 How are you?')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '试听整句' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '查看详情' })).toBeInTheDocument()
  })

  it('filters phrases by english or chinese', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 2,
      groups: [{
        moduleCode: 'greet',
        moduleName: '问候',
        moduleOrder: 1,
        items: [
          morning,
          {
            ...morning,
            kpId: 2,
            title: 'Thank you.',
            zh: '谢谢你。',
            scene: '别人帮了你',
            hasSpeech: false,
          },
        ],
      }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()
    expect(await screen.findByText('Good morning.')).toBeInTheDocument()
    expect(screen.getByText('Thank you.')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('搜索短句 / 中文 / 场景'), { target: { value: '谢谢' } })
    expect(screen.queryByText('Good morning.')).not.toBeInTheDocument()
    expect(screen.getByText('Thank you.')).toBeInTheDocument()
  })

  it('opens readonly details and restores focus on Escape', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 1,
      groups: [{ moduleCode: 'greet', moduleName: '问候', moduleOrder: 1, items: [morning] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    renderPage()
    const trigger = await screen.findByRole('button', { name: '查看详情' })
    trigger.focus()
    fireEvent.click(trigger)
    expect(await screen.findByRole('dialog', { name: '短句详情' })).toBeInTheDocument()
    expect(screen.getByText('听一听')).toBeInTheDocument()
    fireEvent(screen.getByRole('dialog', { name: '短句详情' }), new Event('cancel', { bubbles: true, cancelable: true }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(trigger).toHaveFocus()
  })
})
