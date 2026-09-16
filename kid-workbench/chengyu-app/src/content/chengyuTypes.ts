import type { ChengyuCode } from '../api/types'

export const chengyuTypes = [
  { code: 'meaning', title: '听释义' },
  { code: 'pick', title: '选成语' },
  { code: 'pinyin', title: '看拼音' },
  { code: 'example', title: '看句子' },
] as const

export const chengyuTypeNames: Record<ChengyuCode, string> = {
  meaning: '听释义',
  pick: '选成语',
  pinyin: '看拼音',
  example: '看句子',
}

export function isChengyuCode(value: string | undefined): value is ChengyuCode {
  return chengyuTypes.some((type) => type.code === value)
}

export function blankExample(example: string, chengyu: string) {
  const word = chengyu.trim()
  if (!word || !example.includes(word)) return example
  return example.replaceAll(word, '____')
}
