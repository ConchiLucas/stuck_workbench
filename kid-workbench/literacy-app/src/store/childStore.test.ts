import { afterEach, beforeEach, expect, it, vi } from 'vitest'
beforeEach(() => { localStorage.removeItem('literacy-child-id'); vi.resetModules() })
afterEach(() => { localStorage.removeItem('literacy-child-id'); vi.unstubAllEnvs() })
it('uses the configured acceptance child when no saved child exists', async () => {
  vi.stubEnv('VITE_CHILD_ID','2')
  const {useChildStore} = await import('./childStore')
  expect(useChildStore.getState().childId).toBe(2)
})
it('keeps the saved child ahead of the environment default', async () => {
  vi.stubEnv('VITE_CHILD_ID','2'); localStorage.setItem('literacy-child-id','7')
  const {useChildStore} = await import('./childStore')
  expect(useChildStore.getState().childId).toBe(7)
})
it('retains the production default without acceptance configuration', async () => {
  vi.stubEnv('VITE_CHILD_ID',undefined)
  const {useChildStore} = await import('./childStore')
  expect(useChildStore.getState().childId).toBe(1)
})
