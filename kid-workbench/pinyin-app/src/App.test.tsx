import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { AppRoutes } from './App'
import type { PinyinGeneratedQuiz } from './api/types'
import { useChildStore } from './store/childStore'
import { useDemoQuizStore } from './store/demoQuizStore'

function quiz(
  type: PinyinGeneratedQuiz['type'],
  targetId: number,
  options: PinyinGeneratedQuiz['options'],
  extra: Partial<PinyinGeneratedQuiz> = {},
): PinyinGeneratedQuiz {
  return {
    instanceId: `${type}-${targetId}`,
    type,
    stem: 'stem',
    targetId,
    speechText: extra.speechText ?? '玻',
    speechUrl: extra.speechUrl ?? `/api/v1/pinyin/items/${targetId}/speech/solo.mp3`,
    visual: extra.visual ?? { kind: 'sound' },
    options: options.map(option => ({...option, speechUrl: option.speechUrl ?? `/api/v1/pinyin/syllables/${option.id}/speech.mp3`})),
    kpId: targetId + 100,
    expiresAt: '2099-01-01T00:00:00Z',
  }
}

const banks: Record<string, PinyinGeneratedQuiz[]> = {
  listen: [
    quiz('listen', 1, [{ id: '1', label: 'b' }, { id: '2', label: 'p' }, { id: '3', label: 'm' }, { id: '4', label: 'f' }]),
    quiz('listen', 2, [{ id: '5', label: 'd' }, { id: '6', label: 't' }, { id: '7', label: 'n' }, { id: '8', label: 'l' }], { speechText: '得' }),
    quiz('listen', 3, [{ id: '9', label: 'g' }, { id: '10', label: 'k' }, { id: '11', label: 'h' }, { id: '12', label: 'j' }]),
    quiz('listen', 4, [{ id: '13', label: 'j' }, { id: '14', label: 'q' }, { id: '15', label: 'x' }, { id: '16', label: 'z' }]),
  ],
  inword: [
    quiz('inword', 1, [{ id: '1', label: 'p' }, { id: '2', label: 'b' }, { id: '3', label: 'd' }, { id: '4', label: 't' }], {
      speechText: '播', visual: { kind: 'char', text: '播' },
    }),
    quiz('inword', 2, [{ id: '5', label: 'b' }, { id: '6', label: 'd' }, { id: '7', label: 't' }, { id: '8', label: 'n' }], {
      visual: { kind: 'char', text: '大' },
    }),
    quiz('inword', 3, [{ id: '9', label: 'n' }, { id: '10', label: 'm' }, { id: '11', label: 'f' }, { id: '12', label: 'l' }], {
      visual: { kind: 'char', text: '妈' },
    }),
    quiz('inword', 4, [{ id: '13', label: 'b' }, { id: '14', label: 'p' }, { id: '15', label: 'm' }, { id: '16', label: 'f' }], {
      visual: { kind: 'char', text: '苹' },
    }),
  ],
  shape: [1, 2, 3, 4].map((id) => quiz('shape', id, [
    { id: '1', label: 'o', speechText: '哦' },
    { id: '2', label: 'ɑ', speechText: '啊' },
    { id: '3', label: 'e', speechText: '鹅' },
    { id: '4', label: 'i', speechText: '衣' },
  ], { visual: { kind: 'glyph', text: 'ɑ' } })),
  blend: [1, 2, 3, 4].map((id) => quiz('blend', id, [
    { id: '1', label: 'pā', speechText: '趴' },
    { id: '2', label: 'bā', speechText: '八' },
    { id: '3', label: 'mā', speechText: '妈' },
    { id: '4', label: 'fā', speechText: '发' },
  ], { visual: { kind: 'blend', initial: 'b', final: 'ā' } })),
}

