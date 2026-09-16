import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LogicPage } from './LogicPage'

const patternItem = {
  kpId: 1,
  title: '红蓝交替',
  kind: 'pattern',
  seq: ['🔴', '🔵', '🔴', '🔵'],
  answer: '🔴',
  wrong: ['🟢', '🟡'],
  prompt: '下一个是哪个？',
  speech: '下一个是哪个？',
  difficulty: 1,
  moduleCode: 'pattern',
  moduleName: '找规律',
  moduleOrder: 1,
  kpOrder: 1,
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><LogicPage /></QueryClientProvider>)
}

describe('LogicPage', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('lists logic items with sequence, prompt and answer, without sync or view toggles', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 1,
      groups: [{ moduleCode: 'pattern', moduleName: '找规律', moduleOrder: 1, items: [patternItem] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()

    expect(await screen.findByRole('heading', { name: '逻辑素材' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19192')
    expect(screen.queryByRole('button', { name: '从题库同步' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '按组' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '表格' })).not.toBeInTheDocument()
    expect(await screen.findByText('找规律')).toBeInTheDocument()
    expect(screen.getByText('红蓝交替')).toBeInTheDocument()
    expect(screen.getByText('下一个是哪个？')).toBeInTheDocument()
    expect(screen.getByText('🔴 🔵 🔴 🔵')).toBeInTheDocument()
    expect(screen.getByText('答案 🔴')).toBeInTheDocument()
    expect(screen.getByText('🟢')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /生成|编辑|保存|导入|启用|停用|发布|删除/ })).not.toBeInTheDocument()
  })

  it('filters items by title or prompt', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 2,
      groups: [{
        moduleCode: 'pattern',
        moduleName: '找规律',
        moduleOrder: 1,
        items: [
          patternItem,
          {
            ...patternItem,
            kpId: 2,
            title: '哪个不是水果',
            kind: 'classify',
            seq: [],
            answer: '🚗',
            wrong: ['🍎', '🍌'],
            prompt: '哪个不是水果？',
            moduleCode: 'classify',
            moduleName: '分类',
            moduleOrder: 2,
          },
        ],
      }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    renderPage()
    expect(await screen.findByText('红蓝交替')).toBeInTheDocument()
    expect(screen.getByText('哪个不是水果')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('搜索题目 / 序列 / 答案'), { target: { value: '水果' } })
    expect(screen.queryByText('红蓝交替')).not.toBeInTheDocument()
    expect(screen.getByText('哪个不是水果')).toBeInTheDocument()
  })
})
