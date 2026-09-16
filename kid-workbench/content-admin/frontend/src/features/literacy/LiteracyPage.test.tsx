import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LiteracyPage } from './LiteracyPage'

describe('LiteracyPage', () => {
  afterEach(() => vi.restoreAllMocks())

  it('links the heading to the literacy kid app', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      view: 'groups',
      total: 0,
      groups: [],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><LiteracyPage /></QueryClientProvider>)

    expect(await screen.findByRole('heading', { name: '识字素材' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '试做' })).not.toBeInTheDocument()
    expect(screen.queryByText('可出题')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '孩子端' })).toHaveAttribute('href', 'http://localhost:19152')
  })
})