beforeEach(() => {
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue()
  vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => {})
  localStorage.clear()
  useChildStore.setState({ childId: 1 })
  useDemoQuizStore.setState({ sessions: {} })
  const queues: Record<string, PinyinGeneratedQuiz[]> = {
    listen: banks.listen.map((item) => ({ ...item })),
    inword: banks.inword.map((item) => ({ ...item })),
    shape: banks.shape.map((item) => ({ ...item })),
    blend: banks.blend.map((item) => ({ ...item })),
  }
  vi.stubGlobal('fetch', vi.fn(async (input, init) => {
    const url = String(input)
    if (url.endsWith('/answer')) {
      const body = JSON.parse(String(init?.body ?? '{}'))
      const instanceId = url.split('/').at(-2)
      const question = Object.values(banks).flat().find(item => item.instanceId === instanceId)!
      return new Response(JSON.stringify({ data: { instanceId, attemptId: 1, selectedOptionId: body.optionId, correct: body.optionId === question.options[0].id, answerOptionId: question.options[0].id, skill: { code: question.type, status: 'learning' }, knowledge: { kpId: question.kpId, status: 'learning', newlyMastered: false } }, error: null }))
    }
    if (!url.includes('/children/1/pinyin/quiz/generate')) {
      return new Response(JSON.stringify({ data: null, error: { code: 'unexpected', message: url } }), { status: 404 })
    }
    const body = JSON.parse(String(init?.body ?? '{}')) as { type: string }
    const next = queues[body.type]?.shift()
    return new Response(JSON.stringify({ data: next, error: null }), { status: 200 })
  }))
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('renders the pinyin home route', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('link', { name: '听音选字母' })).toHaveAttribute('href', '/practice/type/listen')
  expect(screen.getByRole('link', { name: '字中找拼音' })).toHaveAttribute('href', '/practice/type/inword')
  expect(screen.getByRole('link', { name: '看形认读' })).toHaveAttribute('href', '/practice/type/shape')
  expect(screen.getByRole('link', { name: '声韵拼读' })).toHaveAttribute('href', '/practice/type/blend')
  expect(screen.queryByRole('heading', { name: '今天练什么？' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '查看拼音图' })).not.toBeInTheDocument()
  expect(screen.queryByRole('link', { name: '回到首页' })).not.toBeInTheDocument()
  expect(screen.queryByText('听清发音，从相近的拼音里找出答案。')).not.toBeInTheDocument()
  expect(screen.queryByText('当前支持')).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: '听音选字母' }).querySelector('img')).toHaveAttribute('src', '/cards/listen.png')
  expect(screen.queryByRole('button', { name: '开始答题' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '继续答题' })).not.toBeInTheDocument()
})

it('tags long pinyin options so they can shrink inside the card', async () => {
  useDemoQuizStore.setState({ sessions: { '1:listen': {
    id: 'long-options', childId: 1, type: 'listen', status: 'ready', verified: true, error: '', position: 1,
    entries: [{ question: quiz('listen', 99, [
      { id: '1', label: 'an' }, { id: '2', label: 'en' }, { id: '3', label: 'iu' }, { id: '4', label: 'ang' },
    ]) }],
  } } })
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: 'ang' })).toHaveAttribute('data-chars', '3')
  expect(screen.getByRole('button', { name: 'an' })).toHaveAttribute('data-chars', '2')
  expect(screen.getByRole('button', { name: 'iu' })).toHaveAttribute('data-chars', '2')
})

it('opens the question preview as a fullscreen child quiz', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: 'b' })).toBeInTheDocument()
  expect(screen.queryByRole('link', { name: '回到首页' })).not.toBeInTheDocument()
  expect(document.querySelector('.topbar')).not.toBeInTheDocument()
  expect(document.querySelector('.app-shell')).toHaveClass('immersive')
  expect(screen.getByRole('link', { name: '退出练习' })).toHaveAttribute('href', '/')
  expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', '/practice/type/listen/2')
  expect(screen.getByLabelText('上一题')).toHaveAttribute('aria-disabled', 'true')
  expect(screen.getByRole('region', { name: '当前题目' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: '听一听，选出你听到的拼音' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '听一听' })).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: '播放读音' })).toBeInTheDocument()
  expect(screen.getByRole('progressbar', { name: '听音选字母，0 / 4' })).toHaveTextContent('0 / 4')
})

