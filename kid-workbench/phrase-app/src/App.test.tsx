import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { AppRoutes } from './App'
import type { PlanDetail, PlanItem } from './api/types'

function sceneItem(status: PlanItem['status'] = 'pending'): PlanItem {
  return {
    id: 1,
    seq: 1,
    kpId: 100,
    tries: 0,
    phrase: 'Good morning.',
    meaningZh: '早上好。',
    scene: '早上见到老师',
    replyTo: '',
    bucket: 'new',
    status,
    picks: '',
    optionOrder: '0,1,2,3',
    question: {
      id: 1002,
      code: 'scene',
      type: 'choice',
      stem: '这种时候该说哪一句？',
      options: [
        { label: 'Good morning.' },
        { label: 'Good afternoon.' },
        { label: 'Hello!' },
        { label: 'Good night.' },
      ],
      visual: { kind: 'scene', text: '早上见到老师' },
      speech: { text: 'Good morning.', lang: 'en-US' },
    },
  }
}

function scenePlan(status: PlanItem['status'] = 'pending'): PlanDetail {
  return {
    plan: {
      id: 9,
      planDate: '2026-09-06',
      seqNo: 1,
      subjectCode: 'phrase',
      status: status === 'pending' ? 'pending' : 'doing',
      targetCount: 1,
      doneCount: status === 'pending' ? 0 : 1,
      correctCount: status === 'correct' ? 1 : 0,
      stars: 0,
      durationSec: 0,
    },
    items: [sceneItem(status)],
  }
}

beforeEach(() => {
  let current = scenePlan()
  vi.stubGlobal('fetch', vi.fn(async (input, init) => {
    const url = String(input)
    const method = String(init?.method ?? 'GET').toUpperCase()
    if (method === 'POST' && url.endsWith('/phrase/plans')) {
      const body = JSON.parse(String(init?.body ?? '{}'))
      expect(body).toEqual({ mode: 'type', questionCode: 'scene', count: 4 })
      return new Response(JSON.stringify({ data: current, error: null }), { status: 201 })
    }
    if (url.includes('/phrase/plans/9/items/') && url.endsWith('/answer')) {
      current = scenePlan('correct')
      return new Response(JSON.stringify({ data: { correct: true, canRetry: false, tries: 1, status: 'correct', mastery: {} }, error: null }), { status: 200 })
    }
    if (url.includes('/phrase/plans/9/finish')) {
      current = { ...current, plan: { ...current.plan, status: 'done', stars: 3 } }
      return new Response(JSON.stringify({ data: { plan: current.plan, stars: 3, flowers: 0, weakPhrases: [] }, error: null }), { status: 200 })
    }
    if (url.includes('/phrase/plans/9')) {
      return new Response(JSON.stringify({ data: current, error: null }), { status: 200 })
    }
    return new Response(JSON.stringify({ data: null, error: { code: 'unexpected', message: url } }), { status: 404 })
  }))
})

afterEach(() => vi.unstubAllGlobals())

it('renders four phrase type cards', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('link', { name: '听一听' })).toHaveAttribute('href', '/practice/type/listen_zh')
  expect(screen.getByRole('link', { name: '选句子' })).toHaveAttribute('href', '/practice/type/listen_en')
  expect(screen.getByRole('link', { name: '什么时候说' })).toHaveAttribute('href', '/practice/type/scene')
  expect(screen.getByRole('link', { name: '问与答' })).toHaveAttribute('href', '/practice/type/reply')
  expect(screen.getByRole('link', { name: '听一听' }).querySelector('img')).toHaveAttribute('src', '/cards/listen_zh.png')
  expect(screen.getByRole('link', { name: '选句子' }).querySelector('img')).toHaveAttribute('src', '/cards/listen_en.png')
  expect(screen.getByRole('link', { name: '什么时候说' }).querySelector('img')).toHaveAttribute('src', '/cards/scene.png')
  expect(screen.getByRole('link', { name: '问与答' }).querySelector('img')).toHaveAttribute('src', '/cards/reply.png')
  expect(screen.queryByRole('button', { name: '听一听' })).not.toBeInTheDocument()
})

it('starts a scene plan from the gallery', async () => {
  const user = userEvent.setup()
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  await user.click(screen.getByRole('link', { name: '什么时候说' }))
  expect(await screen.findByLabelText('当前题目')).toBeInTheDocument()
  expect(await screen.findByText('早上见到老师')).toBeInTheDocument()
  expect(screen.getByRole('progressbar')).toHaveTextContent('0 / 1')
})
