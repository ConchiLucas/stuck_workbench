import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

afterEach(() => cleanup())

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal('ResizeObserver', ResizeObserverMock)
HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
  setTransform() {},
  clearRect() {},
  beginPath() {},
  moveTo() {},
  lineTo() {},
  stroke() {},
})) as unknown as typeof HTMLCanvasElement.prototype.getContext

vi.mock('hanzi-writer', () => ({
  default: {
    create: () => ({
      quiz: () => Promise.resolve(),
      cancelQuiz: () => undefined,
      animateCharacter: () => Promise.resolve({ canceled: false }),
    }),
    loadCharacterData: () => Promise.resolve({ strokes: [], medians: [] }),
  },
}))
