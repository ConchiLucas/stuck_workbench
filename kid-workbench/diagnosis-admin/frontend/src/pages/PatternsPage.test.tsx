import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PatternsPage } from './PatternsPage'

describe('PatternsPage', () => {
  afterEach(() => vi.restoreAllMocks())

  it('lists error-pattern kinds from the read-only API', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({
        patterns: [
          {
            kind: 'skill_imbalance',
            title: '只会其中一种题型',
            detail: '一：看字选义已掌握，手写仍弱',
            kp_id: 10,
            kp_title: '一',
          },
        ],
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <PatternsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(await screen.findByText('技能不平衡')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /只会其中一种题型/ })).toHaveAttribute('href', '/knowledge-points/10')
  })
})
