import { beforeEach, describe, expect, it } from 'vitest'
import { createPendingAnswerStore } from './pendingAnswerStore'

describe('pending answer ids', () => {
  beforeEach(() => localStorage.clear())

  it('keeps one uuid across recreation until acknowledged', () => {
    const firstStore = createPendingAnswerStore(localStorage)
    const first = firstStore.getState().clientId(7, 9, 1)
    expect(firstStore.getState().clientId(7, 9, 1)).toBe(first)

    const recreated = createPendingAnswerStore(localStorage)
    expect(recreated.getState().clientId(7, 9, 1)).toBe(first)
    expect(recreated.getState().clientId(7, 9, 2)).not.toBe(first)

    recreated.getState().acknowledge(7, 9, 1)
    expect(createPendingAnswerStore(localStorage).getState().clientId(7, 9, 1)).not.toBe(first)
  })
})
