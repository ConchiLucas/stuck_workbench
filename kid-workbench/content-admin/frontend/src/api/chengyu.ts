import { appPath } from '../appPath'
import type { ChengyuListResult } from './chengyuTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function listChengyu(params: { view: 'groups' | 'table' }): Promise<ChengyuListResult> {
  const q = new URLSearchParams({ view: params.view })
  const res = await fetch(appPath(`/api/v1/chengyu/items?${q}`))
  return parseJSON(res)
}