it('keeps next and previous questions in the same type', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/listen/2']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: 'd' })).toBeInTheDocument()
  expect(screen.getByRole('progressbar', { name: '听音选字母，0 / 4' })).toHaveTextContent('0 / 4')
  expect(screen.getByRole('button', { name: '播放读音' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'b' })).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: '上一题' })).toHaveAttribute('href', '/practice/type/listen/1')
  expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', '/practice/type/listen/3')
})

it('asks for pinyin from a single character', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/inword']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByText('播')).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: '听一听，这个字里藏着哪个拼音？' })).toBeInTheDocument()
  expect(screen.getByRole('progressbar', { name: '字中找拼音，0 / 4' })).toHaveTextContent('0 / 4')
  expect(screen.queryByText('bō')).not.toBeInTheDocument()
  expect(screen.queryByText('广')).not.toBeInTheDocument()
  expect(screen.queryByText('广播')).not.toBeInTheDocument()
})

it('does not reveal the answer while practicing', async () => {
  const user = (await import('@testing-library/user-event')).default.setup()
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  await user.click(await screen.findByRole('button', { name: 'b' }))
  expect(screen.queryByText(/答对啦/)).not.toBeInTheDocument()
  expect(screen.queryByText(/再听一听/)).not.toBeInTheDocument()
  expect(document.querySelector('.correct-answer')).toBeNull()
  expect(document.querySelector('.wrong-answer')).toBeNull()
})

it('moves to the next question after an answer is picked', async () => {
  const user = (await import('@testing-library/user-event')).default.setup()
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  await user.click(await screen.findByRole('button', { name: 'p' }))
  expect(screen.getByRole('progressbar', { name: '听音选字母，1 / 4' })).toBeInTheDocument()
  expect(await screen.findByRole('button', { name: 'd' })).toBeInTheDocument()
  expect(screen.getByRole('progressbar', { name: '听音选字母，1 / 4' })).toHaveTextContent('1 / 4')
  expect(screen.queryByText(/答对啦/)).not.toBeInTheDocument()
})

it('opens the result list after the last answer is picked', async () => {
  const user = (await import('@testing-library/user-event')).default.setup()
  render(<MemoryRouter initialEntries={['/practice/type/listen/4']}><AppRoutes /></MemoryRouter>)
  await user.click(await screen.findByRole('button', { name: 'j' }))
  expect(await screen.findByRole('heading', { name: '答题结果' })).toBeInTheDocument()
  expect(screen.getAllByRole('listitem')).toHaveLength(4)
})

it('opens a result list after the last question', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/listen/4']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: 'j' })).toBeInTheDocument()
  expect(screen.getByRole('progressbar', { name: '听音选字母，0 / 4' })).toHaveTextContent('0 / 4')
  expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', '/practice/type/listen/result')
  expect(screen.getByRole('link', { name: '上一题' })).toHaveAttribute('href', '/practice/type/listen/3')
})

it('shows a result list for a finished demo', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/listen/result']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('heading', { name: '答题结果' })).toBeInTheDocument()
  expect(screen.getAllByRole('listitem')).toHaveLength(4)
  expect(screen.getByRole('link', { name: '回到首页' })).toHaveAttribute('href', '/')
  expect(screen.queryByText(/答对啦！你已经会做这种题了/)).not.toBeInTheDocument()
})

it('uses the same speaker icon on audio options', async () => {
  render(<MemoryRouter initialEntries={['/practice/type/shape']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: '播放读音 1' })).toBeInTheDocument()
  expect(screen.queryByText('🔊')).not.toBeInTheDocument()
  expect(document.querySelectorAll('.audio-option [data-icon="speaker"]')).toHaveLength(4)
  expect(screen.getByRole('button', { name: '播放读音 1' }).querySelectorAll('.wave')).toHaveLength(3)
  expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', '/practice/type/shape/2')
})

