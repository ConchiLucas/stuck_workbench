import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { ResultPage } from './ResultPage'

vi.mock('../api/math', () => ({ mathApi: {
  plan: vi.fn().mockResolvedValue({ plan: { id: 9, status: 'completed', targetCount: 10, doneCount: 10, correctCount: 8, stars: 2, durationSec: 96 }, items: [] }),
  home: vi.fn().mockResolvedValue({ child: { id: 1, name: '豆豆', flowers: 15 }, plan: null, dueCount: 1, modules: [] }),
} }))

it('shows stars, accuracy and reward after practice', async () => {
  render(<MemoryRouter initialEntries={['/result/9']}><Routes><Route path="result/:planId" element={<ResultPage />} /></Routes></MemoryRouter>)
  expect(await screen.findByRole('heading', { name: '练习完成' })).toBeInTheDocument()
  expect(screen.getByText('8 / 10')).toBeInTheDocument()
  expect(screen.getByLabelText('获得 2 颗星')).toBeInTheDocument()
  expect(screen.getByText(/15 朵/)).toBeInTheDocument()
})
