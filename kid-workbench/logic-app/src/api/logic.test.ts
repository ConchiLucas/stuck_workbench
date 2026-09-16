import { afterEach, expect, it, vi } from 'vitest'
import { api } from './client'
import { generateLogicQuizSet } from './logic'

afterEach(() => vi.unstubAllGlobals())

it('generateLogicQuizSet asks for four unique targets', async () => {
  const seen: number[][] = []
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (_url: string, init?: RequestInit) => {
    const body = JSON.parse(String(init?.body))
    seen.push(body.excludeTargetIds ?? [])
    const n = seen.length
    return new Response(JSON.stringify({
      data: {
        instanceId: `q${n}`, type: body.type, stem: '下一个是哪个？', targetId: n,
        visual: { kind: 'seq', items: ['🔴', '🔵'] },
        options: [{ id: 0, emoji: '🔴' }, { id: 1, emoji: '🟢' }, { id: 2, emoji: '🟡' }, { id: 3, emoji: '⬛' }],
        answerIndex: 0,
      },
      error: null,
    }))
  }))
  const questions = await generateLogicQuizSet('pattern')
  expect(questions).toHaveLength(4)
  expect(seen[0]).toEqual([])
  expect(seen[3]).toEqual([1, 2, 3])
})

it('non-json generate responses become 无法识别的内容', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))
  await expect(api.post('/logic/quiz/generate', { type: 'pattern' })).rejects.toThrow('服务返回了无法识别的内容')
})
