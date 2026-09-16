import type { PhraseCode } from '../api/types'

export const phraseTypes = [
  { code: 'listen_zh', title: '听一听' },
  { code: 'listen_en', title: '选句子' },
  { code: 'scene', title: '什么时候说' },
  { code: 'reply', title: '问与答' },
] as const

export const phraseTypeNames: Record<PhraseCode, string> = {
  listen_zh: '听一听',
  listen_en: '选句子',
  scene: '什么时候说',
  reply: '问与答',
}

export function isPhraseCode(value: string | undefined): value is PhraseCode {
  return phraseTypes.some((type) => type.code === value)
}
