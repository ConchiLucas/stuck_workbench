import { appPath } from '../appPath'
import type {
  PinyinBatchResult,
  PinyinItem,
  PinyinListResult,
  PinyinSyncResult,
  PinyinGeneratedQuiz,
  PinyinGeneratedQuizType,
} from './pinyinTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function syncPinyin(): Promise<PinyinSyncResult> {
  const res = await fetch(appPath('/api/v1/pinyin/sync'), { method: 'POST' })
  return parseJSON(res)
}

export async function generatePinyinQuiz(
  type: PinyinGeneratedQuizType,
  excludeTargetIds: number[],
): Promise<PinyinGeneratedQuiz> {
  const res = await fetch(appPath('/api/v1/pinyin/quiz/generate'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ type, excludeTargetIds }),
  })
  return parseJSON(res)
}

export async function generatePinyinQuizSet(
  type: PinyinGeneratedQuizType,
  count = 4,
): Promise<PinyinGeneratedQuiz[]> {
  const questions: PinyinGeneratedQuiz[] = []
  const excludeTargetIds: number[] = []
  for (let i = 0; i < count; i++) {
    const question = await generatePinyinQuiz(type, excludeTargetIds)
    questions.push(question)
    excludeTargetIds.push(question.targetId)
  }
  return questions
}

export async function listPinyin(params: {
  view: 'groups' | 'table'
}): Promise<PinyinListResult> {
  const q = new URLSearchParams({ view: params.view })
  const res = await fetch(appPath(`/api/v1/pinyin/items?${q}`))
  return parseJSON(res)
}

export function speechAudioURL(
  kpId: number,
  kind: string,
  speechUrl?: string,
): string {
  const base = appPath(`/api/v1/pinyin/items/${kpId}/speech/${kind}.mp3`)
  if (!speechUrl) return base
  try {
    const u = new URL(speechUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function regenerateSpeech(
  kpId: number,
  kind: string,
): Promise<PinyinItem> {
  const res = await fetch(appPath(`/api/v1/pinyin/items/${kpId}/speech/${kind}`), { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateSpeech(moduleCode: string): Promise<PinyinBatchResult> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(appPath(`/api/v1/pinyin/speech/batch?${q}`), { method: 'POST' })
  return parseJSON(res)
}

export async function importHumanPack(moduleCode: string): Promise<PinyinBatchResult> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(appPath(`/api/v1/pinyin/speech/human-pack?${q}`), { method: 'POST' })
  return parseJSON(res)
}

export function glyphImageURL(kpId: number, glyphUrl?: string): string {
  const base = appPath(`/api/v1/pinyin/items/${kpId}/glyph.png`)
  if (!glyphUrl) return base
  try {
    const u = new URL(glyphUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function generateGlyph(kpId: number): Promise<PinyinItem> {
  const res = await fetch(appPath(`/api/v1/pinyin/items/${kpId}/glyph`), { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateGlyphs(moduleCode: string): Promise<PinyinBatchResult> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(appPath(`/api/v1/pinyin/glyphs/batch?${q}`), { method: 'POST' })
  return parseJSON(res)
}

export type PinyinSyllable = {
  id: number; initialText: string; finalText: string; tone: number
  syllableText: string; speechText: string; speechUrl: string; enabled: boolean
}
export async function listSyllables(): Promise<{ items: PinyinSyllable[]; total: number }> {
 return parseJSON(await fetch(appPath('/api/v1/pinyin/syllables')))
}
export async function updateSyllable(id: number, speechText: string, enabled: boolean): Promise<PinyinSyllable> {
 return parseJSON(await fetch(appPath(`/api/v1/pinyin/syllables/${id}`), {method:'PATCH', headers:{'Content-Type':'application/json'}, body:JSON.stringify({speechText,enabled})}))
}
export async function importSyllableRecordings(): Promise<PinyinBatchResult> {
 return parseJSON(await fetch(appPath('/api/v1/pinyin/syllables/human-pack'), {method:'POST'}))
}
export function syllableAudioURL(item: PinyinSyllable): string {
 const parsed = new URL(item.speechUrl || '/', 'http://localhost')
 return appPath(`/api/v1/pinyin/syllables/${item.id}/speech.mp3${parsed.search}`)
}