it('lets kids preview audio options without answering', async () => {
  const user = (await import('@testing-library/user-event')).default.setup()
  render(<MemoryRouter initialEntries={['/practice/type/blend']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: '选择读音 1' })).toBeDisabled()
  await user.click(screen.getByRole('button', { name: '播放读音 1' }))
  expect(screen.getByRole('progressbar', { name: '声韵拼读，0 / 4' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '选择读音 1' })).toBeEnabled()
  expect(screen.getByRole('button', { name: '选择读音 2' })).toBeDisabled()
})

it('advances audio questions only after a heard option is chosen', async () => {
  const user = (await import('@testing-library/user-event')).default.setup()
  render(<MemoryRouter initialEntries={['/practice/type/blend']}><AppRoutes /></MemoryRouter>)
  await user.click(await screen.findByRole('button', { name: '播放读音 2' }))
  await user.click(screen.getByRole('button', { name: '选择读音 2' }))
  expect(await screen.findByRole('progressbar', { name: '声韵拼读，1 / 4' })).toBeInTheDocument()
})

it('shows a retryable message when quiz generate returns HTML', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => new Response('<html>404 page not found</html>', {
    status: 404,
    headers: { 'Content-Type': 'text/plain; charset=utf-8' },
  })))
  render(<MemoryRouter initialEntries={['/practice/type/listen']}><AppRoutes /></MemoryRouter>)
  expect(await screen.findByText('出题没有成功，点下面再试一次')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '再试一次' })).toBeInTheDocument()
})

it('closes the pinyin map back to home', () => {
  render(<MemoryRouter initialEntries={['/map']}><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('heading', { name: '我的拼音图' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: '关闭' })).toHaveAttribute('href', '/')
})

it('offers a fresh practice when an unanswered legacy blend has no recorded media', async () => {
 const legacy={...banks.blend[0],options:banks.blend[0].options.map(({speechUrl: _audio,...option})=>option)}
 useDemoQuizStore.setState({sessions:{'1:blend':{id:'legacy',childId:1,type:'blend',status:'ready',verified:true,error:'',position:1,entries:[{question:legacy}]}}})
 render(<MemoryRouter initialEntries={['/practice/type/blend']}><AppRoutes/></MemoryRouter>)
 expect(await screen.findByText('这份旧题未保存读音，请重新练习')).toBeInTheDocument()
 await (await import('@testing-library/user-event')).default.click(screen.getByRole('button',{name:'重新练习'}))
 expect(await screen.findByRole('button',{name:'播放读音 1'})).toBeInTheDocument()
 expect(legacy.options.every(option=>!('speechUrl' in option))).toBe(true)
 expect(vi.mocked(fetch).mock.calls.some(([url])=>String(url).endsWith('/answer'))).toBe(false)
})

it('protects a pending submission elsewhere in a legacy session from restart', async () => {
 const legacy={...banks.blend[0],options:banks.blend[0].options.map(({speechUrl: _audio,...option})=>option)}
 const pending={clientId:'keep-this-payload',optionId:'1',costMs:77}
 useDemoQuizStore.setState({sessions:{'1:blend':{id:'legacy-pending',childId:1,type:'blend',status:'ready',verified:true,error:'',position:1,entries:[{question:legacy},{question:banks.blend[1],pending,error:'上次提交尚未确认，请重试本次提交'}]}}})
 render(<MemoryRouter initialEntries={['/practice/type/blend/1']}><AppRoutes/></MemoryRouter>)
 expect(await screen.findByRole('button',{name:'返回待确认题目'})).toBeInTheDocument()
 expect(screen.queryByRole('button',{name:'重新练习'})).not.toBeInTheDocument()
 expect(useDemoQuizStore.getState().sessions['1:blend']?.entries[1].pending).toEqual(pending)
})
