import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { literacyApi } from './api/literacy'

vi.mock('./api/literacy', async (original) => {
  const actual = await original<typeof import('./api/literacy')>()
  return { ...actual, literacyApi: { ...actual.literacyApi,
    modules: vi.fn().mockResolvedValue([{code: 'basic'}]),
    items: vi.fn().mockResolvedValue(['山','水','日','木','火','土','石','一','二','三','十'].map((character, index) => ({kpId: index + 1, character, hasGlyph: true, hasSense: true, hasSpeech: true}))),
    tasks: vi.fn().mockRejectedValue(new Error('题目后台未连接')),
    claimTask: vi.fn(),
  } }
})
import { AppRoutes } from './App'

it('renders a quiet literacy home', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('region', { name: '题型' })).toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: '今天认哪个字？' })).not.toBeInTheDocument()
  expect(screen.getByRole('heading', { name: '看字选义' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: '看义选字' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: '写一写' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: '识字图' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: /看字选义/ })).toHaveAttribute('href', '/practice/type/glyph')
  expect(screen.getByRole('link', { name: /看义选字/ })).toHaveAttribute('href', '/practice/type/sense')
  expect(screen.getByRole('link', { name: /写一写/ })).toHaveAttribute('href', '/practice/type/write')
  expect(screen.getByRole('link', { name: /识字图/ })).toHaveAttribute('href', '/map')
  expect(screen.queryByRole('heading', { name: '听音选字' })).not.toBeInTheDocument()
  expect(screen.queryByRole('link', { name: '田字格' })).not.toBeInTheDocument()
  expect(screen.queryByRole('link', { name: '回到首页' })).not.toBeInTheDocument()
})

it('opens a listen-and-write studio without showing the character', () => {
  render(<MemoryRouter initialEntries={['/char/山']}><AppRoutes /></MemoryRouter>)
  expect(screen.getByText('听一听，写出你听到的字')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '播放读音' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '写完了' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '再写一次' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '看笔顺' })).not.toBeInTheDocument()
  expect(screen.queryByText('山')).not.toBeInTheDocument()
})

it('puts play buttons on meaning pictures when choosing a character prompt', async () => {
  const user = userEvent.setup()
  render(<MemoryRouter initialEntries={['/practice/type/glyph']}><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('progressbar', { name: '第 1 题，共 4 题' })).toHaveTextContent('1 / 4')
  expect(screen.getByRole('button', { name: '上一题' })).toBeDisabled()
  expect(screen.getByRole('button', { name: '下一题' })).toBeEnabled()
  expect(screen.queryByText('1 / 1')).not.toBeInTheDocument()
  expect(await screen.findByRole('img', {name: '山'})).toHaveAttribute('src', '/api/v1/literacy/items/1/glyph.png')
  expect(screen.getByLabelText('看字选义题面')).toBeInTheDocument()
  expect(literacyApi.tasks).not.toHaveBeenCalled()
  expect(literacyApi.claimTask).not.toHaveBeenCalled()
  expect(screen.queryByRole('button', { name: '播放读音' })).not.toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: /播放「.+」/ })).toHaveLength(4)
  await user.click(screen.getByRole('button', { name: '下一题' }))
  expect(screen.getByRole('progressbar', { name: '第 2 题，共 4 题' })).toHaveTextContent('2 / 4')
  await user.click(screen.getByRole('button', { name: '上一题' }))
  await user.click(screen.getByRole('button', { name: '山' }))
  expect(screen.getByText('答对啦')).toBeInTheDocument()
})

it('lets kids hear then write without showing the character', async () => {
  const user = userEvent.setup()
  render(<MemoryRouter initialEntries={['/practice/type/write']}><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('progressbar', { name: '第 1 题，共 4 题' })).toHaveTextContent('1 / 4')
  expect(screen.getByRole('button', { name: '播放读音' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '写完了' })).toBeInTheDocument()
  expect(screen.getByLabelText('写字板')).toBeInTheDocument()
  expect(screen.queryByText('一')).not.toBeInTheDocument()
  expect(screen.queryByText('先听读音，再写到格子里')).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: '下一题' }))
  expect(screen.getByRole('progressbar', { name: '第 2 题，共 4 题' })).toHaveTextContent('2 / 4')
  expect(screen.queryByText('二')).not.toBeInTheDocument()
})

it('puts a play button beside the meaning picture when choosing a character', async () => {
  const user = userEvent.setup()
  render(<MemoryRouter initialEntries={['/practice/type/sense']}><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('button', { name: '播放读音' })).toBeInTheDocument()
  expect(screen.queryAllByRole('button', { name: /播放「.+」/ })).toHaveLength(0)
  expect(screen.getByRole('button', { name: '水' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '火' })).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: '水' }))
  expect(screen.getByText('答对啦')).toBeInTheDocument()
})
