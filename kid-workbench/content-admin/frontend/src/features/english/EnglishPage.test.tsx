import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { EnglishPage } from './EnglishPage'

describe('EnglishPage materials', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('shows materials without question type controls', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 0,
      groups: [],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    render(<QueryClientProvider client={client}><MemoryRouter><EnglishPage /></MemoryRouter></QueryClientProvider>)

    await screen.findByText('共 0 词')
    expect(screen.queryByRole('button', { name: '从题库同步' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '按组' })).not.toBeInTheDocument()
    expect(screen.queryByRole('group', { name: '义图筛选' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '题型' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '要义图' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '生成义图' })).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19132')
  })

  it('browses a word and opens readonly detail without edit actions', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 1,
      groups: [{ moduleCode: 'animals', moduleName: '动物', moduleOrder: 1, words: [{
        kpId: 1, wordText: 'apple', moduleCode: 'animals', moduleName: '动物', moduleOrder: 1, kpOrder: 1,
        needsSenseImage: true, needsSenseImageOverride: null, effectiveNeedsSenseImage: true,
        glyphImageUrl: '/g.png', senseImageUrl: '/s.png', speechAudioUrl: '/a.mp3', meaningZh: '苹果',
      }] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><MemoryRouter><EnglishPage /></MemoryRouter></QueryClientProvider>)
    expect(await screen.findByRole('button', { name: '查看详情' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '读音' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '要义图' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '生成读音' })).not.toBeInTheDocument()
  })

  it('shows five readonly question previews when sentence and passage materials exist', async () => {
    const word = {
      kpId: 1, wordText: 'apple', moduleCode: 'animals', moduleName: '动物', moduleOrder: 1, kpOrder: 1,
      needsSenseImage: true, needsSenseImageOverride: null, effectiveNeedsSenseImage: true,
      glyphImageUrl: '/g.png', senseImageUrl: '/s1.png', speechAudioUrl: '/a.mp3', meaningZh: '苹果',
    }
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups', total: 4,
      groups: [{ moduleCode: 'animals', moduleName: '动物', moduleOrder: 1, words: [
        word,
        { ...word, kpId: 2, wordText: 'dog', senseImageUrl: '/s2.png', meaningZh: '小狗' },
        { ...word, kpId: 3, wordText: 'cat', senseImageUrl: '/s3.png', meaningZh: '小猫' },
        { ...word, kpId: 4, wordText: 'bird', senseImageUrl: '/s4.png', meaningZh: '小鸟' },
      ] }],
      sentences: [{ id: 11, code: 'this-is-an-apple', text: 'This is an apple', tokens: ['This', 'is', 'an', 'apple'], targetKpId: 1, targetWord: 'apple', speechAudioUrl: '/sent.mp3' }],
      passages: [{ id: 21, code: 'lucy-apple', passage: 'Lucy has a red apple. She puts it on the table.', prompt: 'Lucy 把什么放在桌上？', answerKpId: 1, answerWord: 'apple', optionKpIds: [1, 2, 3, 4] }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><MemoryRouter><EnglishPage /></MemoryRouter></QueryClientProvider>)
    expect(await screen.findByText('组句子素材')).toBeInTheDocument()
    expect(screen.getByText('读一读素材')).toBeInTheDocument()
    fireEvent.change(screen.getByPlaceholderText('搜索单词、句子或短文'), { target: { value: 'apple' } })
    fireEvent.click(screen.getAllByRole('button', { name: '查看详情' })[0])
    expect(await screen.findByRole('dialog', { name: 'apple素材详情' })).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: '听音选词' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '看图选词' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '组句子' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '写单词' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '读一读' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '生成读音' })).not.toBeInTheDocument()
  })
})
