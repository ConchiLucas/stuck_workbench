import { afterEach, expect, it, vi } from 'vitest'
import { ApiError, api } from './client'

afterEach(() => vi.unstubAllGlobals())

it('unwraps the API envelope', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { ok: true }, error: null }), { status: 200 })))
  await expect(api.get<{ ok: boolean }>('/health')).resolves.toEqual({ ok: true })
})

it('preserves API error status and code', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: null, error: { code: 'no_questions', message: '暂无题目' } }), { status: 409 })))
  await expect(api.post('/plans')).rejects.toMatchObject({ status: 409, code: 'no_questions', message: '暂无题目' } satisfies Partial<ApiError>)
})
