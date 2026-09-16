import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AppRoutes } from '../App'
import { useDemoAnswerStore } from '../store/demoAnswerStore'
import { useLiveQuizStore } from '../store/liveQuizStore'

afterEach(() => {
  cleanup()
  useDemoAnswerStore.setState({ solved: {} })
  useLiveQuizStore.getState().invalidate()
  vi.unstubAllGlobals()
})

function stubQuizFail() {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))
}

async function renderType(path: string) {
  const choice = path.startsWith('/question-types/choice')
  const result = path.endsWith('/result')
  if (choice && result) useLiveQuizStore.getState().useFallback()
  if (choice && !result) stubQuizFail()
  render(<MemoryRouter initialEntries={[path]}><AppRoutes /></MemoryRouter>)
  if (choice && !result) fireEvent.click(await screen.findByRole('button', { name: '用示例题' }))
}

const prompts = [
  ['choice', '哪种动物的脚掌适合在水里游泳？'],
  ['match', '把动物和它们的生活环境连在一起。'],
  ['sequence', '把植物的生长过程排成正确顺序。'],
  ['label', '把“根”标到植物的正确位置。'],
] as const

describe('question type answering pages', () => {
  it.each(prompts)('shows the %s question like a kid practice screen', async (slug, prompt) => {
    await renderType(`/question-types/${slug}`)
    expect(screen.getByRole('heading', { level: 1, name: prompt })).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: /第 1 题 \/ \d+ 题/ })).toBeInTheDocument()
    expect(screen.getByText(/^\d+ \/ \d+$/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '退出' })).toHaveAttribute('href', '/')
    expect(screen.queryByText('退出')).not.toBeInTheDocument()
    expect(screen.queryByText('上一题')).not.toBeInTheDocument()
    expect(screen.queryByText('下一题')).not.toBeInTheDocument()
    expect(screen.queryByText(/^第 \d+ 题 \/ \d+ 题$/)).not.toBeInTheDocument()
    expect(document.querySelector('.progress-track')).toBeInTheDocument()
    expect(screen.queryByText(/QUESTION TYPE/)).not.toBeInTheDocument()
    expect(screen.queryByText('现场小题 · 直接在画面中作答')).not.toBeInTheDocument()
    expect(screen.queryByText('题型实验室')).not.toBeInTheDocument()
    expect(screen.queryByText('单项选择')).not.toBeInTheDocument()
    expect(screen.queryByText('操作提示')).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /下一题型/ })).not.toBeInTheDocument()
    expect(screen.queryByText('小小发现局')).not.toBeInTheDocument()
  })

  it('puts previous and next question controls in the top bar', async () => {
    await renderType('/question-types/choice')
    expect(screen.getByRole('button', { name: '上一题' })).toBeDisabled()
    expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', expect.stringMatching(/^\/question-types\/choice\//))
    expect(screen.getByRole('link', { name: '下一题' })).not.toHaveAttribute('href', '/question-types/match')
  })

  it('shows the question count on the progress bar', async () => {
    const user = userEvent.setup()
    await renderType('/question-types/choice')
    expect(screen.getByText('1 / 4')).toBeInTheDocument()
    await user.click(screen.getByRole('link', { name: '下一题' }))
    expect(screen.getByText('2 / 4')).toBeInTheDocument()
  })

  it('keeps next inside the same question type', async () => {
    const user = userEvent.setup()
    await renderType('/question-types/choice')
    const firstPrompt = screen.getByRole('heading', { level: 1 }).textContent
    await user.click(screen.getByRole('link', { name: '下一题' }))
    expect(screen.getByRole('heading', { level: 1 }).textContent).not.toBe(firstPrompt)
    expect(screen.queryByRole('button', { name: '连接北极熊和冰原' })).not.toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: /选择：/ })).toHaveLength(3)
    expect(screen.getByRole('link', { name: '上一题' })).toHaveAttribute('href', '/question-types/choice')
  })

  it('opens the result page from the last question of that type', async () => {
    await renderType('/question-types/choice/fish-gills')
    expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', '/question-types/choice/result')
    expect(screen.queryByRole('link', { name: '完成' })).not.toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: /选择：/ })).toHaveLength(3)
  })

  it('shows a result summary after the last science question', async () => {
    await renderType('/question-types/choice/result')
    expect(screen.getByRole('heading', { name: '答题结果' })).toBeInTheDocument()
    expect(screen.getByText(/答对 \d+ \/ 4 题/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '再练一次' })).toHaveAttribute('href', '/question-types/choice')
    expect(screen.getByRole('link', { name: '回到首页' })).toHaveAttribute('href', '/')
  })

  it('puts the match question above three animals, then the habitats', () => {
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    const heading = screen.getByRole('heading', { level: 1, name: '把动物和它们的生活环境连在一起。' })
    const bear = screen.getByRole('button', { name: '选择动物：北极熊' })
    const ice = screen.getByRole('button', { name: '选择环境：冰原' })
    expect(screen.getAllByRole('button', { name: /选择动物：/ })).toHaveLength(3)
    expect(screen.getAllByRole('button', { name: /选择环境：/ })).toHaveLength(3)
    expect(heading.compareDocumentPosition(bear) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0)
    expect(bear.compareDocumentPosition(ice) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0)
  })

  it('puts the choice question above the duck, then the options', async () => {
    await renderType('/question-types/choice')
    const heading = screen.getByRole('heading', { level: 1, name: '哪种动物的脚掌适合在水里游泳？' })
    const visual = screen.getByTestId('question-visual')
    const duck = screen.getByRole('button', { name: '选择：鸭子' })
    expect(heading.compareDocumentPosition(visual) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0)
    expect(visual.compareDocumentPosition(duck) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0)
  })

  it('shows immediate feedback in the observation question', async () => {
    const user = userEvent.setup()
    await renderType('/question-types/choice')
    await user.click(screen.getByRole('button', { name: '选择：鸭子' }))
    expect(screen.getByRole('status')).toHaveTextContent('找对了')
    expect(screen.getByRole('link', { name: '下一题' })).toHaveAttribute('href', expect.stringMatching(/^\/question-types\/choice\//))
    expect(screen.getByRole('link', { name: '下一题' })).not.toHaveAttribute('href', '/question-types/match')
  })

  it('uses distinct interaction mechanics instead of repeating choice buttons', () => {
    const cases = [
      ['match', 'button', '选择动物：北极熊'],
      ['sequence', 'listitem', '拖动排序：种子'],
      ['label', 'button', '标注植物的根'],
    ] as const

    for (const [slug, role, name] of cases) {
      cleanup()
      render(<MemoryRouter initialEntries={[`/question-types/${slug}`]}><AppRoutes /></MemoryRouter>)
      expect(screen.getByRole(role, { name })).toBeInTheDocument()
    }
  })

  it('shows three animals above three habitats on the match board', () => {
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    expect(screen.getByRole('button', { name: '选择动物：青蛙' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '选择动物：北极熊' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '选择动物：骆驼' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '选择环境：热带雨林' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '选择环境：冰原' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '选择环境：沙漠' })).toBeInTheDocument()
  })

  it('draws a line after connecting an animal to a habitat', async () => {
    const user = userEvent.setup()
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    await user.click(screen.getByRole('button', { name: '选择动物：北极熊' }))
    await user.click(screen.getByRole('button', { name: '选择环境：冰原' }))
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
  })

  it('solves the match board after all three pairs are connected', async () => {
    const user = userEvent.setup()
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    await user.click(screen.getByRole('button', { name: '选择动物：北极熊' }))
    await user.click(screen.getByRole('button', { name: '选择环境：冰原' }))
    await user.click(screen.getByRole('button', { name: '选择动物：青蛙' }))
    await user.click(screen.getByRole('button', { name: '选择环境：热带雨林' }))
    await user.click(screen.getByRole('button', { name: '选择动物：骆驼' }))
    await user.click(screen.getByRole('button', { name: '选择环境：沙漠' }))
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
    expect(screen.getByTestId('match-line-frog-forest')).toBeInTheDocument()
    expect(screen.getByTestId('match-line-camel-desert')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('连对了')
  })

  it('does not retarget a finished pair when another habitat is tapped', async () => {
    const user = userEvent.setup()
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    const bear = screen.getByRole('button', { name: '选择动物：北极熊' })
    await user.click(bear)
    await user.click(screen.getByRole('button', { name: '选择环境：冰原' }))
    expect(bear).toHaveAttribute('aria-pressed', 'false')
    await user.click(screen.getByRole('button', { name: '选择环境：沙漠' }))
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
    expect(screen.queryByTestId('match-line-bear-desert')).not.toBeInTheDocument()
  })

  it('keeps the polar bear line after a drag-connect to the camel', () => {
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    const bear = screen.getByRole('button', { name: '选择动物：北极熊' })
    const ice = screen.getByRole('button', { name: '选择环境：冰原' })
    const camel = screen.getByRole('button', { name: '选择动物：骆驼' })
    const desert = screen.getByRole('button', { name: '选择环境：沙漠' })
    let hit: Element | null = ice
    document.elementFromPoint = () => hit
    document.elementsFromPoint = () => (hit ? [hit] : [])

    fireEvent.pointerDown(bear, { pointerId: 1, clientX: 20, clientY: 10 })
    fireEvent.pointerMove(bear, { pointerId: 1, clientX: 20, clientY: 40 })
    fireEvent.pointerUp(bear, { pointerId: 1, clientX: 20, clientY: 40 })
    fireEvent.click(bear)
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
    expect(bear).toHaveAttribute('aria-pressed', 'false')

    fireEvent.click(desert)
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
    expect(screen.queryByTestId('match-line-bear-desert')).not.toBeInTheDocument()

    hit = desert
    fireEvent.pointerDown(camel, { pointerId: 1, clientX: 80, clientY: 10 })
    fireEvent.pointerMove(camel, { pointerId: 1, clientX: 80, clientY: 40 })
    fireEvent.pointerUp(camel, { pointerId: 1, clientX: 80, clientY: 40 })
    fireEvent.click(camel)
    fireEvent.click(desert)
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
    expect(screen.getByTestId('match-line-camel-desert')).toBeInTheDocument()
  })

  it('connects when the pointer is released over a habitat, even with a short drag', () => {
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    const bear = screen.getByRole('button', { name: '选择动物：北极熊' })
    const ice = screen.getByRole('button', { name: '选择环境：冰原' })
    document.elementFromPoint = () => ice
    document.elementsFromPoint = () => [ice]
    fireEvent.pointerDown(bear, { pointerId: 1, clientX: 20, clientY: 10 })
    fireEvent.pointerUp(bear, { pointerId: 1, clientX: 22, clientY: 18 })
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
  })

  it('does not drop an existing line when a later drag misses a habitat', () => {
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    const bear = screen.getByRole('button', { name: '选择动物：北极熊' })
    const ice = screen.getByRole('button', { name: '选择环境：冰原' })
    const camel = screen.getByRole('button', { name: '选择动物：骆驼' })
    document.elementFromPoint = () => ice
    document.elementsFromPoint = () => [ice]
    fireEvent.pointerDown(bear, { pointerId: 1, clientX: 20, clientY: 10 })
    fireEvent.pointerMove(bear, { pointerId: 1, clientX: 20, clientY: 40 })
    fireEvent.pointerUp(bear, { pointerId: 1, clientX: 20, clientY: 40 })
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()

    document.elementFromPoint = () => null
    document.elementsFromPoint = () => []
    fireEvent.pointerDown(camel, { pointerId: 1, clientX: 80, clientY: 10 })
    fireEvent.pointerMove(camel, { pointerId: 1, clientX: 80, clientY: 80 })
    fireEvent.pointerUp(camel, { pointerId: 1, clientX: 80, clientY: 80 })
    expect(screen.getByTestId('match-line-bear-ice')).toBeInTheDocument()
    expect(screen.queryByTestId('match-line-camel-desert')).not.toBeInTheDocument()
  })

  it('snaps to a habitat when the pointer is over the card even if elementFromPoint misses', () => {
    render(<MemoryRouter initialEntries={['/question-types/match']}><AppRoutes /></MemoryRouter>)
    const camel = screen.getByRole('button', { name: '选择动物：骆驼' })
    const desert = screen.getByRole('button', { name: '选择环境：沙漠' })
    desert.getBoundingClientRect = () => ({ x: 100, y: 200, left: 100, top: 200, right: 220, bottom: 320, width: 120, height: 120, toJSON() { return {} } })
    document.elementFromPoint = () => null
    document.elementsFromPoint = () => []
    fireEvent.pointerDown(camel, { pointerId: 1, clientX: 150, clientY: 40 })
    fireEvent.pointerMove(camel, { pointerId: 1, clientX: 150, clientY: 250 })
    fireEvent.pointerUp(camel, { pointerId: 1, clientX: 150, clientY: 250 })
    expect(screen.getByTestId('match-line-camel-desert')).toBeInTheDocument()
  })

  it('sorts the sequence by dragging, without arrow buttons on the cards', () => {
    render(<MemoryRouter initialEntries={['/question-types/sequence']}><AppRoutes /></MemoryRouter>)
    expect(screen.queryByRole('button', { name: '向右移动种子' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '向左移动发芽' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '向右移动发芽' })).not.toBeInTheDocument()

    const seed = screen.getByRole('listitem', { name: '拖动排序：种子' })
    const sprout = screen.getByRole('listitem', { name: '拖动排序：发芽' })
    fireEvent.mouseDown(seed)
    fireEvent.mouseEnter(sprout)
    fireEvent.mouseUp(sprout)

    expect(screen.getByRole('status')).toHaveTextContent('顺序正确')
  })

  it('reorders sequence cards with arrow keys when a card is focused', async () => {
    const user = userEvent.setup()
    render(<MemoryRouter initialEntries={['/question-types/sequence']}><AppRoutes /></MemoryRouter>)
    screen.getByRole('listitem', { name: '拖动排序：发芽' }).focus()
    await user.keyboard('{ArrowRight}')
    const items = screen.getAllByRole('listitem')
    expect(items[0]).toHaveAccessibleName('拖动排序：种子')
    expect(items[1]).toHaveAccessibleName('拖动排序：发芽')
  })

  it.each([
    ['choice', 'button', '选择：鸭子'],
    ['match', 'button', '选择动物：北极熊'],
    ['sequence', 'list', '植物生长排序'],
    ['label', 'button', '标注植物的根'],
  ] as const)('places the %s interaction on the answering stage', async (slug, role, name) => {
    await renderType(`/question-types/${slug}`)
    const canvas = screen.getByTestId('question-canvas')
    expect(within(canvas).getByRole(role, { name })).toBeInTheDocument()
    expect(canvas).toHaveClass('question-canvas')
  })
})
