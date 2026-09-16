import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

class FakeUtterance {
  text = ''
  lang = ''
  rate = 1
  constructor(text = '') {
    this.text = text
  }
}

Object.defineProperty(window, 'SpeechSynthesisUtterance', { configurable: true, writable: true, value: FakeUtterance })
Object.defineProperty(window, 'speechSynthesis', {
  configurable: true,
  writable: true,
  value: { cancel: () => undefined, speak: () => undefined, speaking: false },
})

afterEach(() => cleanup())
