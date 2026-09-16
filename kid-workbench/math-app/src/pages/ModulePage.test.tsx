import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { ModulePage } from './ModulePage'

const { mutate } = vi.hoisted(() => ({ mutate: vi.fn() }))
vi.mock('../api/math', () => ({
  useMathModule: () => ({ data: { code: 'shape', name: '认识图形', orderNo: 3, itemCount: 1, stages: [{ code: 'basic-shapes', name: '基本图形', itemCount: 1 }] } }),
  useMathStage: () => ({ data: [{ kpId: 8, title: '圆形', moduleCode: 'shape', stageCode: 'basic-shapes', kind: 'shape', shape: 'circle' }] }),
  useCreatePlan: () => ({ mutate, isPending: false }),
  learningAudioURL: () => '/audio/calc.mp3',
}))

it('explores shape items and starts a scoped plan', () => {
  render(<MemoryRouter initialEntries={['/module/shape']}><Routes><Route path="module/:moduleCode" element={<ModulePage />} /></Routes></MemoryRouter>)
  expect(screen.getByRole('img', { name: /circle/ })).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: /练一练/ }))
  expect(mutate).toHaveBeenCalledWith({ kind: 'module', moduleCode: 'shape', stageCode: 'basic-shapes' }, expect.any(Object))
})
