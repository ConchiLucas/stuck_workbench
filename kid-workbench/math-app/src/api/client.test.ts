import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api } from './client'

describe('api envelope', () => {
  afterEach(() => vi.restoreAllMocks())

  it('unwraps successful data', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ data: { value: 7 }, error: null }), { status: 200 }))
    await expect(api.get<{ value: number }>('/math/modules')).resolves.toEqual({ value: 7 })
  })

  it('preserves server error status and code', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ data: null, error: { code: 'child_not_found', message: '孩子不存在' } }), { status: 404 }))
    const error = await api.get('/children/9/math/home').catch((value) => value)
    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({ status: 404, code: 'child_not_found', message: '孩子不存在' })
  })
})
