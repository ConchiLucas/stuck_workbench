import { afterEach, expect, it, vi } from 'vitest'
import { generateScienceQuizSet } from './science'

afterEach(() => vi.unstubAllGlobals())

it('generateScienceQuizSet asks for four unique targets', async () => {
  const seen: number[][] = []
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (_url: string, init?: RequestInit) => {
    const body = JSON.parse(String(init?.body))
    seen.push(body.excludeTargetIds ?? [])
    const n = seen.length
    return new Response(JSON.stringify({
      data: {
        instanceId: `q${n}`, type: 'choice', stem: `stem-${n}`, targetId: n,
        visual: { kind: 'emoji', emoji: '🦆' },
        options: [{ id: 0, label: 'A' }, { id: 1, label: 'B' }, { id: 2, label: 'C' }],
        answerIndex: 1,
      },
      error: null,
    }))
  }))
  const questions = await generateScienceQuizSet()
  expect(questions).toHaveLength(4)
  expect(seen[0]).toEqual([])
  expect(seen[3]).toEqual([1, 2, 3])
})
