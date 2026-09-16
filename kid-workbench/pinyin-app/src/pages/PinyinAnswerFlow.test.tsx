import { act, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { beforeEach, afterEach, expect, it, vi } from 'vitest'
import { AppRoutes } from '../App'
import { useDemoQuizStore } from '../store/demoQuizStore'
import { useChildStore } from '../store/childStore'

function CurrentPath() { return <output data-testid="route">{useLocation().pathname}</output> }

function response(data: unknown) { return new Response(JSON.stringify({ data, error: null })) }
let finishAnswer: (value: Response) => void
let count = 0
beforeEach(() => {
  localStorage.clear()
  useChildStore.setState({ childId: 1 })
  useDemoQuizStore.setState({ sessions: {} })
  count = 0
  vi.stubGlobal('fetch', vi.fn(async (_url, init) => {
    if (String(_url).endsWith('/answer')) return new Promise<Response>(resolve => { finishAnswer = resolve })
    const body = JSON.parse(String(init?.body ?? '{}'))
    const id = ++count
    return response({ instanceId: `q-${id}`, kpId: id, targetId: id, expiresAt: '2099-01-01T00:00:00Z', type: body.type, stem: '听一听', speechUrl: '/api/v1/pinyin/items/1/speech/solo.mp3', visual: { kind: 'sound' }, options: [{ id: 'a', label: `选项${id}A` }, { id: 'b', label: `选项${id}B` }] })
  }))
})
afterEach(() => vi.unstubAllGlobals())

it('waits for server confirmation and freezes double clicks before advancing', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '选项1A' }))
  fireEvent.click(screen.getByRole('button', { name: '选项1B' }))
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 550)) })
  expect(screen.getByRole('button', { name: '选项1A' })).toBeDisabled()
  expect(screen.queryByRole('button', { name: '选项2A' })).not.toBeInTheDocument()
  expect(screen.getByLabelText('下一题')).toHaveAttribute('aria-disabled', 'true')
  const submissions = vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith('/answer'))
  expect(submissions).toHaveLength(1)
  await act(async () => { finishAnswer(response({ instanceId: 'q-1', attemptId: 1, selectedOptionId: 'a', correct: false, answerOptionId: 'b', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 1, status: 'learning', newlyMastered: false } })) })
  expect(await screen.findByRole('button', { name: '选项2A' })).toBeInTheDocument()
})

it('can review the first answered question without resubmitting', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '选项1A' }))
  await act(async () => { finishAnswer(response({ instanceId: 'q-1', attemptId: 1, selectedOptionId: 'a', correct: false, answerOptionId: 'b', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 1, status: 'learning', newlyMastered: false } })) })
  await screen.findByRole('button', { name: '选项2A' })
  fireEvent.click(screen.getByRole('link', { name: '上一题' }))
  expect(await screen.findByRole('button', { name: '选项1A' })).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: '选项1B' }))
  expect(vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith('/answer'))).toHaveLength(1)
})

it('keeps a failed choice frozen and retries the exact payload before advancing', async () => {
  const originalFetch = fetch
  let first = true
  vi.stubGlobal('fetch', vi.fn(async (input, init) => {
    if (String(input).endsWith('/answer') && first) {
      first = false
      return new Response(JSON.stringify({ data: null, error: { code: 'unavailable', message: '暂时无法确认' } }), { status: 503 })
    }
    return originalFetch(input, init)
  }))
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '选项1A' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('暂时无法确认')
  expect(screen.getByRole('button', { name: '选项1B' })).toBeDisabled()
  expect(screen.getByLabelText('下一题')).toHaveAttribute('aria-disabled', 'true')
  fireEvent.click(screen.getByRole('button', { name: '重试提交' }))
  const submissions = vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith('/answer'))
  expect(submissions).toHaveLength(2)
  expect(submissions[0][1]?.body).toBe(submissions[1][1]?.body)
  const payload = JSON.parse(String(submissions[0][1]?.body))
  expect(payload).toMatchObject({ optionId: 'a', clientId: expect.any(String), costMs: expect.any(Number) })
  expect(submissions[0][0]).toBe('/api/v1/children/1/pinyin/quiz/q-1/answer')
  await act(async () => { finishAnswer(response({ instanceId: 'q-1', attemptId: 1, selectedOptionId: 'a', correct: true, answerOptionId: 'a', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 1, status: 'learning', newlyMastered: false } })) })
  expect(await screen.findByRole('button', { name: '选项2A' })).toBeInTheDocument()
})

it('uses server grading, labels skipped questions unattempted, and restarts with new instances', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/listen/4']}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '选项4A' }))
  await act(async () => { finishAnswer(response({ instanceId: 'q-4', attemptId: 1, selectedOptionId: 'a', correct: false, answerOptionId: 'b', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 4, status: 'learning', newlyMastered: false } })) })
  expect(await screen.findByRole('heading', { name: '答题结果' })).toBeInTheDocument()
  expect(screen.getByText('答对 0 / 4 题')).toBeInTheDocument()
  expect(screen.getAllByText('未作答')).toHaveLength(3)
  expect(screen.getByText('正确答案 选项4B')).toBeInTheDocument()
  expect(screen.queryByText('正确答案 选项1A')).not.toBeInTheDocument()
  expect(vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith('/answer'))).toHaveLength(1)
  fireEvent.click(screen.getByRole('link', { name: '再练一次' }))
  expect(await screen.findByRole('button', { name: '选项5A' })).toBeEnabled()
  const generations = vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith('/generate'))
  expect(generations).toHaveLength(8)
  expect(generations.every(([url]) => String(url) === '/api/v1/children/1/pinyin/quiz/generate')).toBe(true)
  expect(JSON.parse(String(generations[1][1]?.body))).toEqual({ type: 'listen', excludeTargetIds: [1] })
})

