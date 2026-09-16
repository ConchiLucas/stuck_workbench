import { describe, expect, it } from 'vitest'
import { demoToExample, questionDemos } from './questionDemos'

describe('question demos bind real media', () => {
  it('does not use emoji or browser-only placeholders for pictures and speech', () => {
    for (const [kind, items] of Object.entries(questionDemos)) {
      for (const demo of items) {
        const example = demoToExample(kind as 'audio-choice', demo)
        if (example.cue) {
          expect(example.cue.startsWith('/api/v1/english/words/')).toBe(true)
        }
        if (example.speechUrl) {
          expect(example.speechUrl.startsWith('/api/v1/english/')).toBe(true)
        }
        if (kind === 'card-builder') {
          expect(example.speechUrl).toMatch(/\/sentences\//)
        }
        for (const option of example.options ?? []) {
          expect(option.picture).toMatch(/^\/api\/v1\/english\/words\/\d+\/sense\.png$/)
          expect(option.picture).not.toMatch(/[\u{1F300}-\u{1FAFF}]/u)
        }
      }
    }
  })
})
