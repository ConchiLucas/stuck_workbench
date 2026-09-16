import { beforeEach, expect, it } from 'vitest'
import { useChildStore } from './childStore'
import { usePendingAnswerStore } from './pendingAnswerStore'

beforeEach(() => {
  localStorage.clear()
  useChildStore.setState({ childId: 1 })
  usePendingAnswerStore.setState({ ids: {} })
})

it('defaults child id to 1', () => {
  expect(useChildStore.getState().childId).toBe(1)
})

it('keeps one uuid per plan item try', () => {
  const first = usePendingAnswerStore.getState().get('9:1:1')
  expect(usePendingAnswerStore.getState().get('9:1:1')).toBe(first)
  expect(usePendingAnswerStore.getState().get('9:1:2')).not.toBe(first)
})
