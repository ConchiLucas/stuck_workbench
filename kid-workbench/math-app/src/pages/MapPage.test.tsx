import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { MapPage } from './MapPage'

vi.mock('../api/math', () => ({
  useMathModules: () => ({ data: [{ code: 'add10', name: '10以内加法', orderNo: 1, itemCount: 2, stages: [{ code: 'within5', name: '5以内', itemCount: 2 }] }] }),
  useMathProgress: () => ({ data: { modules: [{ code: 'add10', name: '10以内加法', stages: [{ code: 'within5', items: [
    { kpId: 1, title: '1+1', status: 'mastered', skills: [] }, { kpId: 2, title: '2+2', status: 'review_due', skills: [] },
  ] }] }] } }),
}))

it('shows stage progress with readable status labels', () => {
  render(<MemoryRouter><MapPage /></MemoryRouter>)
  expect(screen.getByRole('heading', { name: '算数地图' })).toBeInTheDocument()
  expect(screen.getByText('已掌握')).toBeInTheDocument()
  expect(screen.getByText('该复习')).toBeInTheDocument()
})
