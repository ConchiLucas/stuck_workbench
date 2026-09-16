import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, expect, it, vi } from 'vitest'
import type { PhraseCode, PlanDetail, PlanItem } from '../api/types'
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

function item(code: PhraseCode, extra: Partial<PlanItem> = {}, question: Partial<PlanItem['question']> = {}): PlanItem {
  const defaults: Record<PhraseCode, PlanItem> = {
    listen_zh: {
      id: 1, seq: 1, kpId: 100, tries: 0, phrase: 'Good morning.', meaningZh: '早上好。', scene: '早上见到老师', replyTo: '',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1000, code: 'listen_zh', type: 'choice', stem: '听一听，选出中文意思',
        options: [{ label: '早上好。' }, { label: '下午好。' }, { label: '晚上好。' }, { label: '晚安。' }],
        visual: {}, speech: { text: 'Good morning.', lang: 'en-US' },
      },
    },
    listen_en: {
      id: 1, seq: 1, kpId: 100, tries: 0, phrase: 'Good morning.', meaningZh: '早上好。', scene: '早上见到老师', replyTo: '',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1001, code: 'listen_en', type: 'choice', stem: '听一听，点出这句英语',
        options: [{ label: 'Good morning.' }, { label: 'Hello!' }, { label: 'Thank you.' }, { label: 'Good night.' }],
        visual: {}, speech: { text: 'Good morning.', lang: 'en-US' },
      },
    },
    scene: {
      id: 1, seq: 1, kpId: 100, tries: 0, phrase: 'Good morning.', meaningZh: '早上好。', scene: '早上见到老师', replyTo: '',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1002, code: 'scene', type: 'choice', stem: '这种时候该说哪一句？',
        options: [{ label: 'Good morning.' }, { label: 'Good afternoon.' }, { label: 'Hello!' }, { label: 'Good night.' }],
        visual: { kind: 'scene', text: '早上见到老师' }, speech: { text: 'Good morning.', lang: 'en-US' },
      },
    },
    reply: {
      id: 1, seq: 1, kpId: 101, tries: 0, phrase: "I'm fine.", meaningZh: '我很好。', scene: '别人问你好不好', replyTo: 'How are you?',
      bucket: 'new', status: 'pending', picks: '', optionOrder: '0,1,2,3',
      question: {
        id: 1003, code: 'reply', type: 'choice', stem: '对方说了这句话，你怎么答？',
        options: [{ label: "I'm fine." }, { label: 'Hello!' }, { label: 'Thank you.' }, { label: 'Good morning.' }],
        visual: { kind: 'prompt', text: 'How are you?' }, speech: { text: 'How are you?', lang: 'en-US' },
      },
    },
  }
  const base = defaults[code]
  return { ...base, ...extra, question: { ...base.question, ...question } }
}

function planFor(current: PlanItem, status: PlanDetail['plan']['status'] = 'doing'): PlanDetail {
  return {
    plan: {
      id: 9, planDate: '2026-09-06', seqNo: 1, subjectCode: 'phrase', status,
      targetCount: 1, doneCount: current.status === 'pending' ? 0 : 1,
      correctCount: current.status === 'correct' ? 1 : 0, stars: 0, durationSec: 0,
    },
    items: [current],
  }
}

