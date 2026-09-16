import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, expect, it, vi } from 'vitest'
import { AppRoutes } from './App'
import { useLiveQuizStore } from './store/liveQuizStore'

afterEach(() => {
  cleanup()
  useLiveQuizStore.getState().invalidate()
  vi.unstubAllGlobals()
})

it('renders the science home route', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.queryByText('小小发现局')).not.toBeInTheDocument()
  expect(screen.queryByText('你好，卢沁一')).not.toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: '今天想怎么发现？' })).not.toBeInTheDocument()
  expect(screen.getByText('选择题')).toBeInTheDocument()
  expect(screen.getByText('连线题')).toBeInTheDocument()
  expect(screen.getByText('排序题')).toBeInTheDocument()
  expect(screen.getByText('结构标注题')).toBeInTheDocument()
  expect(screen.getAllByRole('link', { name: /查看题型：/ })).toHaveLength(4)
  expect(screen.getByRole('link', { name: '查看题型：选择题' })).toHaveAttribute('href', '/question-types/choice')
  expect(screen.getByRole('link', { name: '查看题型：结构标注题' })).toHaveAttribute('href', '/question-types/label')
})

it('opens the choice practice from the home card', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('link', { name: '查看题型：选择题' })).toHaveAttribute('href', '/question-types/choice')
})

it('renders generated choice stems when the quiz API succeeds', async () => {
  let n = 0
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (input: RequestInfo) => {
    const url = String(input)
    if (url.includes('/science/home')) {
      return new Response(JSON.stringify({ data: { child: { id: 1, name: '小朋友', grade: '', avatarUrl: '', flowers: 0 }, currentPlan: null, dueCount: 0, exploredCount: 0, modules: [] }, error: null }))
    }
    n += 1
    return new Response(JSON.stringify({
      data: {
        instanceId: `live-${n}`, type: 'choice', stem: `水里的动物是第 ${n} 题？`, targetId: n,
        visual: { kind: 'emoji', emoji: '🐠' },
        options: [{ id: 0, label: '猫' }, { id: 1, label: '金鱼' }, { id: 2, label: '兔子' }],
        answerIndex: 1,
      },
      error: null,
    }))
  }))
  render(<MemoryRouter initialEntries={['/question-types/choice']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('heading', { name: '水里的动物是第 1 题？' })).toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: '哪种动物的脚掌适合在水里游泳？' })).not.toBeInTheDocument()
})
