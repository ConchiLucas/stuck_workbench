import { appPath } from '../appPath'
import type { LogicListResult } from './logicTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function listLogic(params: { view: 'groups' | 'table' }): Promise<LogicListResult> {
  const q = new URLSearchParams({ view: params.view })
  const res = await fetch(appPath(`/api/v1/logic/items?${q}`))
  return parseJSON(res)
}
