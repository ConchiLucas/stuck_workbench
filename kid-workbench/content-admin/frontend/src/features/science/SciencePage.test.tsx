import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SciencePage } from './SciencePage'

const item = {
  kpId: 7,
  title: '鸭子的脚掌',
  prompt: '哪种动物的脚掌最适合在水里游泳？',
  kind: 'choice',
  moduleCode: 'observe',
  moduleName: '观察与过程',
  moduleOrder: 80,
  kpOrder: 1,
  needsSenseImage: false,
  needsSenseImageOverride: null,
  effectiveNeedsSenseImage: false,
  glyphImageUrl: '',
  senseImageUrl: '',
  speechAudioUrl: '',
  hasSpeech: false,
  hasDiagram: false,
  summary: '鸭子脚趾间有蹼',
  explanation: '鸭子的脚趾之间有蹼，像小扇子，划水更有力。',
  funFact: '',
  reviewStatus: 'published',
  reviewedAt: null,
  contentVersion: 1,
  example: {
    kind: 'choice',
    prompt: '哪种动物的脚掌最适合在水里游泳？',
    options: [{ id: 'cat', label: '猫' }, { id: 'duck', label: '鸭子' }, { id: 'rabbit', label: '兔子' }],
    answerId: 'duck',
  },
}

describe('SciencePage', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('lists science materials without generate, save or publish actions', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups', total: 1,
      groups: [{ moduleCode: 'observe', moduleName: '观察与过程', moduleOrder: 80, items: [item] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><SciencePage /></QueryClientProvider>)

    expect(await screen.findByRole('heading', { name: '科普素材' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19122')
    expect(await screen.findByText('鸭子的脚掌')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '提交审核' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '发布' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '保存内容' })).not.toBeInTheDocument()
    expect(screen.queryByText('待审核')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '查看详情' }))
    expect(await screen.findByRole('dialog', { name: '科普详情' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '选择：鸭子' })).toBeDisabled()
  })
})