function stubPlan(code: PhraseCode, question: Partial<PlanItem['question']> = {}) {
  let current = item(code, {}, question)
  const speak = vi.fn()
  vi.stubGlobal('speechSynthesis', { cancel: vi.fn(), speak, speaking: false })
  vi.stubGlobal('fetch', vi.fn(async (input, init) => {
    const url = String(input)
    const method = String(init?.method ?? 'GET').toUpperCase()
    if (url.includes('/task-media/') && method === 'GET') {
      return new Response(new Uint8Array([0x49, 0x44, 0x33]), { status: 200, headers: { 'Content-Type': 'audio/mpeg' } })
    }
    if (url.includes('/items/') && url.endsWith('/answer') && method === 'POST') {
      const body = JSON.parse(String(init?.body ?? '{}')) as { clientId: string; optionIndex: number }
      expect(body.clientId).toBeTruthy()
      const correct = body.optionIndex === 0
      current = { ...current, status: correct ? 'correct' : 'wrong', tries: 1, picks: String(body.optionIndex) }
      return json({ correct, canRetry: false, tries: 1, status: current.status, mastery: {} })
    }
    if (url.includes('/finish') && method === 'POST') {
      return json({ plan: { ...planFor(current, 'done').plan, status: 'done', stars: 3 }, stars: 3, flowers: 0, weakPhrases: [] })
    }
    if (url.includes('/phrase/plans/9')) {
      return json(planFor(current, current.status === 'pending' ? 'doing' : 'doing'))
    }
    return new Response(JSON.stringify({ data: null, error: { code: 'unexpected', message: url } }), { status: 404 })
  }))
  return { speak }
}

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify({ data, error: null }), { status })
}

afterEach(() => vi.unstubAllGlobals())

it('shows a scene caption without the home-card illustration', async () => {
  stubPlan('scene')
  render(wrap('9'))
  expect(await screen.findByText('早上见到老师')).toBeInTheDocument()
  expect(screen.getByLabelText('当前题目').querySelector('img')).toBeNull()
})

it('answers a scene plan without live feedback, then shows the result list', async () => {
  const user = userEvent.setup()
  stubPlan('scene')
  render(wrap('9'))
  expect(await screen.findByText('早上见到老师')).toBeInTheDocument()
  expect(screen.getByRole('progressbar')).toHaveTextContent('0 / 1')
  expect(screen.getByRole('button', { name: 'Good morning.' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: '查看结果' })).toHaveAttribute('href', '/practice/9/result')
  await user.click(screen.getByRole('button', { name: 'Good afternoon.' }))
  expect(screen.getByRole('progressbar')).toHaveTextContent('1 / 1')
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
  expect(screen.queryByText('再试一次')).not.toBeInTheDocument()
  expect(screen.queryByText('答对了')).not.toBeInTheDocument()
  expect(screen.queryByText('看正确答案')).not.toBeInTheDocument()
  expect(await screen.findByRole('heading', { name: '答题结果' })).toBeInTheDocument()
  expect(screen.getByText('答错')).toBeInTheDocument()
  expect(screen.getByText('你选了 Good afternoon.')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: '再练一次' })).toHaveAttribute('href', '/practice/type/scene')
})

it('shows the English prompt for a reply question without the home-card illustration', async () => {
  stubPlan('reply')
  render(wrap('9'))
  expect(await screen.findByText('How are you?')).toBeInTheDocument()
  expect(screen.getByText('对方说了这句话，你怎么答？')).toBeInTheDocument()
  expect(screen.getByLabelText('当前题目').querySelector('img')).toBeNull()
})

it('plays frozen sentence audio for listen_zh without submitting', async () => {
  const user = userEvent.setup()
  const play = vi.fn(async () => undefined)
  class FakeAudio {
    src = ''
    onended: (() => void) | null = null
    constructor(src: string) { this.src = src }
    play = play
  }
  vi.stubGlobal('Audio', FakeAudio)
  URL.createObjectURL = () => 'blob:fake'
  URL.revokeObjectURL = () => {}
  const { speak } = stubPlan('listen_zh', { speech: { text: 'Good morning.', lang: 'en-US', url: '/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3' } })
  render(wrap('9'))
  expect(await screen.findByRole('button', { name: '早上好。' })).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: '播放读音' }))
  expect(speak).not.toHaveBeenCalled()
  expect(play).toHaveBeenCalled()
  expect(vi.mocked(fetch).mock.calls.some((call) => String(call[1]?.method ?? 'GET').toUpperCase() === 'POST')).toBe(false)
})

it('shows missing speech instead of browser TTS', async () => {
  stubPlan('listen_zh')
  vi.stubGlobal('speechSynthesis', { cancel: vi.fn(), speak: vi.fn(), speaking: false })
  render(wrap('9'))
  expect(await screen.findByText('读音素材暂不可用')).toBeInTheDocument()
})
