import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, expect, it, vi } from 'vitest'
import type { ChengyuCode, PlanDetail, PlanItem } from '../api/types'
import { PracticePage } from './PracticePage'
import { ResultPage } from './ResultPage'

function wrap(planId: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 0 } } })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/practice/${planId}/1`]}>
        <Routes>
          <Route path="practice/:planId/result" element={<ResultPage />} />
          <Route path="practice/:planId/:n?" element={<PracticePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

function item(code: ChengyuCode, extra: Partial<PlanItem> = {}, question: Partial<PlanItem['question']> = {}): PlanItem {
  const defaults: Record<ChengyuCode, PlanItem> = {
    meaning: {
      id: 1, seq: 1, kpId: 100, tries: 0, chengyu: '一心一意', pinyin: 'yì xīn yì yì', meaning: '集中精神，做事专心', example: '做作业要一心一意。',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1000, code: 'meaning', type: 'choice', stem: '这个成语是什么意思？',
        options: [
          { id: 'label:集中精神，做事专心', label: '集中精神，做事专心' },
          { id: 'label:心思不专一', label: '心思不专一' },
          { id: 'label:慢慢来', label: '慢慢来' },
          { id: 'label:随便玩玩', label: '随便玩玩' },
        ],
        visual: { kind: 'char', text: '一心一意' },
        speech: { text: '一心一意', lang: 'zh-CN', url: '/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3' },
      },
    },
    pick: {
      id: 1, seq: 1, kpId: 100, tries: 0, chengyu: '一心一意', pinyin: 'yì xīn yì yì', meaning: '集中精神，做事专心', example: '做作业要一心一意。',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1001, code: 'pick', type: 'choice', stem: '看意思，点出这个成语',
        options: [{ id: '100', label: '一心一意' }, { id: '101', label: '二话不说' }, { id: '102', label: '三心二意' }, { id: '103', label: '五颜六色' }],
        visual: { kind: 'meaning', text: '集中精神，做事专心' }, speech: { text: '一心一意', lang: 'zh-CN' },
      },
    },
    pinyin: {
      id: 1, seq: 1, kpId: 100, tries: 0, chengyu: '一心一意', pinyin: 'yì xīn yì yì', meaning: '集中精神，做事专心', example: '做作业要一心一意。',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1002, code: 'pinyin', type: 'choice', stem: '',
        options: [{ id: '100', label: '一心一意' }, { id: '101', label: '二话不说' }, { id: '102', label: '三心二意' }, { id: '103', label: '五颜六色' }],
        visual: { kind: 'pinyin', text: 'yì xīn yì yì' }, speech: { text: '一心一意', lang: 'zh-CN' },
      },
    },
    example: {
      id: 1, seq: 1, kpId: 100, tries: 0, chengyu: '一心一意', pinyin: 'yì xīn yì yì', meaning: '集中精神，做事专心', example: '做作业要一心一意。',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1003, code: 'example', type: 'choice', stem: '',
        options: [{ id: '100', label: '一心一意' }, { id: '101', label: '二话不说' }, { id: '102', label: '三心二意' }, { id: '103', label: '五颜六色' }],
        visual: { kind: 'example', text: '做作业要____。', full: '做作业要一心一意。', blanked: '做作业要____。', target: '一心一意', start: 4, length: 4 },
        speech: { text: '一心一意', lang: 'zh-CN' },
      },
    },
  }
  const base = defaults[code]
  return { ...base, ...extra, question: { ...base.question, ...question } }
}

function stubPlan(code: ChengyuCode, count = 1) {
  const items = Array.from({ length: count }, (_, index) => item(code, { id: index + 1, seq: index + 1 }))
  const play = vi.fn().mockResolvedValue(undefined)
  const state: PlanDetail = {
    plan: {
      id: 9, planDate: '2026-09-06', seqNo: 1, subjectCode: 'chengyu', status: 'doing',
      targetCount: count, doneCount: 0, correctCount: 0, stars: 0, durationSec: 0,
    },
    items,
  }
  vi.stubGlobal('Audio', class {
    onended: (() => void) | null = null
    play = play
  })
  vi.stubGlobal('speechSynthesis', { cancel: vi.fn(), speak: vi.fn(), speaking: false })
  vi.stubGlobal('fetch', vi.fn(async (input, init) => {
    const url = String(input)
    const method = String(init?.method ?? 'GET').toUpperCase()
    if (url.includes('/task-media/') && method === 'GET') {
      return new Response(new Uint8Array([73, 68, 51, 1, 2, 3]), { status: 200, headers: { 'Content-Type': 'audio/mpeg' } })
    }
    if (url.includes('/items/') && url.endsWith('/answer') && method === 'POST') {
      const body = JSON.parse(String(init?.body ?? '{}')) as { clientId: string; optionIndex: number }
      expect(body.clientId).toBeTruthy()
      const itemId = Number(url.split('/items/')[1]?.split('/')[0])
      const current = state.items.find((entry) => entry.id === itemId)
      expect(current).toBeTruthy()
      if (current) {
        current.status = 'correct'
        current.tries = 1
        current.picks = current.question.options[body.optionIndex]?.id || String(body.optionIndex)
      }
      state.plan.doneCount += 1
      state.plan.correctCount += 1
      return json({ correct: true, canRetry: false, tries: 1, status: 'correct', mastery: {} })
    }
    if (url.includes('/finish') && method === 'POST') {
      return json({ plan: { ...state.plan, status: 'done', stars: 3 }, stars: 3, flowers: 0, weakChengyu: [] })
    }
    if (url.includes('/chengyu/plans/9')) {
      return json(state)
    }
    return new Response(JSON.stringify({ data: null, error: { code: 'unexpected', message: url } }), { status: 404 })
  }))
  return { play }
}

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify({ data, error: null }), { status })
}

afterEach(() => vi.unstubAllGlobals())

it('moves to the next question without saying if the answer was right', async () => {
  const user = userEvent.setup()
  stubPlan('meaning', 2)
  render(wrap('9'))
  expect(await screen.findByRole('progressbar')).toHaveTextContent('0 / 2')
  await user.click(screen.getByRole('button', { name: '心思不专一' }))
  expect(screen.queryByText('答对了')).not.toBeInTheDocument()
  expect(screen.queryByText('看正确答案')).not.toBeInTheDocument()
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
  await waitFor(() => {
    expect(screen.getByRole('progressbar')).toHaveTextContent('1 / 2')
  })
  expect(screen.queryByRole('heading', { name: '答题结果' })).not.toBeInTheDocument()
})

it('shows the result list after the last question', async () => {
  const user = userEvent.setup()
  stubPlan('meaning')
  render(wrap('9'))
  expect(await screen.findByRole('button', { name: '播放读音' })).toBeInTheDocument()
  expect(screen.queryByText('一心一意')).not.toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: '心思不专一' }))
  expect(screen.queryByText('答对了')).not.toBeInTheDocument()
  expect(await screen.findByRole('heading', { name: '答题结果' })).toBeInTheDocument()
  expect(screen.getByText('第 1 题')).toBeInTheDocument()
  expect(screen.getByText('答对')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: '再练一次' })).toHaveAttribute('href', '/practice/type/meaning')
})

it('shows a blanked example sentence', async () => {
  stubPlan('example')
  render(wrap('9'))
  expect(await screen.findByText('做作业要____。')).toBeInTheDocument()
  expect(screen.queryByText('这句话说的是哪个成语？')).not.toBeInTheDocument()
  expect(screen.queryByText('做作业要一心一意。')).not.toBeInTheDocument()
})

it('shows the meaning as the prompt for pick questions', async () => {
  stubPlan('pick')
  render(wrap('9'))
  expect(await screen.findByText('集中精神，做事专心')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '一心一意' })).toBeInTheDocument()
})

it('shows pinyin without the idiom characters as the prompt', async () => {
  stubPlan('pinyin')
  render(wrap('9'))
  expect(await screen.findByText('yì xīn yì yì')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '一心一意' })).toBeInTheDocument()
})

it('plays frozen idiom audio for meaning questions without submitting', async () => {
  const user = userEvent.setup()
  const { play } = stubPlan('meaning')
  render(wrap('9'))
  expect(await screen.findByRole('button', { name: '集中精神，做事专心' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '集中精神，做事专心' }).querySelector('br')).toBeTruthy()
  await user.click(screen.getByRole('button', { name: '播放读音' }))
  await waitFor(() => expect(play).toHaveBeenCalled())
  expect(screen.queryByRole('heading', { name: '答题结果' })).not.toBeInTheDocument()
  expect(vi.mocked(window.speechSynthesis.speak)).not.toHaveBeenCalled()
})
