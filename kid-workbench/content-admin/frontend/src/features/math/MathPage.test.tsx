import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MathPage } from './MathPage'

const mathList = {
  view: 'groups',
  total: 1,
  groups: [{
    moduleCode: 'add10', moduleName: '20以内加法', moduleOrder: 1,
    items: [{
      kpId: 10, title: '2+5', kind: 'add', payload: '{"a":2,"b":5}', difficulty: 1,
      moduleCode: 'add10', moduleName: '20以内加法', moduleOrder: 1, kpOrder: 1,
      glyphImageUrl: '', speechAudioUrl: '', speechText: '二加五',
    }],
  }],
}

describe('MathPage question speech publishing', () => {
  afterEach(() => vi.restoreAllMocks())

  it('shows readiness and generates question audio for the module', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input)
      if (url.startsWith('/api/v1/math/items?')) {
        return new Response(JSON.stringify(mathList), { status: 200 })
      }
      if (url.startsWith('/api/v1/math/question-speech/status?')) {
        return new Response(JSON.stringify({ moduleCode: 'add10', total: 2, ready: 1, missing: 1 }), { status: 200 })
      }
      if (url.startsWith('/api/v1/math/question-speech/batch?') && init?.method === 'POST') {
        return new Response(JSON.stringify({ generated: 1, skipped: 1, failed: 0 }), { status: 200 })
      }
      throw new Error(`unexpected fetch ${url}`)
    })
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const user = userEvent.setup()

    render(<QueryClientProvider client={client}><MathPage /></QueryClientProvider>)

    expect(await screen.findByText('题目音频 1/2')).toBeInTheDocument()
    expect(screen.getByText(/缺少字图 · 缺少读音/)).toBeInTheDocument()
    expect(screen.queryByText(/已启用|已停用/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '停用' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '重新生成' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '审核启用' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '重做字图' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '生成字图' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '读音' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^题目$/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('region', { name: '题型详情素材' })).not.toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([url]) => String(url).includes('/math/details'))).toBe(false)
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19142')
    await user.click(screen.getByRole('button', { name: '生成本组题目音频' }))
    expect(await screen.findByText(/题目音频：生成 1，跳过 1，失败 0/)).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/math/question-speech/batch?moduleCode=add10',
      { method: 'POST' },
    )
  })
})
