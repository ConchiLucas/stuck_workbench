import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { AppRoutes } from './App'
import type { PlanDetail, PlanItem } from './api/types'

function meaningItem(status: PlanItem['status'] = 'pending'): PlanItem {
  return {
    id: 1, seq: 1, kpId: 100, tries: 0, chengyu: '一心一意', pinyin: 'yì xīn yì yì', meaning: '集中精神，做事专心', example: '做作业要一心一意。',
    bucket: 'new', status, picks: '', optionOrder: '0,1,2,3',
    question: {
      id: 1000, code: 'meaning', type: 'choice', stem: '这个成语是什么意思？',
      options: [{ label: '集中精神，做事专心' }, { label: '三心二意' }, { label: '慢慢来' }, { label: '随便玩玩' }],
      visual: { kind: 'char', text: '一心一意' }, speech: { text: '一心一意', lang: 'zh-CN' },
    },
  }
}

function meaningPlan(status: PlanItem['status'] = 'pending'): PlanDetail {
  return {
    plan: {
      id: 9, planDate: '2026-09-06', seqNo: 1, subjectCode: 'chengyu',
      status: status === 'pending' ? 'pending' : 'doing',
      targetCount: 1, doneCount: status === 'pending' ? 0 : 1,
      correctCount: status === 'correct' ? 1 : 0, stars: 0, durationSec: 0,
    },
    items: [meaningItem(status)],
  }
}

beforeEach(() => {
  let current = meaningPlan()
  vi.stubGlobal('fetch', vi.fn(async (input, init) => {
    const url = String(input)
    const method = String(init?.method ?? 'GET').toUpperCase()
    if (method === 'POST' && url.endsWith('/chengyu/plans')) {
      const body = JSON.parse(String(init?.body ?? '{}'))
      expect(body).toEqual({ mode: 'type', questionCode: 'meaning', count: 4 })
      return new Response(JSON.stringify({ data: current, error: null }), { status: 201 })
    }
    if (url.includes('/chengyu/plans/9/items/') && url.endsWith('/answer')) {
      current = meaningPlan('correct')
      return new Response(JSON.stringify({ data: { correct: true, canRetry: false, tries: 1, status: 'correct', mastery: {} }, error: null }), { status: 200 })
    }
    if (url.includes('/chengyu/plans/9/finish')) {
      current = { ...current, plan: { ...current.plan, status: 'done', stars: 3 } }
      return new Response(JSON.stringify({ data: { plan: current.plan, stars: 3, flowers: 0, weakChengyu: [] }, error: null }), { status: 200 })
    }
    if (url.includes('/chengyu/plans/9')) {
      return new Response(JSON.stringify({ data: current, error: null }), { status: 200 })
    }
    return new Response(JSON.stringify({ data: null, error: { code: 'unexpected', message: url } }), { status: 404 })
  }))
})

afterEach(() => vi.unstubAllGlobals())

it('renders four chengyu type cards', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('link', { name: '听释义' })).toHaveAttribute('href', '/practice/type/meaning')
  expect(screen.getByRole('link', { name: '选成语' })).toHaveAttribute('href', '/practice/type/pick')
  expect(screen.getByRole('link', { name: '看拼音' })).toHaveAttribute('href', '/practice/type/pinyin')
  expect(screen.getByRole('link', { name: '看句子' })).toHaveAttribute('href', '/practice/type/example')
  expect(screen.getByRole('link', { name: '听释义' }).querySelector('img')).toHaveAttribute('src', '/cards/meaning.png')
  expect(screen.getByRole('link', { name: '选成语' }).querySelector('img')).toHaveAttribute('src', '/cards/pick.png')
  expect(screen.getByRole('link', { name: '看拼音' }).querySelector('img')).toHaveAttribute('src', '/cards/pinyin.png')
  expect(screen.getByRole('link', { name: '看句子' }).querySelector('img')).toHaveAttribute('src', '/cards/example.png')
  expect(screen.queryByRole('button', { name: '听释义' })).not.toBeInTheDocument()
})

it('starts a meaning plan from the gallery', async () => {
  const user = userEvent.setup()
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  await user.click(screen.getByRole('link', { name: '听释义' }))
  expect(await screen.findByLabelText('当前题目')).toBeInTheDocument()
  expect(await screen.findByRole('button', { name: '播放读音' })).toBeInTheDocument()
  expect(screen.queryByText('一心一意')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: '集中精神，做事专心' })).toBeInTheDocument()
  expect(screen.getByRole('progressbar')).toHaveTextContent('0 / 1')
})
