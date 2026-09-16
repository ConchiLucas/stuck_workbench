import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AppRoutes } from './App'
import { practiceTypes } from './content/typePracticeBanks'
import { useLiveQuizStore } from './store/liveQuizStore'
import { useTypePracticeStore } from './store/typePracticeStore'

function stubQuizFail() {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))
}

async function renderLocal(path: string) {
  await renderLocalAndContainer(path)
}

async function renderLocalAndContainer(path: string) {
  stubQuizFail()
  const view = render(<MemoryRouter initialEntries={[path]}><AppRoutes /></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button', { name: '用示例题' }))
  return view
}

afterEach(() => {
  cleanup()
  useTypePracticeStore.setState({ picks: {}, sequences: {}, wrongs: {}, rejected: {} })
  for (const type of practiceTypes) useLiveQuizStore.getState().invalidate(type)
  vi.unstubAllGlobals()
})

describe('logic app routes', () => {
  it('renders six clay illustrated logic type cards', () => {
    render(<MemoryRouter><AppRoutes /></MemoryRouter>)
    expect(screen.getByRole('link', { name: '找规律' })).toHaveAttribute('href', '/practice/type/pattern')
    expect(screen.getByRole('link', { name: '分类' })).toHaveAttribute('href', '/practice/type/classify')
    expect(screen.getByRole('link', { name: '排序' })).toHaveAttribute('href', '/practice/type/order')
    expect(screen.getByRole('link', { name: '图形推理' })).toHaveAttribute('href', '/practice/type/shape_reason')
    expect(screen.getByRole('link', { name: '找不同' })).toHaveAttribute('href', '/practice/type/diff')
    expect(screen.getByRole('link', { name: '比较' })).toHaveAttribute('href', '/practice/type/compare')
    expect(screen.getByRole('link', { name: '找规律' }).querySelector('img')).toHaveAttribute('src', '/cards/pattern.png')
    expect(screen.getByRole('link', { name: '比较' }).querySelector('img')).toHaveAttribute('src', '/cards/compare.png')
    expect(screen.getByRole('link', { name: '找不同' })).not.toHaveClass('is-wide')
  })

  it('starts pattern practice from the example bank', async () => {
    await renderLocal('/practice/type/pattern')
    expect(screen.getByRole('heading', { name: '下一个是哪个？' })).toBeInTheDocument()
    expect(screen.getAllByRole('button').length).toBeGreaterThanOrEqual(4)
    expect(screen.getByRole('progressbar')).toHaveTextContent('1 / 4')
  })

  it('shows feedback after a correct pattern pick', async () => {
    await renderLocal('/practice/type/pattern')
    fireEvent.click(screen.getByRole('button', { name: /红/ }))
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
  })

  it('splits pattern practice into a stem column and a full option grid', async () => {
    const { container } = await renderLocalAndContainer('/practice/type/pattern')
    expect(container.querySelector('.question-workspace.is-split')).not.toBeNull()
    expect(container.querySelector('.question-stem .seq-row')).not.toBeNull()
    expect(container.querySelector('.seq-hole')).not.toBeNull()
    expect(container.querySelector('.option-grid')).not.toBeNull()
  })

  it('labels the last next control as 查看结果', async () => {
    await renderLocal('/practice/type/pattern/4')
    expect(screen.getByRole('link', { name: '查看结果' })).toHaveAttribute('href', '/practice/type/pattern/result')
  })

  it('renders classify questions without a sequence hole', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => new Response(JSON.stringify({
      data: {
        instanceId: 'cl-live', type: 'classify', stem: '哪个不是动物？', targetId: 1,
        visual: {},
        options: [{ id: 0, label: '兔' }, { id: 1, label: '狗' }, { id: 2, label: '猫' }, { id: 3, label: '树' }],
        answerIndex: 3,
        example: {
          kind: 'classify', prompt: '哪个不是动物？',
          rule: { type: 'odd-one-out', dimension: 'kingdom', inGroup: 'animal', explain: '动物' },
          objects: [
            { id: 'rabbit', caption: '兔', glyph: 'rabbit', attrs: { category: 'animal' } },
            { id: 'dog', caption: '狗', glyph: 'dog', attrs: { category: 'animal' } },
            { id: 'cat', caption: '猫', glyph: 'cat', attrs: { category: 'animal' } },
            { id: 'tree', caption: '树', glyph: 'tree', attrs: { category: 'plant' } },
          ],
          options: ['rabbit', 'dog', 'cat', 'tree'], answerId: 'tree',
        },
      },
      error: null,
    }))))
    const { container } = render(<MemoryRouter initialEntries={['/practice/type/classify']}><AppRoutes /></MemoryRouter>)
    expect(await screen.findByRole('heading', { name: '哪个不是动物？' })).toBeInTheDocument()
    expect(container.querySelector('.seq-hole')).toBeNull()
    expect(container.querySelector('.classify-board')).toBeNull()
    expect(screen.getByText('点出不一样的那个')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /树/ }))
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
  })

  it('lets kids tap order items into numbered slots', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => new Response(JSON.stringify({
      data: {
        instanceId: 'od-live', type: 'order', stem: '四季的顺序是？', targetId: 2,
        visual: { kind: 'seq', items: ['🌸', '☀️', '🍂', '❄️'] },
        options: [
          { id: 0, label: '春 → 秋 → 夏 → 冬' },
          { id: 1, label: '春 → 夏 → 秋 → 冬' },
          { id: 2, label: '冬 → 秋 → 夏 → 春' },
          { id: 3, label: '夏 → 春 → 冬 → 秋' },
        ],
        answerIndex: 1,
      },
      error: null,
    }))))
    const { container } = render(<MemoryRouter initialEntries={['/practice/type/order']}><AppRoutes /></MemoryRouter>)
    expect(await screen.findByRole('heading', { name: '四季的顺序是？' })).toBeInTheDocument()
    expect(screen.getByRole('listitem', { name: '第 1 位' })).toHaveTextContent('？')
    expect(container.querySelector('.seq-hole')).toBeNull()
    for (const item of ['🌸', '☀️', '🍂', '❄️']) {
      fireEvent.click(screen.getByRole('button', { name: new RegExp(item) }))
    }
    expect(screen.getByRole('listitem', { name: '第 1 位' })).toHaveTextContent('🌸')
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
  })

  it('renders generated pattern stems when the quiz API succeeds', async () => {
    let n = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => {
      n += 1
      return new Response(JSON.stringify({
        data: {
          instanceId: `live-${n}`, type: 'pattern', stem: `第 ${n} 个规律？`, targetId: n,
          visual: { kind: 'seq', items: ['🔴', '🔵'] },
          options: [{ id: 0, emoji: '🔴' }, { id: 1, emoji: '🟢' }, { id: 2, emoji: '🟡' }, { id: 3, emoji: '⬛' }],
          answerIndex: 0,
        },
        error: null,
      }))
    }))
    render(<MemoryRouter initialEntries={['/practice/type/pattern']}><AppRoutes /></MemoryRouter>)
    expect(await screen.findByRole('heading', { name: '第 1 个规律？' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: '下一个是哪个？' })).not.toBeInTheDocument()
  })

  it('starts compare practice from the example bank', async () => {
    const { container } = await renderLocalAndContainer('/practice/type/compare')
    expect(screen.getByRole('heading', { name: '哪个更高？' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /长颈鹿/ })).toHaveAttribute('data-weight', '4')
    expect(Number(screen.getByRole('button', { name: /老鼠/ }).getAttribute('data-weight'))).toBeLessThan(4)
    expect(container.querySelector('.option-caption')).not.toBeNull()
    fireEvent.click(screen.getByRole('button', { name: /长颈鹿/ }))
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
  })
})