it('keeps the new child on their own question when the old child response arrives', async () => {
  const { useChildStore } = await import('../store/childStore')
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '选项1A' }))
  act(() => useChildStore.getState().setChildId(2))
  expect(await screen.findByRole('button', { name: '选项5A' })).toBeInTheDocument()
  await act(async () => { finishAnswer(response({ instanceId: 'q-1', attemptId: 1, selectedOptionId: 'a', correct: true, answerOptionId: 'a', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 1, status: 'learning', newlyMastered: false } })) })
  expect(screen.getByRole('button', { name: '选项5A' })).toBeEnabled()
  expect(screen.getByRole('progressbar', { name: '听音选字母，0 / 4' })).toBeInTheDocument()
  expect(vi.mocked(fetch).mock.calls.some(([url]) => String(url).includes('/children/2/pinyin/quiz/generate'))).toBe(true)
})

it('checks saved instances on reload and displays the accepted question as read only', async () => {
  const first = render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '选项1A' }))
  first.unmount()
  useDemoQuizStore.setState({ sessions: {} })
  const accepted = { instanceId: 'q-1', attemptId: 1, selectedOptionId: 'a', correct: false, answerOptionId: 'b', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 1, status: 'learning', newlyMastered: false } }
  vi.stubGlobal('fetch', vi.fn(async input => {
    const id = Number(String(input).split('q-').at(-1))
    return response({ instanceId: `q-${id}`, kpId: id, targetId: id, expiresAt: '2099-01-01T00:00:00Z', type: 'listen', stem: '听一听', speechUrl: '/api/v1/pinyin/items/1/speech/solo.mp3', visual: { kind: 'sound' }, options: [{ id: 'a', label: `选项${id}A` }, { id: 'b', label: `选项${id}B` }], ...(id === 1 ? { acceptedResult: accepted } : {}) })
  }))
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: '选项1A' })).toBeDisabled()
  expect(screen.getByRole('progressbar', { name: '听音选字母，1 / 4' })).toBeInTheDocument()
  expect(vi.mocked(fetch).mock.calls).toHaveLength(4)
  expect(vi.mocked(fetch).mock.calls.every(([url]) => /^\/api\/v1\/children\/1\/pinyin\/quiz\/q-\d$/.test(String(url)))).toBe(true)
  // The old tab's response may still arrive after reload; it cannot alter navigation.
  await act(async () => { finishAnswer(response(accepted)) })
  expect(screen.getByRole('button', { name: '选项1A' })).toBeDisabled()
})

it('invalidates only the submitting child’s home and progress after confirmation', async () => {
  const { QueryClient } = await import('@tanstack/react-query')
  const invalidation = vi.spyOn(QueryClient.prototype, 'invalidateQueries')
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '选项1A' }))
  expect(invalidation).not.toHaveBeenCalled()
  await act(async () => { finishAnswer(response({ instanceId: 'q-1', attemptId: 1, selectedOptionId: 'a', correct: true, answerOptionId: 'a', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 1, status: 'learning', newlyMastered: false } })) })
  expect(invalidation.mock.calls.map(([filters]) => filters?.queryKey)).toEqual([['home', 1], ['progress', 1]])
  invalidation.mockRestore()
})

it.each([undefined, 2])('switching children from an explicit question URL restores the new child position (%s)', async savedPosition => {
  let expectedOption = '选项5A'
  if (savedPosition) {
    await useDemoQuizStore.getState().ensure(2, 'listen')
    const childSession = useDemoQuizStore.getState().sessions['2:listen']!
    useDemoQuizStore.getState().setPosition(2, 'listen', childSession.id, savedPosition)
    expectedOption = `选项${savedPosition}A`
  }
  render(<MemoryRouter initialEntries={['/practice/type/listen/4']}><AppRoutes /><CurrentPath /></MemoryRouter>)
  await screen.findByRole('button', { name: savedPosition ? '选项8A' : '选项4A' })
  act(() => useChildStore.getState().setChildId(2))
  expect(await screen.findByRole('button', { name: expectedOption })).toBeEnabled()
  expect(useDemoQuizStore.getState().sessions['2:listen']!.position).toBe(savedPosition ?? 1)
  expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', `/practice/type/listen/${(savedPosition ?? 1) + 1}`)
  expect(useDemoQuizStore.getState().sessions['1:listen']!.position).toBe(4)
  expect(screen.getByTestId('route')).toHaveTextContent(`/practice/type/listen/${savedPosition ?? 1}`)
  act(() => useChildStore.getState().setChildId(1))
  expect(await screen.findByRole('button', { name: savedPosition ? '选项8A' : '选项4A' })).toBeEnabled()
  expect(screen.getByTestId('route')).toHaveTextContent('/practice/type/listen/4')
  expect(useDemoQuizStore.getState().sessions['2:listen']!.position).toBe(savedPosition ?? 1)
})
