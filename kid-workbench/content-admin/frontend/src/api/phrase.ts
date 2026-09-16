import { appPath } from '../appPath'
import type { PhraseListResult } from './phraseTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function listPhrase(params: { view: 'groups' | 'table' }): Promise<PhraseListResult> {
  const q = new URLSearchParams({ view: params.view })
  const res = await fetch(appPath(`/api/v1/phrase/items?${q}`))
  return parseJSON(res)
}
