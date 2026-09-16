import { appPath } from '../appPath'
import type {
  ScienceItem,
  ScienceListResult,
  ScienceSyncResult,
} from './scienceTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const messages: Record<string, string> = {
      review_required: '请先提交审核，再发布内容。',
      no_published_question: '缺少科普选择题，请先重新生成题库。',
      invalid_review_state: '当前状态不能执行这个操作，请刷新后重试。',
    }
    const msg = messages[body?.code] ?? (typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`)
    throw new Error(msg)
  }
  return body as T
}

export async function patchScienceContent(
  kpId: number,
  content: Pick<ScienceItem, 'summary' | 'explanation' | 'funFact'>,
): Promise<ScienceItem> {
  const res = await fetch(appPath(`/api/v1/science/items/${kpId}/content`), {
    method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(content),
  })
  return parseJSON(res)
}

async function transition(kpId: number, action: 'review' | 'publish' | 'unpublish'): Promise<ScienceItem> {
  const res = await fetch(appPath(`/api/v1/science/items/${kpId}/${action}`), { method: 'POST' })
  return parseJSON(res)
}

export const reviewScienceItem = (kpId: number) => transition(kpId, 'review')
export const publishScienceItem = (kpId: number) => transition(kpId, 'publish')
export const unpublishScienceItem = (kpId: number) => transition(kpId, 'unpublish')

export async function syncScience(): Promise<ScienceSyncResult> {
  const res = await fetch(appPath('/api/v1/science/sync'), { method: 'POST' })
  return parseJSON(res)
}

export async function listScience(params: {
  view: 'groups' | 'table'
  needsSenseImage?: '' | 'true' | 'false'
}): Promise<ScienceListResult> {
  const q = new URLSearchParams({ view: params.view })
  if (params.needsSenseImage) q.set('needsSenseImage', params.needsSenseImage)
  const res = await fetch(appPath(`/api/v1/science/items?${q}`))
  return parseJSON(res)
}

export function speechAudioURL(kpId: number, speechAudioUrl?: string): string {
  const base = appPath(`/api/v1/science/items/${kpId}/speech.mp3`)
  if (!speechAudioUrl) return base
  try {
    const u = new URL(speechAudioUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function patchScienceItem(
  kpId: number,
  needsSenseImageOverride: boolean | null,
): Promise<ScienceItem> {
  const res = await fetch(appPath(`/api/v1/science/items/${kpId}`), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ needsSenseImageOverride }),
  })
  return parseJSON(res)
}

export async function batchGenerateGlyphs(): Promise<{
  generated: number
  skipped: number
  failed: number
  errors?: string[]
}> {
  const res = await fetch(appPath('/api/v1/science/glyphs/batch'), { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateSenses(
  moduleCode?: string,
  opts?: { workers?: number; maxRetries?: number },
): Promise<{
  generated: number
  skipped: number
  failed: number
  retried?: number
  workers?: number
  errors?: string[]
}> {
  const q = new URLSearchParams()
  if (moduleCode) q.set('moduleCode', moduleCode)
  if (opts?.workers) q.set('workers', String(opts.workers))
  if (opts?.maxRetries != null) q.set('maxRetries', String(opts.maxRetries))
  const res = await fetch(appPath(`/api/v1/science/senses/batch?${q}`), { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateSpeech(moduleCode: string): Promise<{
  generated: number
  skipped: number
  failed: number
  errors?: string[]
}> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(appPath(`/api/v1/science/speech/batch?${q}`), { method: 'POST' })
  return parseJSON(res)
}
