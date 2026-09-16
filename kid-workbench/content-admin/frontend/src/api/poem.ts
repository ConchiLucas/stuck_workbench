import { appPath } from '../appPath'
import type { PoemListResult, PoemSyncResult } from './poemTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function syncPoem(): Promise<PoemSyncResult> {
  const res = await fetch(appPath('/api/v1/poem/sync'), { method: 'POST' })
  return parseJSON(res)
}

export async function listPoem(params: { view: 'groups' | 'table' }): Promise<PoemListResult> {
  const q = new URLSearchParams({ view: params.view })
  const res = await fetch(appPath(`/api/v1/poem/items?${q}`))
  return parseJSON(res)
}
