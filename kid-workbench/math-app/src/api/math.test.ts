import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './client'
import { generateMathQuizSet, learningAudioURL } from './math'

afterEach(() => vi.unstubAllGlobals())

describe('math urls', () => {
  it('builds only the controlled learning audio path', () => {
    expect(learningAudioURL(1, 42, 'find')).toBe('/api/v1/children/1/math/items/42/audio/find.mp3')
  })
})

it('generateMathQuizSet asks for four unique targets', async () => {
  const seen: number[][] = []
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (_url: string, init?: RequestInit) => {
    const body = JSON.parse(String(init?.body))
    seen.push(body.excludeTargetIds ?? [])
    const n = seen.length
    return new Response(JSON.stringify({
      data: {
        instanceId: `q${n}`, type: body.type, stem: `${n} + 1 = ?`, targetId: n,
        visual: { kind: 'add', a: n, b: 1 },
        options: [{ id: 0, label: '1' }, { id: 1, label: '2' }, { id: 2, label: '3' }, { id: 3, label: '4' }],
        answerIndex: 1,
      },
      error: null,
    }))
  }))
  const questions = await generateMathQuizSet('equation')
  expect(questions).toHaveLength(4)
  expect(seen[0]).toEqual([])
  expect(seen[3]).toEqual([1, 2, 3])
})

it('non-json generate responses become 无法识别的内容', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))
  await expect(api.post('/math/quiz/generate', { type: 'equation' })).rejects.toThrow('服务返回了无法识别的内容')
})
