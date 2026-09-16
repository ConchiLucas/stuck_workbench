import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PoemPage } from './PoemPage'

const jingye = {
  kpId: 1,
  title: '静夜思',
  author: '李白',
  line1: '床前明月光',
  line2: '疑是地上霜',
  lines: ['床前明月光', '疑是地上霜', '举头望明月', '低头思故乡'],
  difficulty: 1,
  moduleCode: 'poem50',
  moduleName: '必背古诗',
  moduleOrder: 1,
  kpOrder: 1,
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><PoemPage /></QueryClientProvider>)
}

describe('PoemPage', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('lists synced poems with title, author and lines', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 1,
      groups: [{ moduleCode: 'poem50', moduleName: '必背古诗', moduleOrder: 1, items: [jingye] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()

    expect(await screen.findByRole('heading', { name: '古诗素材' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19162')
    expect(screen.queryByRole('button', { name: '从题库同步' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '按组' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '表格' })).not.toBeInTheDocument()
    expect(await screen.findByText('必背古诗')).toBeInTheDocument()
    expect(screen.getByText('静夜思')).toBeInTheDocument()
    expect(screen.getByText('〔李白〕')).toBeInTheDocument()
    expect(screen.getByText('床前明月光')).toBeInTheDocument()
    expect(screen.getByText('低头思故乡')).toBeInTheDocument()
  })

  it('filters poems by author or line', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 2,
      groups: [{
        moduleCode: 'poem50',
        moduleName: '必背古诗',
        moduleOrder: 1,
        items: [
          jingye,
          {
            ...jingye,
            kpId: 2,
            title: '春晓',
            author: '孟浩然',
            line1: '春眠不觉晓',
            line2: '处处闻啼鸟',
            lines: ['春眠不觉晓', '处处闻啼鸟'],
            kpOrder: 2,
          },
        ],
      }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()
    expect(await screen.findByText('静夜思')).toBeInTheDocument()
    expect(screen.getByText('春晓')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('搜索诗名 / 作者 / 诗句'), { target: { value: '孟浩然' } })
    expect(screen.queryByText('静夜思')).not.toBeInTheDocument()
    expect(screen.getByText('春晓')).toBeInTheDocument()
  })
})
