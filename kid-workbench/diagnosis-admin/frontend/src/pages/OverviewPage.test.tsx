import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { OverviewPage } from './OverviewPage'

const overview = {
  child: { id: 1, name: '卢沁一', grade: '大班' },
  headline: '识字能认、手写偏弱；英语听音不稳。',
  subjects: [
    {
      code: 'literacy',
      name: '识字',
      icon: '',
      total: 1,
      counts: { not_started: 0, learning: 1, shaky: 0, mastered: 0, review_due: 0 },
      health: 'watch',
      summary: '识字有待稳住的知识点',
    },
  ],
  urgent: [
    {
      kp_id: 20,
      title: 'Hello!',
      subject_code: 'english',
      subject_name: '英语',
      module_name: '问候',
      status: 'shaky',
      accuracy: 0.25,
      wrong_count: 6,
      reason: '反复出错，优先补练',
    },
  ],
}

function renderOverview() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('OverviewPage', () => {
  afterEach(() => vi.restoreAllMocks())

  it('renders the clinical headline and urgent weak point', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(overview), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )

    renderOverview()

    expect(await screen.findByText('识字能认、手写偏弱；英语听音不稳。')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '诊断总览' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Hello!/ })).toHaveAttribute('href', '/knowledge-points/20')
    expect(screen.getByRole('link', { name: /识字/ })).toHaveAttribute('href', '/subjects/literacy')
  })

  it('does not render the game subject card', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({
        ...overview,
        subjects: [
          ...overview.subjects,
          {
            code: 'game',
            name: '游戏',
            icon: '',
            total: 8,
            counts: { not_started: 8, learning: 0, shaky: 0, mastered: 0, review_due: 0 },
            health: 'ok',
            summary: '游戏目前比较稳',
          },
        ],
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )

    renderOverview()

    expect(await screen.findByRole('heading', { name: '诊断总览' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: '游戏' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /游戏目前比较稳/ })).not.toBeInTheDocument()
  })
})
