import { afterEach, expect, test, vi } from 'vitest'
import { api } from './client'
import { generateEnglishQuizSet } from './english'

afterEach(() => vi.unstubAllGlobals())

test('non-json responses become 无法识别的内容', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))
  await expect(api('/api/v1/english/quiz/generate', { method: 'POST', body: '{}' })).rejects.toThrow('服务返回了无法识别的内容')
})

test('generateEnglishQuizSet asks for four unique targets', async () => {
  const seen: number[][] = []
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (_url: string, init?: RequestInit) => {
    const body = JSON.parse(String(init?.body))
    seen.push(body.excludeTargetIds ?? [])
    const n = seen.length
    return new Response(JSON.stringify({
      data: {
        instanceId: `q${n}`, type: body.type, stem: 'stem', targetId: n,
        speechText: 'apple', visual: { kind: 'sound' },
        options: [{ id: n, label: '苹果' }, { id: 10 + n, label: '香蕉' }, { id: 20 + n, label: '小狗' }, { id: 30 + n, label: '小鸟' }],
        answerIndex: 0,
      },
      error: null,
    }))
  }))
  const questions = await generateEnglishQuizSet('listen')
  expect(questions).toHaveLength(4)
  expect(seen[0]).toEqual([])
  expect(seen[3]).toEqual([1, 2, 3])
})
