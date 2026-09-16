import type { PinyinQuestionType } from './types'

const stemByType: Record<PinyinQuestionType, string> = {
  listen: '听一听，选出你听到的拼音',
  inword: '听一听，这个字里藏着哪个拼音？',
  shape: '看一看，选出四线格里拼音的读音',
  blend: '把声母和韵母拼在一起',
}

export function practiceStem(type: string, stem?: string) {
  if (stem && /[\u4e00-\u9fff]/.test(stem)) return stem
  return stemByType[type as PinyinQuestionType] ?? stem ?? ''
}

export function optionCharCount(label: string | undefined | null) {
  return Math.max(1, Array.from(label ?? '').length)
}
